package session

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGetLapsEndpoint(t *testing.T) {
	testSessionClaims := &auth.SessionClaims{
		IRacingUserID:   1100750,
		IRacingUserName: "Jon Sabados",
	}
	testSensitiveClaims := &auth.SensitiveClaims{
		IRacingAccessToken: "test-access-token",
	}

	bestQualLapAt := time.Date(2024, 1, 15, 14, 25, 0, 0, time.UTC)

	testLapDataResponse := &iracing.LapDataResponse{
		Success: true,
		SessionInfo: iracing.LapDataSessionInfo{
			SubsessionID:     12345678,
			SessionID:        87654321,
			SimsessionNumber: 0,
			SimsessionType:   6,
			SimsessionName:   "RACE",
		},
		BestLapNum:      8,
		BestLapTime:     95500,
		BestNLapsNum:    3,
		BestNLapsTime:   287000,
		BestQualLapNum:  2,
		BestQualLapTime: 95300,
		BestQualLapAt:   &bestQualLapAt,
		CustID:          1100750,
		Name:            "Jon Sabados",
		CarID:           67,
		LicenseLevel:    8,
		Laps: []iracing.Lap{
			{
				LapNumber:       1,
				Flags:           0,
				Incident:        false,
				SessionTime:     60000,
				LapTime:         98500,
				PersonalBestLap: false,
				LapEvents:       []string{},
			},
			{
				LapNumber:       2,
				Flags:           0,
				Incident:        false,
				SessionTime:     158500,
				LapTime:         96200,
				PersonalBestLap: false,
				LapEvents:       []string{},
			},
			{
				LapNumber:       3,
				Flags:           0,
				Incident:        true,
				SessionTime:     254700,
				LapTime:         97800,
				PersonalBestLap: false,
				LapEvents:       []string{"off track"},
			},
			{
				LapNumber:       8,
				Flags:           0,
				Incident:        false,
				SessionTime:     750000,
				LapTime:         95500,
				PersonalBestLap: true,
				LapEvents:       []string{},
			},
		},
	}

	type clientCall struct {
		subsessionID int64
		simsession   int
		driverID     int64
		result       *iracing.LapDataResponse
		err          error
	}

	testCases := []struct {
		name string

		subsessionID string
		simsession   string
		driverID     string

		sessionClaims   *auth.SessionClaims
		sensitiveClaims *auth.SensitiveClaims
		tokenErr        error

		clientCall *clientCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:            "success",
			subsessionID:    "12345678",
			simsession:      "0",
			driverID:        "1100750",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				simsession:   0,
				driverID:     1100750,
				result:       testLapDataResponse,
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_laps_success_response.json",
		},
		{
			name:            "invalid subsession_id",
			subsessionID:    "not-a-number",
			simsession:      "0",
			driverID:        "1100750",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_laps_invalid_subsession_id_response.json",
		},
		{
			name:            "invalid simsession",
			subsessionID:    "12345678",
			simsession:      "not-a-number",
			driverID:        "1100750",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_laps_invalid_simsession_response.json",
		},
		{
			name:            "invalid driver_id",
			subsessionID:    "12345678",
			simsession:      "0",
			driverID:        "not-a-number",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_laps_invalid_driver_id_response.json",
		},
		{
			name:                "unauthorized",
			subsessionID:        "12345678",
			simsession:          "0",
			driverID:            "1100750",
			sessionClaims:       nil,
			sensitiveClaims:     nil,
			tokenErr:            errors.New("invalid token"),
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_laps_unauthorized_response.json",
		},
		{
			name:            "iracing token expired",
			subsessionID:    "12345678",
			simsession:      "0",
			driverID:        "1100750",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				simsession:   0,
				driverID:     1100750,
				err:          iracing.ErrUpstreamUnauthorized,
			},
			expectedStatus:      http.StatusUnauthorized,
			expectedBodyFixture: "fixtures/get_laps_iracing_expired_response.json",
		},
		{
			name:            "client error",
			subsessionID:    "12345678",
			simsession:      "0",
			driverID:        "1100750",
			sessionClaims:   testSessionClaims,
			sensitiveClaims: testSensitiveClaims,
			clientCall: &clientCall{
				subsessionID: 12345678,
				simsession:   0,
				driverID:     1100750,
				err:          errors.New("iracing API error"),
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_laps_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator := &stubTokenValidator{
				sessionClaims:   tc.sessionClaims,
				sensitiveClaims: tc.sensitiveClaims,
				err:             tc.tokenErr,
			}

			mockClient := NewMockLapDataClient(t)
			if tc.clientCall != nil {
				mockClient.EXPECT().GetLapData(mock.Anything, "test-access-token", tc.clientCall.subsessionID, tc.clientCall.simsession, mock.Anything).
					Return(tc.clientCall.result, tc.clientCall.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Use(api.AuthMiddleware(validator))
			r.Get("/{"+SubsessionIDPathParam+"}/simsession/{"+SimsessionPathParam+"}/driver/{"+DriverIDPathParam+"}/laps", NewGetLapsEndpoint(mockClient).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.subsessionID + "/simsession/" + tc.simsession + "/driver/" + tc.driverID + "/laps"

			req, err := http.NewRequest(http.MethodGet, url, nil)
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