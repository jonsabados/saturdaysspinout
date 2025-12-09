package iracing

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	tokenURL = "https://oauth.iracing.com/oauth2/token"
)

// TokenResponse represents the response from iRacing's token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// TokenExpiry calculates the token expiry time from the ExpiresIn value
func (t *TokenResponse) TokenExpiry() time.Time {
	return time.Now().Add(time.Duration(t.ExpiresIn) * time.Second)
}

// OAuthClient handles OAuth operations with iRacing
type OAuthClient interface {
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*TokenResponse, error)
}

// HTTPClient is the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// oauthClient implements OAuthClient
type oauthClient struct {
	httpClient   HTTPClient
	clientID     string
	clientSecret string
}

// NewOAuthClient creates a new iRacing OAuth client
func NewOAuthClient(httpClient HTTPClient, clientID, clientSecret string) *oauthClient {
	return &oauthClient{
		httpClient:   httpClient,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

// ExchangeCode exchanges an authorization code for tokens
func (c *oauthClient) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*TokenResponse, error) {
	maskedSecret := c.maskSecret()

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", maskedSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	return &tokenResp, nil
}

// maskSecret creates the masked client secret per iRacing's requirements:
// base64(sha256(secret + lowercase(trim(client_id))))
func (c *oauthClient) maskSecret() string {
	input := c.clientSecret + strings.ToLower(strings.TrimSpace(c.clientID))
	hash := sha256.Sum256([]byte(input))
	return base64.StdEncoding.EncodeToString(hash[:])
}