package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_HandleCallback(t *testing.T) {
	type oauthClientCall struct {
		inputCode         string
		inputCodeVerifier string
		inputRedirectURI  string
		result            *iracing.TokenResponse
		err               error
	}

	type userInfoProviderCall struct {
		inputAccessToken string
		result           *iracing.UserInfo
		err              error
	}

	type getDriverCall struct {
		inputDriverID int64
		result        *store.Driver
		err           error
	}

	type insertDriverCall struct {
		expectedDriver store.Driver
		err            error
	}

	type recordLoginCall struct {
		expectedDriverID  int64
		expectedLoginTime time.Time
		err               error
	}

	type jwtCreatorCall struct {
		inputUserID       int64
		inputUserName     string
		inputAccessToken  string
		inputRefreshToken string
		inputTokenExpiry  time.Time
		result            string
		err               error
	}

	fixedNow := time.Unix(5000, 0)
	expectedTokenExpiry := fixedNow.Add(time.Hour) // ExpiresIn is 3600 seconds

	testCases := []struct {
		name string

		inputCode         string
		inputCodeVerifier string
		inputRedirectURI  string

		oauthClientCalls      []oauthClientCall
		userInfoProviderCalls []userInfoProviderCall
		getDriverCalls        []getDriverCall
		insertDriverCalls     []insertDriverCall
		recordLoginCalls      []recordLoginCall
		jwtCreatorCalls       []jwtCreatorCall

		expectedResult *Result
		expectedErr    string
	}{
		{
			name:              "success - new driver",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					result: &iracing.UserInfo{
						UserID:   12345,
						UserName: "Test Driver",
					},
				},
			},
			getDriverCalls: []getDriverCall{
				{inputDriverID: 12345, result: nil},
			},
			insertDriverCalls: []insertDriverCall{
				{expectedDriver: store.Driver{
					DriverID:   12345,
					DriverName: "Test Driver",
					FirstLogin: fixedNow,
					LastLogin:  fixedNow,
					LoginCount: 1,
				}},
			},
			jwtCreatorCalls: []jwtCreatorCall{
				{
					inputUserID:       12345,
					inputUserName:     "Test Driver",
					inputAccessToken:  "access-token",
					inputRefreshToken: "refresh-token",
					inputTokenExpiry:  expectedTokenExpiry,
					result:            "jwt-token",
				},
			},
			expectedResult: &Result{
				Token:    "jwt-token",
				UserID:   12345,
				UserName: "Test Driver",
			},
		},
		{
			name:              "success - existing driver",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					result: &iracing.UserInfo{
						UserID:   12345,
						UserName: "Test Driver",
					},
				},
			},
			getDriverCalls: []getDriverCall{
				{
					inputDriverID: 12345,
					result: &store.Driver{
						DriverID:   12345,
						DriverName: "Test Driver",
						FirstLogin: time.Unix(1000, 0),
						LastLogin:  time.Unix(2000, 0),
						LoginCount: 5,
					},
				},
			},
			recordLoginCalls: []recordLoginCall{
				{expectedDriverID: 12345, expectedLoginTime: fixedNow},
			},
			jwtCreatorCalls: []jwtCreatorCall{
				{
					inputUserID:       12345,
					inputUserName:     "Test Driver",
					inputAccessToken:  "access-token",
					inputRefreshToken: "refresh-token",
					inputTokenExpiry:  expectedTokenExpiry,
					result:            "jwt-token",
				},
			},
			expectedResult: &Result{
				Token:    "jwt-token",
				UserID:   12345,
				UserName: "Test Driver",
			},
		},
		{
			name:              "oauth exchange fails",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					err:               errors.New("oauth error"),
				},
			},
			expectedErr: "exchanging authorization code: oauth error",
		},
		{
			name:              "get user info fails",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					err:              errors.New("user info error"),
				},
			},
			expectedErr: "getting user info: user info error",
		},
		{
			name:              "get driver fails",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					result: &iracing.UserInfo{
						UserID:   12345,
						UserName: "Test Driver",
					},
				},
			},
			getDriverCalls: []getDriverCall{
				{inputDriverID: 12345, err: errors.New("db error")},
			},
			expectedErr: "getting driver record: db error",
		},
		{
			name:              "insert driver fails",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					result: &iracing.UserInfo{
						UserID:   12345,
						UserName: "Test Driver",
					},
				},
			},
			getDriverCalls: []getDriverCall{
				{inputDriverID: 12345, result: nil},
			},
			insertDriverCalls: []insertDriverCall{
				{expectedDriver: store.Driver{
					DriverID:   12345,
					DriverName: "Test Driver",
					FirstLogin: fixedNow,
					LastLogin:  fixedNow,
					LoginCount: 1,
				}, err: errors.New("insert error")},
			},
			expectedErr: "creating driver: insert error",
		},
		{
			name:              "record login fails",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					result: &iracing.UserInfo{
						UserID:   12345,
						UserName: "Test Driver",
					},
				},
			},
			getDriverCalls: []getDriverCall{
				{
					inputDriverID: 12345,
					result: &store.Driver{
						DriverID:   12345,
						DriverName: "Test Driver",
						LoginCount: 1,
					},
				},
			},
			recordLoginCalls: []recordLoginCall{
				{expectedDriverID: 12345, expectedLoginTime: fixedNow, err: errors.New("record login error")},
			},
			expectedErr: "recording login: record login error",
		},
		{
			name:              "create token fails",
			inputCode:         "auth-code",
			inputCodeVerifier: "code-verifier",
			inputRedirectURI:  "http://localhost/callback",
			oauthClientCalls: []oauthClientCall{
				{
					inputCode:         "auth-code",
					inputCodeVerifier: "code-verifier",
					inputRedirectURI:  "http://localhost/callback",
					result: &iracing.TokenResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    3600,
					},
				},
			},
			userInfoProviderCalls: []userInfoProviderCall{
				{
					inputAccessToken: "access-token",
					result: &iracing.UserInfo{
						UserID:   12345,
						UserName: "Test Driver",
					},
				},
			},
			getDriverCalls: []getDriverCall{
				{inputDriverID: 12345, result: nil},
			},
			insertDriverCalls: []insertDriverCall{
				{expectedDriver: store.Driver{
					DriverID:   12345,
					DriverName: "Test Driver",
					FirstLogin: fixedNow,
					LastLogin:  fixedNow,
					LoginCount: 1,
				}},
			},
			jwtCreatorCalls: []jwtCreatorCall{
				{
					inputUserID:       12345,
					inputUserName:     "Test Driver",
					inputAccessToken:  "access-token",
					inputRefreshToken: "refresh-token",
					inputTokenExpiry:  expectedTokenExpiry,
					err:               errors.New("jwt error"),
				},
			},
			expectedErr: "creating JWT: jwt error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			oauthClient := NewMockOAuthClient(t)
			for _, call := range tc.oauthClientCalls {
				oauthClient.EXPECT().ExchangeCode(mock.Anything, call.inputCode, call.inputCodeVerifier, call.inputRedirectURI).Return(call.result, call.err)
			}

			userInfoProvider := NewMockUserInfoProvider(t)
			for _, call := range tc.userInfoProviderCalls {
				userInfoProvider.EXPECT().GetUserInfo(mock.Anything, call.inputAccessToken).Return(call.result, call.err)
			}

			driverStore := NewMockDriverStore(t)
			for _, call := range tc.getDriverCalls {
				driverStore.EXPECT().GetDriver(mock.Anything, call.inputDriverID).Return(call.result, call.err)
			}
			for _, call := range tc.insertDriverCalls {
				driverStore.EXPECT().InsertDriver(mock.Anything, call.expectedDriver).Return(call.err)
			}
			for _, call := range tc.recordLoginCalls {
				driverStore.EXPECT().RecordLogin(mock.Anything, call.expectedDriverID, call.expectedLoginTime).Return(call.err)
			}

			jwtCreator := NewMockJWTCreator(t)
			for _, call := range tc.jwtCreatorCalls {
				jwtCreator.EXPECT().CreateToken(mock.Anything, call.inputUserID, call.inputUserName, call.inputAccessToken, call.inputRefreshToken, call.inputTokenExpiry).Return(call.result, call.err)
			}

			service := NewService(oauthClient, jwtCreator, userInfoProvider, driverStore)
			service.now = func() time.Time { return fixedNow }

			result, err := service.HandleCallback(ctx, tc.inputCode, tc.inputCodeVerifier, tc.inputRedirectURI)

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tc.expectedResult.Token, result.Token)
				assert.Equal(t, tc.expectedResult.UserID, result.UserID)
				assert.Equal(t, tc.expectedResult.UserName, result.UserName)
			}
		})
	}
}
