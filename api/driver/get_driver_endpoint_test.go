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

func TestNewGetDriverEndpoint(t *testing.T) {
	racesIngestedTo := time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)
	ingestionBlockedUntil := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)

	testDriver := &store.Driver{
		DriverID:        12345,
		DriverName:      "Jon Sabados",
		MemberSince:     time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		RacesIngestedTo: &racesIngestedTo,
		FirstLogin:      time.Date(2023, 6, 1, 10, 0, 0, 0, time.UTC),
		LastLogin:       time.Date(2023, 11, 14, 22, 0, 0, 0, time.UTC),
		LoginCount:      42,
		SessionCount:    150,
	}

	testDriverWithBlocked := &store.Driver{
		DriverID:              12345,
		DriverName:            "Jon Sabados",
		MemberSince:           time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC),
		RacesIngestedTo:       &racesIngestedTo,
		IngestionBlockedUntil: &ingestionBlockedUntil,
		FirstLogin:            time.Date(2023, 6, 1, 10, 0, 0, 0, time.UTC),
		LastLogin:             time.Date(2023, 11, 14, 22, 0, 0, 0, time.UTC),
		LoginCount:            42,
		SessionCount:          150,
	}

	type storeCall struct {
		driverID int64
		driver   *store.Driver
		err      error
	}

	testCases := []struct {
		name string

		driverID string

		storeCalls []storeCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:     "success",
			driverID: "12345",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					driver:   testDriver,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_driver_success_response.json",
		},
		{
			name:     "success with ingestion blocked",
			driverID: "12345",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					driver:   testDriverWithBlocked,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_driver_with_blocked_response.json",
		},
		{
			name:     "driver not found",
			driverID: "99999",
			storeCalls: []storeCall{
				{
					driverID: 99999,
					driver:   nil,
				},
			},
			expectedStatus:      http.StatusNotFound,
			expectedBodyFixture: "fixtures/get_driver_not_found_response.json",
		},
		{
			name:     "store error",
			driverID: "12345",
			storeCalls: []storeCall{
				{
					driverID: 12345,
					err:      errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_driver_store_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockGetDriverStore(t)
			for _, call := range tc.storeCalls {
				mockStore.EXPECT().GetDriver(mock.Anything, call.driverID).
					Return(call.driver, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}", NewGetDriverEndpoint(mockStore).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID

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
