package driver

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDeleteJournalEntryEndpoint(t *testing.T) {
	type deleteCall struct {
		driverID int64
		raceID   int64
		err      error
	}

	testCases := []struct {
		name string

		driverID string
		raceID   string

		deleteCalls []deleteCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:     "success",
			driverID: "12345",
			raceID:   "1700000000",
			deleteCalls: []deleteCall{
				{driverID: 12345, raceID: 1700000000},
			},
			expectedStatus:      http.StatusNoContent,
			expectedBodyFixture: "",
		},
		{
			name:                "invalid driver_id",
			driverID:            "not-a-number",
			raceID:              "1700000000",
			deleteCalls:         []deleteCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/delete_journal_invalid_driver_id_response.json",
		},
		{
			name:                "invalid driver_race_id",
			driverID:            "12345",
			raceID:              "not-a-number",
			deleteCalls:         []deleteCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/delete_journal_invalid_race_id_response.json",
		},
		{
			name:     "store error",
			driverID: "12345",
			raceID:   "1700000000",
			deleteCalls: []deleteCall{
				{driverID: 12345, raceID: 1700000000, err: errors.New("database error")},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/delete_journal_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockDeleteJournalEntryStore(t)
			for _, call := range tc.deleteCalls {
				mockService.EXPECT().Delete(mock.Anything, call.driverID, call.raceID).
					Return(call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Delete("/{driver_id}/races/{driver_race_id}/journal", NewDeleteJournalEntryEndpoint(mockService).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/races/" + tc.raceID + "/journal"
			req, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			res, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedStatus, res.StatusCode)

			if tc.expectedBodyFixture != "" {
				expectedBody, err := os.ReadFile(tc.expectedBodyFixture)
				require.NoError(t, err)
				assert.JSONEq(t, string(expectedBody), string(bodyBytes))
			} else {
				assert.Empty(t, bodyBytes)
			}
		})
	}
}