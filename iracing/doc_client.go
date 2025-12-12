package iracing

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

const DocAPIBaseURL = "https://members-ng.iracing.com/data/doc"

type DocClient struct {
	httpClient HTTPClient
	baseURL    string
}

type DocClientOption func(*DocClient)

func WithDocBaseURL(url string) DocClientOption {
	return func(c *DocClient) {
		c.baseURL = url
	}
}

func NewDocClient(httpClient HTTPClient, opts ...DocClientOption) *DocClient {
	c := &DocClient{
		httpClient: httpClient,
		baseURL:    DocAPIBaseURL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *DocClient) Fetch(ctx context.Context, accessToken string, path string) ([]byte, string, error) {
	endpoint := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	zerolog.Ctx(ctx).Trace().Str("endpoint", endpoint).Int("status", resp.StatusCode).Msg("made request to iracing doc endpoint")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reading response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, "", fmt.Errorf("%w: %s", ErrUpstreamUnauthorized, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, contentType, nil
}
