package driver

import (
	"bytes"
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

func TestNewSaveJournalEndpoint(t *testing.T) {
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

	type validateCall struct {
		driverID int64
		raceID   int64
		exists   bool
		err      error
	}

	type saveCall struct {
		input journal.SaveInput
		entry *journal.Entry
		err   error
	}

	testCases := []struct {
		name string

		driverID    string
		raceID      string
		requestBody string

		validateCalls []validateCall
		saveCalls     []saveCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:        "success",
			driverID:    "12345",
			raceID:      "1700000000",
			requestBody: `{"notes": "Great race!", "tags": ["sentiment:good", "podium"]}`,
			validateCalls: []validateCall{
				{driverID: 12345, raceID: 1700000000, exists: true},
			},
			saveCalls: []saveCall{
				{
					input: journal.SaveInput{DriverID: 12345, RaceID: 1700000000, Notes: "Great race!", Tags: []string{"sentiment:good", "podium"}},
					entry: &testEntry,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/save_journal_success_response.json",
		},
		{
			name:                "invalid driver_id",
			driverID:            "not-a-number",
			raceID:              "1700000000",
			requestBody:         `{"notes": "Great race!"}`,
			validateCalls:       []validateCall{},
			saveCalls:           []saveCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/save_journal_invalid_driver_id_response.json",
		},
		{
			name:                "invalid driver_race_id",
			driverID:            "12345",
			raceID:              "not-a-number",
			requestBody:         `{"notes": "Great race!"}`,
			validateCalls:       []validateCall{},
			saveCalls:           []saveCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/save_journal_invalid_race_id_response.json",
		},
		{
			name:                "invalid json body",
			driverID:            "12345",
			raceID:              "1700000000",
			requestBody:         `{invalid json`,
			validateCalls:       []validateCall{},
			saveCalls:           []saveCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/save_journal_invalid_json_response.json",
		},
		{
			name:                "invalid tag",
			driverID:            "12345",
			raceID:              "1700000000",
			requestBody:         `{"notes": "Great race!", "tags": ["sentiment:invalid"]}`,
			validateCalls:       []validateCall{},
			saveCalls:           []saveCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/save_journal_invalid_tag_response.json",
		},
		{
			name:        "race not found",
			driverID:    "12345",
			raceID:      "1700000000",
			requestBody: `{"notes": "Great race!"}`,
			validateCalls: []validateCall{
				{driverID: 12345, raceID: 1700000000, exists: false},
			},
			saveCalls:           []saveCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/save_journal_race_not_found_response.json",
		},
		{
			name:        "validate race error",
			driverID:    "12345",
			raceID:      "1700000000",
			requestBody: `{"notes": "Great race!"}`,
			validateCalls: []validateCall{
				{driverID: 12345, raceID: 1700000000, err: errors.New("database error")},
			},
			saveCalls:           []saveCall{},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/save_journal_store_error_response.json",
		},
		{
			name:        "save error",
			driverID:    "12345",
			raceID:      "1700000000",
			requestBody: `{"notes": "Great race!"}`,
			validateCalls: []validateCall{
				{driverID: 12345, raceID: 1700000000, exists: true},
			},
			saveCalls: []saveCall{
				{
					input: journal.SaveInput{DriverID: 12345, RaceID: 1700000000, Notes: "Great race!"},
					err:   errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/save_journal_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockJournalServiceForSave(t)
			for _, call := range tc.validateCalls {
				mockService.EXPECT().ValidateRaceExists(mock.Anything, call.driverID, call.raceID).
					Return(call.exists, call.err)
			}
			for _, call := range tc.saveCalls {
				mockService.EXPECT().Save(mock.Anything, call.input).
					Return(call.entry, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Put("/{driver_id}/races/{driver_race_id}/journal", NewSaveJournalEndpoint(mockService).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/races/" + tc.raceID + "/journal"
			req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

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