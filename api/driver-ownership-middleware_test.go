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

func TestDriverOwnershipMiddleware(t *testing.T) {
	testCases := []struct {
		name string

		pathParam     string
		urlPath       string
		sessionClaims *auth.SessionClaims

		expectNextCalled            bool
		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:                        "missing session claims returns 401",
			pathParam:                   "driver_id",
			urlPath:                     "/driver/12345",
			sessionClaims:               nil,
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/driver_ownership_missing_claims_response.json",
		},
		{
			name:      "invalid driver_id (non-integer) returns 400",
			pathParam: "driver_id",
			urlPath:   "/driver/notanumber",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID: 12345,
			},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusBadRequest,
			expectedResponseBodyFixture: "fixtures/driver_ownership_invalid_id_response.json",
		},
		{
			name:      "driver_id mismatch returns 403",
			pathParam: "driver_id",
			urlPath:   "/driver/99999",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID: 12345,
			},
			expectNextCalled:            false,
			expectedResponseStatus:      http.StatusForbidden,
			expectedResponseBodyFixture: "fixtures/driver_ownership_forbidden_response.json",
		},
		{
			name:      "matching driver_id passes through",
			pathParam: "driver_id",
			urlPath:   "/driver/12345",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID: 12345,
			},
			expectNextCalled:            true,
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/driver_ownership_success_response.json",
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
			r.Route("/driver/{driver_id}", func(r chi.Router) {
				r.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						if tc.sessionClaims != nil {
							reqCtx := context.WithValue(req.Context(), sessionClaimsKey, tc.sessionClaims)
							req = req.WithContext(reqCtx)
						}
						next.ServeHTTP(w, req)
					})
				})
				r.Use(DriverOwnershipMiddleware(tc.pathParam))
				r.Get("/", nextHandler)
			})

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+tc.urlPath, nil)
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
