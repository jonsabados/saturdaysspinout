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

	"github.com/rs/zerolog"
)

// DataAPIBaseURL is the default base URL for the iRacing data API
const DataAPIBaseURL = "https://members-ng.iracing.com"

// iRacingTimeFormat is ISO-8601 with minute precision used by iRacing API
const iRacingTimeFormat = "2006-01-02T15:04Z"

// UserInfo contains basic info about an iRacing user
type UserInfo struct {
	UserID      int64
	UserName    string
	MemberSince time.Time
}

type Client struct {
	httpClient HTTPClient
	baseURL    string
}

type ClientOption func(*Client)

func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

func NewClient(httpClient HTTPClient, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: httpClient,
		baseURL:    DataAPIBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// linkResponse represents the initial response from iRacing API endpoints that return a signed S3 URL to fetch the actual data
type linkResponse struct {
	Link string `json:"link"`
}

// chunkInfo represents the chunked download info returned by search endpoints
type chunkInfo struct {
	ChunkSize       int      `json:"chunk_size"`
	NumChunks       int      `json:"num_chunks"`
	Rows            int      `json:"rows"`
	BaseDownloadURL string   `json:"base_download_url"`
	ChunkFileNames  []string `json:"chunk_file_names"`
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

type eventTypesOption []int

func (e eventTypesOption) applySearch(params url.Values) {
	if len(e) == 0 {
		return
	}
	strs := make([]string, len(e))
	for i, t := range e {
		strs[i] = strconv.Itoa(t)
	}
	params.Set("event_types", strings.Join(strs, ","))
}

func WithEventTypes(types ...int) SearchOption {
	return eventTypesOption(types)
}

// GetUserInfo retrieves the current user's info from iRacing
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	logger := zerolog.Ctx(ctx)
	endpoint := c.baseURL + "/data/member/info"

	// First request: get the signed S3 URL
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

	logger.Trace().RawJSON("response", body).Int("status", resp.StatusCode).Msg("received link response from /data/member/info")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user info failed with status %d: %s", resp.StatusCode, string(body))
	}

	var linkResp linkResponse
	if err := json.Unmarshal(body, &linkResp); err != nil {
		return nil, fmt.Errorf("parsing link response: %w", err)
	}

	if linkResp.Link == "" {
		return nil, fmt.Errorf("no link in response")
	}

	// Second request: fetch the actual data from S3
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

	logger.Trace().RawJSON("response", dataBody).Int("status", dataResp.StatusCode).Msg("received member info from S3")

	if dataResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get user data failed with status %d: %s", dataResp.StatusCode, string(dataBody))
	}

	var apiResp struct {
		CustID      int64    `json:"cust_id"`
		DisplayName string   `json:"display_name"`
		MemberSince dateOnly `json:"member_since"`
	}
	if err := json.Unmarshal(dataBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing user info response: %w", err)
	}

	return &UserInfo{
		UserID:      apiResp.CustID,
		UserName:    apiResp.DisplayName,
		MemberSince: apiResp.MemberSince.Time(),
	}, nil
}

func (c *Client) SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...SearchOption) ([]SessionResult, error) {
	logger := zerolog.Ctx(ctx)

	params := url.Values{}
	params.Set("finish_range_begin", finishRangeBegin.UTC().Format(iRacingTimeFormat))
	params.Set("finish_range_end", finishRangeEnd.UTC().Format(iRacingTimeFormat))
	for _, opt := range opts {
		opt.applySearch(params)
	}

	endpoint := c.baseURL + "/data/results/search_series?" + params.Encode()

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

	logger.Trace().RawJSON("response", body).Int("status", resp.StatusCode).Msg("received response from /data/results/search_series")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search series results failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp searchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	if !searchResp.Data.Success {
		return nil, fmt.Errorf("search was not successful")
	}

	chunkInfo := searchResp.Data.ChunkInfo
	if chunkInfo.Rows == 0 {
		return []SessionResult{}, nil
	}

	logger.Debug().
		Int("totalChunks", len(chunkInfo.ChunkFileNames)).
		Int("totalRows", chunkInfo.Rows).
		Msg("fetching result chunks")

	var allResults []SessionResult
	for i, chunkFileName := range chunkInfo.ChunkFileNames {
		chunkURL := chunkInfo.BaseDownloadURL + chunkFileName

		chunkReq, err := http.NewRequestWithContext(ctx, http.MethodGet, chunkURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating chunk request: %w", err)
		}

		chunkResp, err := c.httpClient.Do(chunkReq)
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

		var chunkResults []SessionResult
		if err := json.Unmarshal(chunkBody, &chunkResults); err != nil {
			return nil, fmt.Errorf("parsing chunk %d: %w", i, err)
		}

		allResults = append(allResults, chunkResults...)
	}

	logger.Debug().Int("resultsCount", len(allResults)).Msg("fetched all session results")

	return allResults, nil
}
