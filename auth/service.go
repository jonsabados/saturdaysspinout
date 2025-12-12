package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
)

type Result struct {
	Token     string
	ExpiresAt time.Time
	UserID    int64
	UserName  string
}

type OAuthClient interface {
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*iracing.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*iracing.TokenResponse, error)
}

type UserInfoProvider interface {
	GetUserInfo(ctx context.Context, accessToken string) (*iracing.UserInfo, error)
}

type JWTCreator interface {
	CreateToken(ctx context.Context, userID int64, userName string, accessToken, refreshToken string, tokenExpiry time.Time) (string, error)
}

type DriverStore interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
	InsertDriver(ctx context.Context, driver store.Driver) error
	RecordLogin(ctx context.Context, driverID int64, loginTime time.Time) error
}

type Service struct {
	oauthClient      OAuthClient
	jwtCreator       JWTCreator
	userInfoProvider UserInfoProvider
	driverStore      DriverStore
	now              func() time.Time
}

func NewService(oauthClient OAuthClient, jwtCreator JWTCreator, userInfoProvider UserInfoProvider, driverStore DriverStore) *Service {
	return &Service{
		oauthClient:      oauthClient,
		jwtCreator:       jwtCreator,
		userInfoProvider: userInfoProvider,
		driverStore:      driverStore,
		now:              time.Now,
	}
}

// HandleRefresh refreshes the iRacing tokens and issues a new JWT
func (s *Service) HandleRefresh(ctx context.Context, userID int64, userName string, refreshToken string) (*Result, error) {
	tokenResp, err := s.oauthClient.RefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("refreshing iRacing token: %w", err)
	}

	tokenExpiry := s.now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	jwt, err := s.jwtCreator.CreateToken(ctx, userID, userName, tokenResp.AccessToken, tokenResp.RefreshToken, tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("creating JWT: %w", err)
	}

	return &Result{
		Token:     jwt,
		ExpiresAt: tokenExpiry,
		UserID:    userID,
		UserName:  userName,
	}, nil
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

	driverRecord, err := s.driverStore.GetDriver(ctx, userInfo.UserID)
	if err != nil {
		return nil, fmt.Errorf("getting driver record: %w", err)
	}
	if driverRecord == nil {
		now := s.now()
		err := s.driverStore.InsertDriver(ctx, store.Driver{
			DriverID:   userInfo.UserID,
			DriverName: userInfo.UserName,
			FirstLogin: now,
			LastLogin:  now,
			LoginCount: 1,
		})
		if err != nil {
			return nil, fmt.Errorf("creating driver: %w", err)
		}
	} else {
		err := s.driverStore.RecordLogin(ctx, userInfo.UserID, s.now())
		if err != nil {
			return nil, fmt.Errorf("recording login: %w", err)
		}
	}

	tokenExpiry := s.now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
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
