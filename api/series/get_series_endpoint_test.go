package series

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/series"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestNewGetSeriesEndpoint(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken: "test-access-token",
	}

	type serviceCall struct {
		result []series.Series
		err    error
	}

	testCases := []struct {
		name string

		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		tokenErr        error

		serviceCall *serviceCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:            "success",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				result: []series.Series{
					{
						ID:        159,
						Name:      "Porsche 911 GT3 Cup",
						ShortName: "Porsche Cup",
						Category:  "Road",
						LogoURL:   "https://images-static.iracing.com/img/logos/series/porsche-cup-logo.png",
						Active:    true,
						Official:  true,
					},
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_series_success_response.json",
		},
		{
			name:            "empty series",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				result: []series.Series{},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_series_empty_response.json",
		},
		{
			name:                "unauthorized",
			sessionClaims:       nil,
			sensitiveClaims:     nil,
			tokenErr:            errors.New("invalid token"),
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_series_unauthorized_response.json",
		},
		{
			name:            "iracing token expired",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				err: iracing.ErrUpstreamUnauthorized,
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_series_iracing_expired_response.json",
		},
		{
			name:            "service error",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				err: errors.New("iracing API error"),
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_series_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.tokenErr,
			}

			mockService := NewMockSeriesService(t)
			if tc.serviceCall != nil {
				mockService.EXPECT().GetAll(mock.Anything, "test-access-token").
					Return(tc.serviceCall.result, tc.serviceCall.err)
			}

			endpoint := NewGetSeriesEndpoint(mockService)
			handler := correlation.Middleware(func() string { return testCorrelationID })(api.AuthMiddleware(validator)(endpoint))

			ts := httptest.NewServer(handler)
			defer ts.Close()

			req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer test-token")

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			expectedBody, err := os.ReadFile(tc.expectedBodyFixture)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedBody), string(bodyBytes))
		})
	}
}