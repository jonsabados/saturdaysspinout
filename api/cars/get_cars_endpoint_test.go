package cars

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
	"github.com/jonsabados/saturdaysspinout/cars"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/jonsabados/saturdaysspinout/iracing"
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

func TestNewGetCarsEndpoint(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken: "test-access-token",
	}

	type serviceCall struct {
		result []cars.Car
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
				result: []cars.Car{
					{
						ID:                   1,
						Name:                 "Mazda MX-5 Miata",
						NameAbbreviated:      "MX-5",
						Make:                 "Mazda",
						Model:                "MX-5 Miata",
						Description:          "<p>The perfect starter car</p>",
						Weight:               2332,
						HPUnderHood:          155,
						HPActual:             155,
						Categories:           []string{"road"},
						LogoURL:              "https://images-static.iracing.com/logo.png",
						SmallImageURL:        "https://images-static.iracing.com/small.jpg",
						LargeImageURL:        "https://images-static.iracing.com/large.jpg",
						HasHeadlights:        true,
						HasMultipleDryTires:  false,
						RainEnabled:          true,
						FreeWithSubscription: true,
						Retired:              false,
					},
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_cars_success_response.json",
		},
		{
			name:            "empty cars",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				result: []cars.Car{},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_cars_empty_response.json",
		},
		{
			name:                "unauthorized",
			sessionClaims:       nil,
			sensitiveClaims:     nil,
			tokenErr:            errors.New("invalid token"),
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_cars_unauthorized_response.json",
		},
		{
			name:            "iracing token expired",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				err: iracing.ErrUpstreamUnauthorized,
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_cars_iracing_expired_response.json",
		},
		{
			name:            "service error",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				err: errors.New("iracing API error"),
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_cars_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.tokenErr,
			}

			mockService := NewMockCarsService(t)
			if tc.serviceCall != nil {
				mockService.EXPECT().GetAll(mock.Anything, "test-access-token").
					Return(tc.serviceCall.result, tc.serviceCall.err)
			}

			endpoint := NewGetCarsEndpoint(mockService)
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