package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntitlementMiddleware(t *testing.T) {
	testCases := []struct {
		name string

		requiredEntitlement string
		sessionClaims       *auth.SessionClaims

		expectNextCalled            bool
		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:                        "missing session claims returns 401",
			requiredEntitlement:         "developer",
			sessionClaims:               nil,
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/entitlement_missing_claims_response.json",
		},
		{
			name:                "missing required entitlement returns 403",
			requiredEntitlement: "developer",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID:   12345,
				IRacingUserName: "Test Driver",
				Entitlements:    []string{"beta-tester"},
			},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusForbidden,
			expectedResponseBodyFixture: "fixtures/entitlement_forbidden_response.json",
		},
		{
			name:                "no entitlements returns 403",
			requiredEntitlement: "developer",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID:   12345,
				IRacingUserName: "Test Driver",
				Entitlements:    nil,
			},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusForbidden,
			expectedResponseBodyFixture: "fixtures/entitlement_forbidden_response.json",
		},
		{
			name:                "has required entitlement passes through",
			requiredEntitlement: "developer",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID:   12345,
				IRacingUserName: "Test Driver",
				Entitlements:    []string{"developer"},
			},
			expectNextCalled:            true,
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/entitlement_success_response.json",
		},
		{
			name:                "has required entitlement among multiple passes through",
			requiredEntitlement: "developer",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID:   12345,
				IRacingUserName: "Test Driver",
				Entitlements:    []string{"beta-tester", "developer", "admin"},
			},
			expectNextCalled:            true,
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/entitlement_success_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			nextCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"next_called": true})
			})

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Route("/protected", func(r chi.Router) {
				r.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						if tc.sessionClaims != nil {
							reqCtx := context.WithValue(req.Context(), sessionClaimsKey, tc.sessionClaims)
							req = req.WithContext(reqCtx)
						}
						next.ServeHTTP(w, req)
					})
				})
				r.Use(EntitlementMiddleware(tc.requiredEntitlement))
				r.Get("/", nextHandler)
			})

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/protected", nil)
			require.NoError(t, err)

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
