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
	"github.com/jonsabados/saturdaysspinout/journal"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGetJournalEntryEndpoint(t *testing.T) {
	testEntry := journal.Entry{
		RaceID:    1700000000,
		CreatedAt: time.Unix(1000, 0),
		UpdatedAt: time.Unix(2000, 0),
		Notes:     "Great race!",
		Tags:      []string{"sentiment:good", "podium"},
		Race: &store.DriverSession{
			DriverID:       12345,
			SubsessionID:   100001,
			TrackID:        1,
			CarID:          10,
			SeriesID:       42,
			SeriesName:     "Advanced Mazda MX-5 Cup Series",
			StartTime:      time.Unix(1700000000, 0),
			FinishPosition: 2,
		},
	}

	type getCall struct {
		driverID int64
		raceID   int64
		entry    *journal.Entry
		err      error
	}

	testCases := []struct {
		name string

		driverID string
		raceID   string

		getCalls []getCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:     "success",
			driverID: "12345",
			raceID:   "1700000000",
			getCalls: []getCall{
				{driverID: 12345, raceID: 1700000000, entry: &testEntry},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_journal_success_response.json",
		},
		{
			name:                "invalid driver_id",
			driverID:            "not-a-number",
			raceID:              "1700000000",
			getCalls:            []getCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_journal_invalid_driver_id_response.json",
		},
		{
			name:                "invalid driver_race_id",
			driverID:            "12345",
			raceID:              "not-a-number",
			getCalls:            []getCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_journal_invalid_race_id_response.json",
		},
		{
			name:     "not found",
			driverID: "12345",
			raceID:   "1700000000",
			getCalls: []getCall{
				{driverID: 12345, raceID: 1700000000, entry: nil},
			},
			expectedStatus:      http.StatusNotFound,
			expectedBodyFixture: "fixtures/get_journal_not_found_response.json",
		},
		{
			name:     "store error",
			driverID: "12345",
			raceID:   "1700000000",
			getCalls: []getCall{
				{driverID: 12345, raceID: 1700000000, err: errors.New("database error")},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_journal_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockGetJournalEntryStore(t)
			for _, call := range tc.getCalls {
				mockService.EXPECT().Get(mock.Anything, call.driverID, call.raceID).
					Return(call.entry, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}/races/{driver_race_id}/journal", NewGetJournalEntryEndpoint(mockService).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/races/" + tc.raceID + "/journal"
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