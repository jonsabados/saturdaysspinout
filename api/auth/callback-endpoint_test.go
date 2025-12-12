package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testCorrelationID = "test-correlation-id"

func TestNewAuthCallbackEndpoint(t *testing.T) {
	type authServiceCall struct {
		inputCode         string
		inputCodeVerifier string
		redirectURI       string
		result            *auth.Result
		resultErr         error
	}

	testCases := []struct {
		name string

		inputFixture             string
		expectedAuthServiceCalls []authServiceCall

		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:         "success",
			inputFixture: "fixtures/auth_callback_valid_request.json",
			expectedAuthServiceCalls: []authServiceCall{
				{
					inputCode:         "test-auth-code",
					inputCodeVerifier: "test-code-verifier",
					redirectURI:       "http://localhost:5173/auth/ir/callback",
					result: &auth.Result{
						Token:     "test-jwt-token",
						ExpiresAt: time.Unix(1735689600, 0),
						UserID:    1100750,
						UserName:  "Jon Sabados",
					},
					resultErr: nil,
				},
			},
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/auth_callback_success_response.json",
		},
		{
			name:                        "missing code",
			inputFixture:                "fixtures/auth_callback_missing_code_request.json",
			expectedAuthServiceCalls:    []authServiceCall{},
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/auth_callback_missing_code_response.json",
		},
		{
			name:                        "missing code_verifier",
			inputFixture:                "fixtures/auth_callback_missing_verifier_request.json",
			expectedAuthServiceCalls:    []authServiceCall{},
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/auth_callback_missing_verifier_response.json",
		},
		{
			name:                        "missing redirect_uri",
			inputFixture:                "fixtures/auth_callback_missing_redirect_uri_request.json",
			expectedAuthServiceCalls:    []authServiceCall{},
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/auth_callback_missing_redirect_uri_response.json",
		},
		{
			name:                        "invalid JSON body",
			inputFixture:                "fixtures/auth_callback_invalid_json_request.json",
			expectedAuthServiceCalls:    []authServiceCall{},
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/auth_callback_invalid_json_response.json",
		},
		{
			name:         "auth service error",
			inputFixture: "fixtures/auth_callback_valid_request.json",
			expectedAuthServiceCalls: []authServiceCall{
				{
					inputCode:         "test-auth-code",
					inputCodeVerifier: "test-code-verifier",
					redirectURI:       "http://localhost:5173/auth/ir/callback",
					result:            nil,
					resultErr:         errors.New("token exchange failed"),
				},
			},
			expectedResponseStatus:      http.StatusInternalServerError,
			expectedResponseBodyFixture: "fixtures/auth_callback_service_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			authService := NewMockService(t)
			for _, call := range tc.expectedAuthServiceCalls {
				authService.EXPECT().HandleCallback(mock.Anything, call.inputCode, call.inputCodeVerifier, call.redirectURI).Return(call.result, call.resultErr)
			}

			endpoint := NewAuthCallbackEndpoint(authService)
			handler := correlation.Middleware(func() string { return testCorrelationID })(endpoint)

			ts := httptest.NewServer(handler)
			defer ts.Close()

			requestBody, err := os.ReadFile(tc.inputFixture)
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, bytes.NewReader(requestBody))
			require.NoError(t, err)

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
