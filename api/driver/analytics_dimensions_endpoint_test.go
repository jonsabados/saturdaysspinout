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
	"github.com/jonsabados/saturdaysspinout/analytics"
	"github.com/jonsabados/saturdaysspinout/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyticsDimensionsEndpoint(t *testing.T) {
	type serviceCall struct {
		driverID   int64
		from       time.Time
		to         time.Time
		dimensions *analytics.Dimensions
		err        error
	}

	testCases := []struct {
		name string

		driverID  string
		startTime string
		endTime   string

		serviceCalls []serviceCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:      "success with dimensions",
			driverID:  "12345",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-31T00:00:00Z",
			serviceCalls: []serviceCall{
				{
					driverID: 12345,
					from:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					to:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
					dimensions: &analytics.Dimensions{
						SeriesIDs: []int64{42, 43},
						CarIDs:    []int64{10, 11},
						TrackIDs:  []int64{100, 101},
					},
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_analytics_dimensions_success_response.json",
		},
		{
			name:      "success with empty dimensions",
			driverID:  "12345",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-31T00:00:00Z",
			serviceCalls: []serviceCall{
				{
					driverID: 12345,
					from:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					to:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
					dimensions: &analytics.Dimensions{
						SeriesIDs: nil,
						CarIDs:    nil,
						TrackIDs:  nil,
					},
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_analytics_dimensions_empty_response.json",
		},
		{
			name:                "missing startTime",
			driverID:            "12345",
			endTime:             "2024-01-31T00:00:00Z",
			serviceCalls:        []serviceCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_analytics_dimensions_missing_start_time_response.json",
		},
		{
			name:                "missing endTime",
			driverID:            "12345",
			startTime:           "2024-01-01T00:00:00Z",
			serviceCalls:        []serviceCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_analytics_dimensions_missing_end_time_response.json",
		},
		{
			name:      "service error",
			driverID:  "12345",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-31T00:00:00Z",
			serviceCalls: []serviceCall{
				{
					driverID: 12345,
					from:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					to:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
					err:      errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_analytics_dimensions_service_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockAnalyticsService(t)
			for _, call := range tc.serviceCalls {
				mockService.EXPECT().GetDimensions(mock.Anything, call.driverID, call.from, call.to).
					Return(call.dimensions, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}/analytics/dimensions", NewAnalyticsDimensionsEndpoint(mockService).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/analytics/dimensions?"
			if tc.startTime != "" {
				url += "startTime=" + tc.startTime + "&"
			}
			if tc.endTime != "" {
				url += "endTime=" + tc.endTime + "&"
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