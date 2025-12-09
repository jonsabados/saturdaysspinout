package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
)

// AuthResult contains the result of a successful authentication
type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	UserID    int
	UserName  string
}

// OAuthClient is the interface for OAuth token exchange
type OAuthClient interface {
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*iracing.TokenResponse, error)
}

// UserInfoProvider extracts user info from tokens
type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, accessToken string) (*iracing.UserInfo, error)
}

// JWTCreator is the interface for creating JWTs
type JWTCreator interface {
	CreateToken(ctx context.Context, userID int, userName string, accessToken, refreshToken string, tokenExpiry time.Time) (string, error)
}

// Service handles authentication operations
type Service struct {
	oauthClient      OAuthClient
	jwtCreator       JWTCreator
	userInfoProvider UserInfoProvider
}

// NewService creates a new authentication service
func NewService(oauthClient OAuthClient, jwtCreator JWTCreator, userInfoProvider UserInfoProvider) *Service {
	return &Service{
		oauthClient:      oauthClient,
		jwtCreator:       jwtCreator,
		userInfoProvider: userInfoProvider,
	}
}

// HandleCallback processes an OAuth callback and returns an authenticated session
func (s *Service) HandleCallback(ctx context.Context, code, codeVerifier, redirectURI string) (*AuthResult, error) {
	// Exchange the authorization code for tokens
	tokenResp, err := s.oauthClient.ExchangeCode(ctx, code, codeVerifier, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}

	// Get user info from iRacing
	userInfo, err := s.userInfoProvider.GetUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("getting user info: %w", err)
	}

	// Create JWT with encrypted iRacing tokens
	tokenExpiry := tokenResp.TokenExpiry()
	jwt, err := s.jwtCreator.CreateToken(ctx, userInfo.UserID, userInfo.UserName, tokenResp.AccessToken, tokenResp.RefreshToken, tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("creating JWT: %w", err)
	}

	return &AuthResult{
		Token:     jwt,
		ExpiresAt: tokenExpiry,
		UserID:    userInfo.UserID,
		UserName:  userInfo.UserName,
	}, nil
}