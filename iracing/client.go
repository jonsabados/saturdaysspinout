package iracing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

// DataAPIBaseURL is the default base URL for the iRacing data API
const DataAPIBaseURL = "https://members-ng.iracing.com"

// UserInfo contains basic info about an iRacing user
type UserInfo struct {
	UserID   int64
	UserName string
}

// Client is the interface for the iRacing data API
type Client interface {
	GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error)
}

// client implements Client
type client struct {
	httpClient HTTPClient
	baseURL    string
}

// ClientOption is a function that configures a client
type ClientOption func(*client)

// WithBaseURL sets the base URL for the iRacing data API
func WithBaseURL(url string) ClientOption {
	return func(c *client) {
		c.baseURL = url
	}
}

// NewClient creates a new iRacing API client
func NewClient(httpClient HTTPClient, opts ...ClientOption) *client {
	c := &client{
		httpClient: httpClient,
		baseURL:    DataAPIBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// linkResponse represents the initial response from iRacing API endpoints
// that return a signed S3 URL to fetch the actual data
type linkResponse struct {
	Link string `json:"link"`
}

// GetUserInfo retrieves the current user's info from iRacing
func (c *client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
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
		CustID      int64  `json:"cust_id"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(dataBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing user info response: %w", err)
	}

	return &UserInfo{
		UserID:   apiResp.CustID,
		UserName: apiResp.DisplayName,
	}, nil
}
