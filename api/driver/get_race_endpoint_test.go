package driver

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGetRaceEndpoint(t *testing.T) {
	testSession := &store.DriverSession{
		DriverID:              12345,
		SubsessionID:          100001,
		TrackID:               1,
		CarID:                 10,
		StartTime:             time.Unix(1700000000, 0),
		StartPosition:         5,
		StartPositionInClass:  3,
		FinishPosition:        2,
		FinishPositionInClass: 1,
		Incidents:             4,
		OldCPI:                1.5,
		NewCPI:                1.4,
		OldIRating:            1500,
		NewIRating:            1550,
		ReasonOut:             "Running",
	}

	type storeCall struct {
		driverID  int64
		startTime time.Time
		session   *store.DriverSession
		err       error
	}

	testCases := []struct {
		name string

		driverID     string
		driverRaceID string

		storeCalls []storeCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:         "success",
			driverID:     "12345",
			driverRaceID: "1700000000",
			storeCalls: []storeCall{
				{
					driverID:  12345,
					startTime: time.Unix(1700000000, 0),
					session:   testSession,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_race_success_response.json",
		},
		{
			name:         "not found",
			driverID:     "12345",
			driverRaceID: "1700000000",
			storeCalls: []storeCall{
				{
					driverID:  12345,
					startTime: time.Unix(1700000000, 0),
					session:   nil,
				},
			},
			expectedStatus:      http.StatusNotFound,
			expectedBodyFixture: "fixtures/get_race_not_found_response.json",
		},
		{
			name:                "invalid driver_race_id",
			driverID:            "12345",
			driverRaceID:        "not-an-integer",
			storeCalls:          []storeCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_race_invalid_driver_race_id_response.json",
		},
		{
			name:         "store error",
			driverID:     "12345",
			driverRaceID: "1700000000",
			storeCalls: []storeCall{
				{
					driverID:  12345,
					startTime: time.Unix(1700000000, 0),
					err:       errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_race_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockGetRaceStore(t)
			for _, call := range tc.storeCalls {
				mockStore.EXPECT().GetDriverSession(mock.Anything, call.driverID, call.startTime).
					Return(call.session, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}/races/{driver_race_id}", NewGetRaceEndpoint(mockStore).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/races/" + tc.driverRaceID

			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

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