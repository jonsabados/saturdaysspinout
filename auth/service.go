package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
)

type Result struct {
	Token     string
	ExpiresAt time.Time
	UserID    int
	UserName  string
}

type OAuthClient interface {
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*iracing.TokenResponse, error)
}

type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, accessToken string) (*iracing.UserInfo, error)
}

type JWTCreator interface {
	CreateToken(ctx context.Context, userID int, userName string, accessToken, refreshToken string, tokenExpiry time.Time) (string, error)
}

type Service struct {
	oauthClient      OAuthClient
	jwtCreator       JWTCreator
	userInfoProvider UserInfoProvider
}

func NewService(oauthClient OAuthClient, jwtCreator JWTCreator, userInfoProvider UserInfoProvider) *Service {
	return &Service{
		oauthClient:      oauthClient,
		jwtCreator:       jwtCreator,
		userInfoProvider: userInfoProvider,
	}
}

// HandleCallback processes an OAuth callback from iRacing after a user has authenticated is returning to our site
func (s *Service) HandleCallback(ctx context.Context, code, codeVerifier, redirectURI string) (*Result, error) {
	tokenResp, err := s.oauthClient.ExchangeCode(ctx, code, codeVerifier, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}

	userInfo, err := s.userInfoProvider.GetUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("getting user info: %w", err)
	}

	tokenExpiry := tokenResp.TokenExpiry()
	jwt, err := s.jwtCreator.CreateToken(ctx, userInfo.UserID, userInfo.UserName, tokenResp.AccessToken, tokenResp.RefreshToken, tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("creating JWT: %w", err)
	}

	return &Result{
		Token:     jwt,
		ExpiresAt: tokenExpiry,
		UserID:    userInfo.UserID,
		UserName:  userInfo.UserName,
	}, nil
}
