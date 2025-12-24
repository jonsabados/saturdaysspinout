package tracks

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
	"github.com/jonsabados/saturdaysspinout/tracks"
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

func TestNewGetTracksEndpoint(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken: "test-access-token",
	}

	type serviceCall struct {
		result []tracks.Track
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
				result: []tracks.Track{
					{
						ID:                1,
						Name:              "Lime Rock Park",
						ConfigName:        "Full Course",
						Category:          "road",
						Location:          "Lakeville, Connecticut, USA",
						CornersPerLap:     7,
						LengthMiles:       1.53,
						Description:       "<p>A great road course</p>",
						LogoURL:           "https://images-static.iracing.com/logo.png",
						SmallImageURL:     "https://images-static.iracing.com/small.jpg",
						LargeImageURL:     "https://images-static.iracing.com/large.jpg",
						TrackMapURL:       "https://example.com/map",
						IsDirt:            false,
						IsOval:            false,
						HasNightLighting:  false,
						RainEnabled:       true,
						FreeWithSub:       false,
						Retired:           false,
						PitRoadSpeedLimit: 45,
					},
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_tracks_success_response.json",
		},
		{
			name:            "empty tracks",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				result: []tracks.Track{},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_tracks_empty_response.json",
		},
		{
			name:                "unauthorized",
			sessionClaims:       nil,
			sensitiveClaims:     nil,
			tokenErr:            errors.New("invalid token"),
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_tracks_unauthorized_response.json",
		},
		{
			name:            "iracing token expired",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				err: iracing.ErrUpstreamUnauthorized,
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_tracks_iracing_expired_response.json",
		},
		{
			name:            "service error",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			serviceCall: &serviceCall{
				err: errors.New("iracing API error"),
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_tracks_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.tokenErr,
			}

			mockService := NewMockTracksService(t)
			if tc.serviceCall != nil {
				mockService.EXPECT().GetAll(mock.Anything, "test-access-token").
					Return(tc.serviceCall.result, tc.serviceCall.err)
			}

			endpoint := NewGetTracksEndpoint(mockService)
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