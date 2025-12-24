package iracing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jonsabados/saturdaysspinout/metrics"
	"github.com/rs/zerolog"
)

// DataAPIBaseURL is the default base URL for the iRacing data API
const DataAPIBaseURL = "https://members-ng.iracing.com"

// iRacingTimeFormat is ISO-8601 with minute precision used by iRacing API
const iRacingTimeFormat = "2006-01-02T15:04Z"

type MetricsClient interface {
	EmitGauge(ctx context.Context, name string, value float64) error
}

// UserInfo contains basic info about an iRacing user
type UserInfo struct {
	UserID      int64
	UserName    string
	MemberSince time.Time
}

type Client struct {
	httpClient    HTTPClient
	metricsClient MetricsClient
	baseURL       string
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

func NewClient(httpClient HTTPClient, metricsClient MetricsClient, opts ...ClientOption) *Client {
	c := &Client{
		httpClient:    httpClient,
		metricsClient: metricsClient,
		baseURL:       DataAPIBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// logRateLimitHeaders logs the rate limit headers from an iRacing API response.
// Logs at Trace level normally, but warns when remaining drops below 20% of limit.
// Also emits a metric for the remaining rate limit.
func (c *Client) logRateLimitHeaders(ctx context.Context, logger *zerolog.Logger, resp *http.Response) {
	limitStr := resp.Header.Get("x-ratelimit-limit")
	remainingStr := resp.Header.Get("x-ratelimit-remaining")
	resetStr := resp.Header.Get("x-ratelimit-reset")

	if limitStr == "" && remainingStr == "" && resetStr == "" {
		return
	}

	limit, _ := strconv.Atoi(limitStr)
	remaining, _ := strconv.Atoi(remainingStr)

	if err := c.metricsClient.EmitGauge(ctx, metrics.IRacingRateLimitRemaining, float64(remaining)); err != nil {
		logger.Warn().Err(err).Msg("failed to emit rate limit metric")
	}

	// Warn if remaining is below 20% of limit
	isLow := limit > 0 && remaining < limit/5

	var event *zerolog.Event
	if isLow {
		event = logger.Warn()
	} else {
		event = logger.Trace()
	}

	event.
		Str("ratelimit_limit", limitStr).
		Str("ratelimit_remaining", remainingStr).
		Str("ratelimit_reset", resetStr)

	if resetEpoch, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
		resetTime := time.Unix(resetEpoch, 0)
		event.Time("ratelimit_reset_time", resetTime).
			Dur("ratelimit_reset_in", time.Until(resetTime))
	}

	if isLow {
		event.Msg("iRacing API rate limit running low")
	} else {
		event.Msg("iRacing API rate limit status")
	}
}

// linkResponse represents the initial response from iRacing API endpoints that return a signed S3 URL to fetch the actual data
type linkResponse struct {
	Link string `json:"link"`
}

// doAPIRequest makes an authenticated request to an iRacing API endpoint.
// Handles 401 responses by returning ErrUpstreamUnauthorized.
func (c *Client) doAPIRequest(ctx context.Context, accessToken, endpoint string) ([]byte, error) {
	logger := zerolog.Ctx(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	c.logRateLimitHeaders(ctx, logger, resp)

	logger.Trace().RawJSON("response", body).Int("status", resp.StatusCode).Str("endpoint", endpoint).Msg("received API response")

	if resp.StatusCode == http.StatusUnauthorized {
		zerolog.Ctx(ctx).Warn().Str("body", string(body)).Msg("401 received from iRacing API")
		return nil, ErrUpstreamUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// fetchLinkedData fetches data from an iRacing API endpoint that returns a signed S3 URL.
// It makes the initial API request, parses the link response, and fetches the actual data from S3.
func (c *Client) fetchLinkedData(ctx context.Context, accessToken, endpoint string) ([]byte, error) {
	logger := zerolog.Ctx(ctx)

	body, err := c.doAPIRequest(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	var linkResp linkResponse
	if err := json.Unmarshal(body, &linkResp); err != nil {
		return nil, fmt.Errorf("parsing link response: %w", err)
	}

	if linkResp.Link == "" {
		return nil, fmt.Errorf("no link in response")
	}

	dataReq, err := http.NewRequestWithContext(ctx, http.MethodGet, linkResp.Link, nil)
	if err != nil {
		return nil, fmt.Errorf("creating data request: %w", err)
	}

	dataResp, err := c.httpClient.Do(dataReq)
	if err != nil {
		return nil, fmt.Errorf("executing data request: %w", err)
	}
	defer dataResp.Body.Close()

	dataBody, err := io.ReadAll(dataResp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading data response body: %w", err)
	}

	logger.Trace().RawJSON("response", dataBody).Int("status", dataResp.StatusCode).Msg("received linked data from S3")

	if dataResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching linked data failed with status %d: %s", dataResp.StatusCode, string(dataBody))
	}

	return dataBody, nil
}

// chunkInfo represents the chunked download info returned by search endpoints
type chunkInfo struct {
	ChunkSize       int      `json:"chunk_size"`
	NumChunks       int      `json:"num_chunks"`
	Rows            int      `json:"rows"`
	BaseDownloadURL string   `json:"base_download_url"`
	ChunkFileNames  []string `json:"chunk_file_names"`
}

// fetchChunks fetches and unmarshals chunked data from S3.
func fetchChunks[T any](ctx context.Context, httpClient HTTPClient, info chunkInfo) ([]T, error) {
	logger := zerolog.Ctx(ctx)

	if info.Rows == 0 {
		return []T{}, nil
	}

	logger.Debug().
		Int("totalChunks", len(info.ChunkFileNames)).
		Int("totalRows", info.Rows).
		Msg("fetching chunks")

	var results []T
	for i, chunkFileName := range info.ChunkFileNames {
		chunkURL := info.BaseDownloadURL + chunkFileName

		chunkReq, err := http.NewRequestWithContext(ctx, http.MethodGet, chunkURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating chunk request: %w", err)
		}

		chunkResp, err := httpClient.Do(chunkReq)
		if err != nil {
			return nil, fmt.Errorf("executing chunk request: %w", err)
		}

		chunkBody, err := io.ReadAll(chunkResp.Body)
		chunkResp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading chunk response body: %w", err)
		}

		if chunkResp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch chunk %d failed with status %d: %s", i, chunkResp.StatusCode, string(chunkBody))
		}

		var chunkItems []T
		if err := json.Unmarshal(chunkBody, &chunkItems); err != nil {
			return nil, fmt.Errorf("parsing chunk %d: %w", i, err)
		}

		results = append(results, chunkItems...)
	}

	logger.Debug().Int("count", len(results)).Msg("fetched all chunks")

	return results, nil
}

// searchResponse represents the response from search endpoints like /data/results/search_series
type searchResponse struct {
	Type string `json:"type"`
	Data struct {
		Success   bool      `json:"success"`
		ChunkInfo chunkInfo `json:"chunk_info"`
	} `json:"data"`
}

// dateOnly handles unmarshaling date-only strings like "2024-08-07"
type dateOnly time.Time

func (d *dateOnly) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*d = dateOnly(t)
	return nil
}

func (d *dateOnly) Time() time.Time {
	return time.Time(*d)
}

// SearchOption configures optional parameters for SearchSeriesResults
type SearchOption interface {
	applySearch(params url.Values)
}

type customerIDOption int64

func (c customerIDOption) applySearch(params url.Values) {
	params.Set("cust_id", strconv.FormatInt(int64(c), 10))
}

// WithCustomerID filters results to sessions where this customer participated
func WithCustomerID(custID int64) SearchOption {
	return customerIDOption(custID)
}

type eventTypesOption []EventType

func (e eventTypesOption) applySearch(params url.Values) {
	if len(e) == 0 {
		return
	}
	strs := make([]string, len(e))
	for i, t := range e {
		strs[i] = strconv.Itoa(int(t))
	}
	params.Set("event_types", strings.Join(strs, ","))
}

func WithEventTypes(types ...EventType) SearchOption {
	return eventTypesOption(types)
}

// GetUserInfo retrieves the current user's info from iRacing
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	endpoint := c.baseURL + "/data/member/info"

	data, err := c.fetchLinkedData(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		CustID      int64    `json:"cust_id"`
		DisplayName string   `json:"display_name"`
		MemberSince dateOnly `json:"member_since"`
	}
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing user info response: %w", err)
	}

	return &UserInfo{
		UserID:      apiResp.CustID,
		UserName:    apiResp.DisplayName,
		MemberSince: apiResp.MemberSince.Time(),
	}, nil
}

