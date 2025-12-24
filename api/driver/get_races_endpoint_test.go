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

const testCorrelationID = "test-correlation-id"

func TestNewGetRacesEndpoint(t *testing.T) {
	testSessions := []store.DriverSession{
		{
			DriverID:              12345,
			SubsessionID:          100001,
			TrackID:               1,
			SeriesID:              42,
			SeriesName:            "Advanced Mazda MX-5 Cup Series",
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
			OldLicenseLevel:       17,
			NewLicenseLevel:       18,
			OldSubLevel:           381,
			NewSubLevel:           399,
			ReasonOut:             "Running",
		},
		{
			DriverID:              12345,
			SubsessionID:          100002,
			TrackID:               2,
			SeriesID:              43,
			SeriesName:            "Ferrari GT3 Challenge",
			CarID:                 11,
			StartTime:             time.Unix(1700100000, 0),
			StartPosition:         10,
			StartPositionInClass:  8,
			FinishPosition:        6,
			FinishPositionInClass: 4,
			Incidents:             2,
			OldCPI:                1.4,
			NewCPI:                1.3,
			OldIRating:            1550,
			NewIRating:            1580,
			OldLicenseLevel:       18,
			NewLicenseLevel:       18,
			OldSubLevel:           399,
			NewSubLevel:           412,
			ReasonOut:             "Running",
		},
	}

	type storeCall struct {
		driverID int64
		from     time.Time
		to       time.Time
		sessions []store.DriverSession
		err      error
	}

	testCases := []struct {
		name string

		driverID       string
		startTime      string
		endTime        string
		page           string
		resultsPerPage string

		storeCalls []storeCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:      "success with default pagination",
			driverID:  "12345",
			startTime: "2023-11-01T00:00:00Z",
			endTime:   "2023-11-30T00:00:00Z",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					from:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
					to:       time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					sessions: testSessions,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_races_success_response.json",
		},
		{
			name:           "success with custom pagination",
			driverID:       "12345",
			startTime:      "2023-11-01T00:00:00Z",
			endTime:        "2023-11-30T00:00:00Z",
			page:           "1",
			resultsPerPage: "1",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					from:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
					to:       time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					sessions: testSessions,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_races_paginated_response.json",
		},
		{
			name:                "missing startTime",
			driverID:            "12345",
			endTime:             "2023-11-30T00:00:00Z",
			storeCalls:          []storeCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_races_missing_start_time_response.json",
		},
		{
			name:                "missing endTime",
			driverID:            "12345",
			startTime:           "2023-11-01T00:00:00Z",
			storeCalls:          []storeCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_races_missing_end_time_response.json",
		},
		{
			name:                "invalid startTime format",
			driverID:            "12345",
			startTime:           "not-a-date",
			endTime:             "2023-11-30T00:00:00Z",
			storeCalls:          []storeCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_races_invalid_start_time_response.json",
		},
		{
			name:                "invalid page",
			driverID:            "12345",
			startTime:           "2023-11-01T00:00:00Z",
			endTime:             "2023-11-30T00:00:00Z",
			page:                "0",
			storeCalls:          []storeCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_races_invalid_page_response.json",
		},
		{
			name:      "store error",
			driverID:  "12345",
			startTime: "2023-11-01T00:00:00Z",
			endTime:   "2023-11-30T00:00:00Z",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					from:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
					to:       time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					err:      errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_races_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockGetRacesStore(t)
			for _, call := range tc.storeCalls {
				mockStore.EXPECT().GetDriverSessions(mock.Anything, call.driverID, call.from, call.to).
					Return(call.sessions, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}/races", NewGetRacesEndpoint(mockStore).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/races?"
			if tc.startTime != "" {
				url += "startTime=" + tc.startTime + "&"
			}
			if tc.endTime != "" {
				url += "endTime=" + tc.endTime + "&"
			}
			if tc.page != "" {
				url += "page=" + tc.page + "&"
			}
			if tc.resultsPerPage != "" {
				url += "resultsPerPage=" + tc.resultsPerPage + "&"
			}

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
