package developer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCorrelationID = "test-correlation-id"

type stubTokenValidator struct {
	sessionClaims   *auth.SessionClaims
	sensitiveClaims *auth.SensitiveClaims
	err             error
}

func (s *stubTokenValidator) ValidateToken(_ context.Context, _ string) (*auth.SessionClaims, *auth.SensitiveClaims, error) {
	return s.sessionClaims, s.sensitiveClaims, s.err
}

func TestNewIRacingTokenEndpoint(t *testing.T) {
	testCases := []struct {
		name string

		authHeader      string
		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		validatorErr    error

		expectedResponseStatus      int
		expectedResponseBodyFixture string
	}{
		{
			name:       "success",
			authHeader: "Bearer valid-token",
			sessionClaims: &auth.SessionClaims{
				IRacingUserID:   12345,
				IRacingUserName: "Test Driver",
				Entitlements:    []string{"developer"},
			},
			sensitiveClaims: &auth.SensitiveClaims{
				IRacingAccessToken:  "test-iracing-access-token",
				IRacingRefreshToken: "test-iracing-refresh-token",
			},
			expectedResponseStatus:      http.StatusOK,
			expectedResponseBodyFixture: "fixtures/iracing_token_success_response.json",
		},
		{
			name:                        "missing auth header returns 401",
			authHeader:                  "",
			expectedResponseStatus:      http.StatusUnauthorized,
			expectedResponseBodyFixture: "fixtures/iracing_token_missing_auth_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.validatorErr,
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Route("/developer", func(r chi.Router) {
				r.Use(api.AuthMiddleware(validator))
				r.Get("/iracing-token", NewIRacingTokenEndpoint().ServeHTTP)
			})

			ts := httptest.NewServer(r)
			defer ts.Close()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/developer/iracing-token", nil)
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

			assert.JSONEq(t, string(expectedBody), string(bodyBytes))
		})
	}
}
