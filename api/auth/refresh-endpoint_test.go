package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type stubTokenValidator struct {
	validateFunc func(ctx context.Context, token string) (*auth.SessionClaims, *auth.SensitiveClaims, error)
}

func (s *stubTokenValidator) ValidateToken(ctx context.Context, token string) (*auth.SessionClaims, *auth.SensitiveClaims, error) {
	return s.validateFunc(ctx, token)
}

func TestNewAuthRefreshEndpoint(t *testing.T) {
	type validatorCall struct {
		inputToken      string
		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		err             error
	}

	type authServiceCall struct {
		inputUserID       int64
		inputUserName     string
		inputRefreshToken string
		result            *auth.Result
		resultErr         error
	}

	testCases := []struct {
		name string

		authHeader               string
		expectedValidatorCalls   []validatorCall
		expectedAuthServiceCalls []authServiceCall

		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:       "success",
			authHeader: "Bearer valid-token",
			expectedValidatorCalls: []validatorCall{
				{
					inputToken: "valid-token",
					sessionClaims: &auth.SessionClaims{
						IRacingUserID:   1100750,
						IRacingUserName: "Jon Sabados",
					},
					sensitiveClaims: &auth.SensitiveClaims{
						IRacingRefreshToken: "iracing-refresh-token",
					},
				},
			},
			expectedAuthServiceCalls: []authServiceCall{
				{
					inputUserID:       1100750,
					inputUserName:     "Jon Sabados",
					inputRefreshToken: "iracing-refresh-token",
					result: &auth.Result{
						Token:     "new-jwt-token",
						ExpiresAt: time.Unix(1735689600, 0),
						UserID:    1100750,
						UserName:  "Jon Sabados",
					},
				},
			},
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/auth_refresh_success_response.json",
		},
		{
			name:                        "missing authorization header",
			authHeader:                  "",
			expectedValidatorCalls:      []validatorCall{},
			expectedAuthServiceCalls:    []authServiceCall{},
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/auth_refresh_missing_auth_response.json",
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid-token",
			expectedValidatorCalls: []validatorCall{
				{
					inputToken: "invalid-token",
					err:        errors.New("token validation failed"),
				},
			},
			expectedAuthServiceCalls:    []authServiceCall{},
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/auth_refresh_invalid_token_response.json",
		},
		{
			name:       "auth service error",
			authHeader: "Bearer valid-token",
			expectedValidatorCalls: []validatorCall{
				{
					inputToken: "valid-token",
					sessionClaims: &auth.SessionClaims{
						IRacingUserID:   1100750,
						IRacingUserName: "Jon Sabados",
					},
					sensitiveClaims: &auth.SensitiveClaims{
						IRacingRefreshToken: "iracing-refresh-token",
					},
				},
			},
			expectedAuthServiceCalls: []authServiceCall{
				{
					inputUserID:       1100750,
					inputUserName:     "Jon Sabados",
					inputRefreshToken: "iracing-refresh-token",
					resultErr:         errors.New("refresh failed"),
				},
			},
			expectedResponseStatus:      http.StatusInternalServerError,
			expectedResponseBodyFixture: "fixtures/auth_refresh_service_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			validator := &stubTokenValidator{
				validateFunc: func(ctx context.Context, token string) (*auth.SessionClaims, *auth.SensitiveClaims, error) {
					if len(tc.expectedValidatorCalls) == 0 {
						return nil, nil, fmt.Errorf("unexpected call to ValidateToken")
					}
					call := tc.expectedValidatorCalls[0]
					if token != call.inputToken {
						return nil, nil, fmt.Errorf("unexpected token: got %s, want %s", token, call.inputToken)
					}
					return call.sessionClaims, call.sensitiveClaims, call.err
				},
			}

			authService := NewMockService(t)
			for _, call := range tc.expectedAuthServiceCalls {
				authService.EXPECT().HandleRefresh(mock.Anything, call.inputUserID, call.inputUserName, call.inputRefreshToken).Return(call.result, call.resultErr)
			}

			endpoint := NewAuthRefreshEndpoint(authService)
			handler := correlation.Middleware(func() string { return testCorrelationID })(api.AuthMiddleware(validator)(endpoint))

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, nil)
			require.NoError(t, err)

			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()
			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedResponseStatus, res.StatusCode)

			expectedBody, err := os.ReadFile(tc.expectedResponseBodyFixture)
			require.NoError(t, err)

			assert.Equal(t, string(expectedBody), string(bodyBytes))
		})
	}
}
