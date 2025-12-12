package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testCorrelationID = "test-correlation-id"

func TestAuthMiddleware(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		SessionID:       "test-session-id",
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken:  "test-access-token",
		IRacingRefreshToken: "test-refresh-token",
		IRacingTokenExpiry:  1735689600,
	}

	type validatorCall struct {
		inputToken      string
		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		err             error
	}

	testCases := []struct {
		name string

		httpMethod       string
		authHeader       string
		validatorCalls   []validatorCall
		expectNextCalled bool

		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:                        "OPTIONS bypasses auth",
			httpMethod:                  http.MethodOptions,
			authHeader:                  "",
			validatorCalls:              []validatorCall{},
			expectNextCalled:            true,
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/auth_middleware_options_response.json",
		},
		{
			name:                        "missing authorization header",
			httpMethod:                  http.MethodGet,
			authHeader:                  "",
			validatorCalls:              []validatorCall{},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/auth_middleware_missing_header_response.json",
		},
		{
			name:                        "invalid authorization header format",
			httpMethod:                  http.MethodGet,
			authHeader:                  "Basic dXNlcjpwYXNz",
			validatorCalls:              []validatorCall{},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/auth_middleware_invalid_format_response.json",
		},
		{
			name:       "token validation fails",
			httpMethod: http.MethodGet,
			authHeader: "Bearer invalid-token",
			validatorCalls: []validatorCall{
				{
					inputToken:      "invalid-token",
					sessionClaims:   nil,
					sensitiveClaims: nil,
					err:             errors.New("token expired"),
				},
			},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/auth_middleware_invalid_token_response.json",
		},
		{
			name:       "valid token sets claims in context",
			httpMethod: http.MethodGet,
			authHeader: "Bearer valid-token",
			validatorCalls: []validatorCall{
				{
					inputToken:      "valid-token",
					sessionClaims:   testSessionClaims,
					sensitiveClaims: testSensitiveClaims,
					err:             nil,
				},
			},
			expectNextCalled:            true,
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/auth_middleware_success_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			validator := NewMockTokenValidator(t)
			for _, call := range tc.validatorCalls {
				validator.EXPECT().ValidateToken(mock.Anything, call.inputToken).Return(call.sessionClaims, call.sensitiveClaims, call.err)
			}

			nextCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true

				response := map[string]any{
					"next_called": true,
				}

				if claims := SensitiveClaimsFromContext(r.Context()); claims != nil {
					response["sensitive_claims"] = claims
				}
				if claims := SessionClaimsFromContext(r.Context()); claims != nil {
					response["session_claims"] = map[string]any{
						"session_id":        claims.SessionID,
						"iracing_user_id":   claims.IRacingUserID,
						"iracing_user_name": claims.IRacingUserName,
					}
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			})

			handler := correlation.Middleware(func() string { return testCorrelationID })(AuthMiddleware(validator)(nextHandler))

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequestWithContext(ctx, tc.httpMethod, ts.URL, nil)
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
			assert.Equal(t, tc.expectNextCalled, nextCalled)

			expectedBody, err := os.ReadFile(tc.expectedResponseBodyFixture)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedBody), string(bodyBytes))
		})
	}
}
