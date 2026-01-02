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

func TestNewListJournalEntriesEndpoint(t *testing.T) {
	testEntries := []journal.Entry{
		{
			RaceID:    1700100000,
			CreatedAt: time.Unix(2000, 0),
			UpdatedAt: time.Unix(2000, 0),
			Notes:     "Second race",
			Tags:      []string{"sentiment:neutral"},
			Race: &store.DriverSession{
				DriverID:       12345,
				SubsessionID:   100002,
				TrackID:        2,
				CarID:          11,
				SeriesID:       43,
				SeriesName:     "Ferrari GT3 Challenge",
				StartTime:      time.Unix(1700100000, 0),
				FinishPosition: 5,
			},
		},
		{
			RaceID:    1700000000,
			CreatedAt: time.Unix(1000, 0),
			UpdatedAt: time.Unix(1000, 0),
			Notes:     "First race",
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
		},
	}

	type listCall struct {
		input   journal.ListInput
		entries []journal.Entry
		err     error
	}

	testCases := []struct {
		name string

		driverID       string
		startTime      string
		endTime        string
		page           string
		resultsPerPage string

		listCalls []listCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:      "success with default pagination",
			driverID:  "12345",
			startTime: "2023-11-01T00:00:00Z",
			endTime:   "2023-11-30T00:00:00Z",
			listCalls: []listCall{
				{
					input: journal.ListInput{
						DriverID: 12345,
						From:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
						To:       time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					},
					entries: testEntries,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/list_journal_success_response.json",
		},
		{
			name:           "success with pagination",
			driverID:       "12345",
			startTime:      "2023-11-01T00:00:00Z",
			endTime:        "2023-11-30T00:00:00Z",
			page:           "1",
			resultsPerPage: "1",
			listCalls: []listCall{
				{
					input: journal.ListInput{
						DriverID: 12345,
						From:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
						To:       time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					},
					entries: testEntries,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/list_journal_paginated_response.json",
		},
		{
			name:                "missing startTime",
			driverID:            "12345",
			endTime:             "2023-11-30T00:00:00Z",
			listCalls:           []listCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/list_journal_missing_start_time_response.json",
		},
		{
			name:                "missing endTime",
			driverID:            "12345",
			startTime:           "2023-11-01T00:00:00Z",
			listCalls:           []listCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/list_journal_missing_end_time_response.json",
		},
		{
			name:                "invalid startTime format",
			driverID:            "12345",
			startTime:           "not-a-date",
			endTime:             "2023-11-30T00:00:00Z",
			listCalls:           []listCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/list_journal_invalid_start_time_response.json",
		},
		{
			name:      "store error",
			driverID:  "12345",
			startTime: "2023-11-01T00:00:00Z",
			endTime:   "2023-11-30T00:00:00Z",
			listCalls: []listCall{
				{
					input: journal.ListInput{
						DriverID: 12345,
						From:     time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
						To:       time.Date(2023, 11, 30, 0, 0, 0, 0, time.UTC),
					},
					err: errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/list_journal_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockListJournalEntriesStore(t)
			for _, call := range tc.listCalls {
				mockService.EXPECT().List(mock.Anything, call.input).
					Return(call.entries, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}/journal", NewListJournalEntriesEndpoint(mockService).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/journal?"
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