func (c *Client) SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...SearchOption) ([]SeriesResult, error) {
	params := url.Values{}
	params.Set("finish_range_begin", finishRangeBegin.UTC().Format(iRacingTimeFormat))
	params.Set("finish_range_end", finishRangeEnd.UTC().Format(iRacingTimeFormat))
	for _, opt := range opts {
		opt.applySearch(params)
	}

	endpoint := c.baseURL + "/data/results/search_series?" + params.Encode()

	body, err := c.doAPIRequest(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	var searchResp searchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	if !searchResp.Data.Success {
		return nil, fmt.Errorf("search was not successful")
	}

	return fetchChunks[SeriesResult](ctx, c.httpClient, searchResp.Data.ChunkInfo)
}

// GetSessionResultsOption configures optional parameters for GetSessionResults
type GetSessionResultsOption interface {
	applyGetSessionResults(params url.Values)
}

type includeLicensesOption bool

func (i includeLicensesOption) applyGetSessionResults(params url.Values) {
	params.Set("include_licenses", strconv.FormatBool(bool(i)))
}

// WithIncludeLicenses includes license information in the results
func WithIncludeLicenses(include bool) GetSessionResultsOption {
	return includeLicensesOption(include)
}

// GetSessionResults fetches the results of a subsession.
func (c *Client) GetSessionResults(ctx context.Context, accessToken string, subsessionID int64, opts ...GetSessionResultsOption) (*SessionResult, error) {
	params := url.Values{}
	params.Set("subsession_id", strconv.FormatInt(subsessionID, 10))
	for _, opt := range opts {
		opt.applyGetSessionResults(params)
	}

	endpoint := c.baseURL + "/data/results/get?" + params.Encode()

	data, err := c.fetchLinkedData(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	var result SessionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing session results: %w", err)
	}

	return &result, nil
}

// GetLapDataOption configures optional parameters for GetLapData
type GetLapDataOption interface {
	applyGetLapData(params url.Values)
}

type customerIDLapOption int64

func (c customerIDLapOption) applyGetLapData(params url.Values) {
	params.Set("cust_id", strconv.FormatInt(int64(c), 10))
}

// WithCustomerIDLap sets the customer ID for lap data requests.
// Required for single-driver events, optional for team events.
func WithCustomerIDLap(custID int64) GetLapDataOption {
	return customerIDLapOption(custID)
}

type teamIDOption int64

func (t teamIDOption) applyGetLapData(params url.Values) {
	params.Set("team_id", strconv.FormatInt(int64(t), 10))
}

// WithTeamID sets the team ID for lap data requests.
// Required for team events.
func WithTeamID(teamID int64) GetLapDataOption {
	return teamIDOption(teamID)
}

// lapDataAPIResponse is the raw response from the lap data endpoint including chunk info.
type lapDataAPIResponse struct {
	Success         bool               `json:"success"`
	SessionInfo     LapDataSessionInfo `json:"session_info"`
	BestLapNum      int                `json:"best_lap_num"`
	BestLapTime     int                `json:"best_lap_time"`
	BestNLapsNum    int                `json:"best_nlaps_num"`
	BestNLapsTime   int                `json:"best_nlaps_time"`
	BestQualLapNum  int                `json:"best_qual_lap_num"`
	BestQualLapTime int                `json:"best_qual_lap_time"`
	BestQualLapAt   *time.Time         `json:"best_qual_lap_at"`
	ChunkInfo       chunkInfo          `json:"chunk_info"`
	LastUpdated     time.Time          `json:"last_updated"`
	GroupID         int64              `json:"group_id"`
	CustID          int64              `json:"cust_id"`
	Name            string             `json:"name"`
	CarID           int                `json:"car_id"`
	LicenseLevel    int                `json:"license_level"`
	Livery          Livery             `json:"livery"`
}

// GetLapData fetches lap data for a subsession.
func (c *Client) GetLapData(ctx context.Context, accessToken string, subsessionID int64, simsessionNumber int, opts ...GetLapDataOption) (*LapDataResponse, error) {
	params := url.Values{}
	params.Set("subsession_id", strconv.FormatInt(subsessionID, 10))
	params.Set("simsession_number", strconv.Itoa(simsessionNumber))
	for _, opt := range opts {
		opt.applyGetLapData(params)
	}

	endpoint := c.baseURL + "/data/results/lap_data?" + params.Encode()

	body, err := c.fetchLinkedData(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	var apiResp lapDataAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing lap data response: %w", err)
	}

	laps, err := fetchChunks[Lap](ctx, c.httpClient, apiResp.ChunkInfo)
	if err != nil {
		return nil, err
	}

	return &LapDataResponse{
		Success:         apiResp.Success,
		SessionInfo:     apiResp.SessionInfo,
		BestLapNum:      apiResp.BestLapNum,
		BestLapTime:     apiResp.BestLapTime,
		BestNLapsNum:    apiResp.BestNLapsNum,
		BestNLapsTime:   apiResp.BestNLapsTime,
		BestQualLapNum:  apiResp.BestQualLapNum,
		BestQualLapTime: apiResp.BestQualLapTime,
		BestQualLapAt:   apiResp.BestQualLapAt,
		LastUpdated:     apiResp.LastUpdated,
		GroupID:         apiResp.GroupID,
		CustID:          apiResp.CustID,
		Name:            apiResp.Name,
		CarID:           apiResp.CarID,
		LicenseLevel:    apiResp.LicenseLevel,
		Livery:          apiResp.Livery,
		Laps:            laps,
	}, nil
}

// GetTracks fetches all track information from iRacing.
func (c *Client) GetTracks(ctx context.Context, accessToken string) ([]TrackInfo, error) {
	endpoint := c.baseURL + "/data/track/get"

	data, err := c.fetchLinkedData(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	var tracks []TrackInfo
	if err := json.Unmarshal(data, &tracks); err != nil {
		return nil, fmt.Errorf("parsing tracks response: %w", err)
	}

	return tracks, nil
}

// GetTrackAssets fetches track asset information (images, descriptions, maps) from iRacing.
// Returns a map keyed by track ID.
func (c *Client) GetTrackAssets(ctx context.Context, accessToken string) (map[int64]TrackAssets, error) {
	endpoint := c.baseURL + "/data/track/assets"

	data, err := c.fetchLinkedData(ctx, accessToken, endpoint)
	if err != nil {
		return nil, err
	}

	// API returns map with string keys (track IDs as strings)
	var rawAssets map[string]TrackAssets
	if err := json.Unmarshal(data, &rawAssets); err != nil {
		return nil, fmt.Errorf("parsing track assets response: %w", err)
	}

	// Convert to int64 keys for consistency
	assets := make(map[int64]TrackAssets, len(rawAssets))
	for idStr, asset := range rawAssets {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue // Skip invalid IDs
		}
		asset.TrackID = id
		assets[id] = asset
	}

	return assets, nil
}
