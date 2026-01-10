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

func TestNewAnalyticsEndpoint(t *testing.T) {
	baseSummary := analytics.Summary{
		RaceCount:         3,
		IRatingStart:      1500,
		IRatingEnd:        1600,
		IRatingDelta:      100,
		IRatingGain:       130,
		IRatingLoss:       30,
		CPIStart:          3.0,
		CPIEnd:            3.2,
		CPIDelta:          0.2,
		CPIGain:           0.4,
		CPILoss:           0.2,
		Podiums:           2,
		Top5Finishes:      2,
		Wins:              1,
		AvgFinishPosition: 3.6666666666666665,
		AvgStartPosition:  6,
		PositionsGained:   2.3333333333333335,
		TotalIncidents:    6,
		AvgIncidents:      2,
	}

	seriesID42 := int64(42)
	seriesID43 := int64(43)

	groupedResult := &analytics.AnalyticsResult{
		Summary: baseSummary,
		GroupedBy: []analytics.GroupedSummary{
			{
				SeriesID: &seriesID42,
				Summary: analytics.Summary{
					RaceCount:         2,
					IRatingStart:      1500,
					IRatingEnd:        1520,
					IRatingDelta:      20,
					IRatingGain:       50,
					IRatingLoss:       30,
					CPIStart:          3.0,
					CPIEnd:            2.9,
					CPIDelta:          -0.1,
					CPIGain:           0.1,
					CPILoss:           0.2,
					Podiums:           1,
					Top5Finishes:      1,
					Wins:              0,
					AvgFinishPosition: 5,
					AvgStartPosition:  4,
					PositionsGained:   -1,
					TotalIncidents:    6,
					AvgIncidents:      3,
				},
			},
			{
				SeriesID: &seriesID43,
				Summary: analytics.Summary{
					RaceCount:         1,
					IRatingStart:      1520,
					IRatingEnd:        1600,
					IRatingDelta:      80,
					IRatingGain:       80,
					IRatingLoss:       0,
					CPIStart:          2.9,
					CPIEnd:            3.2,
					CPIDelta:          0.3,
					CPIGain:           0.3,
					CPILoss:           0,
					Podiums:           1,
					Top5Finishes:      1,
					Wins:              1,
					AvgFinishPosition: 1,
					AvgStartPosition:  10,
					PositionsGained:   9,
					TotalIncidents:    0,
					AvgIncidents:      0,
				},
			},
		},
	}

	type serviceCall struct {
		req    analytics.AnalyticsRequest
		result *analytics.AnalyticsResult
		err    error
	}

	testCases := []struct {
		name string

		driverID    string
		startTime   string
		endTime     string
		groupBy     []string
		granularity string
		seriesID    []string

		serviceCalls []serviceCall

		expectedStatus      int
		expectedBodyFixture string
	}{
		{
			name:      "success basic summary",
			driverID:  "12345",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-31T00:00:00Z",
			serviceCalls: []serviceCall{
				{
					req: analytics.AnalyticsRequest{
						DriverID: 12345,
						From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
					},
					result: &analytics.AnalyticsResult{
						Summary: baseSummary,
					},
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_analytics_success_response.json",
		},
		{
			name:      "success with groupBy",
			driverID:  "12345",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-31T00:00:00Z",
			groupBy:   []string{"series"},
			serviceCalls: []serviceCall{
				{
					req: analytics.AnalyticsRequest{
						DriverID: 12345,
						From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
						GroupBy:  []analytics.GroupByDimension{analytics.GroupBySeries},
					},
					result: groupedResult,
				},
			},
			expectedStatus:      http.StatusOK,
			expectedBodyFixture: "fixtures/get_analytics_with_groupby_response.json",
		},
		{
			name:                "missing required params",
			driverID:            "12345",
			serviceCalls:        []serviceCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_analytics_missing_params_response.json",
		},
		{
			name:                "groupBy and granularity mutual exclusive",
			driverID:            "12345",
			startTime:           "2024-01-01T00:00:00Z",
			endTime:             "2024-01-31T00:00:00Z",
			groupBy:             []string{"series"},
			granularity:         "day",
			serviceCalls:        []serviceCall{},
			expectedStatus:      http.StatusBadRequest,
			expectedBodyFixture: "fixtures/get_analytics_mutual_exclusive_response.json",
		},
		{
			name:      "service error",
			driverID:  "12345",
			startTime: "2024-01-01T00:00:00Z",
			endTime:   "2024-01-31T00:00:00Z",
			serviceCalls: []serviceCall{
				{
					req: analytics.AnalyticsRequest{
						DriverID: 12345,
						From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
					},
					err: errors.New("database error"),
				},
			},
			expectedStatus:      http.StatusInternalServerError,
			expectedBodyFixture: "fixtures/get_analytics_service_error_response.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockAnalyticsService(t)
			for _, call := range tc.serviceCalls {
				mockService.EXPECT().GetAnalytics(mock.Anything, call.req).
					Return(call.result, call.err)
			}

			r := chi.NewRouter()
			r.Use(correlation.Middleware(func() string { return testCorrelationID }))
			r.Get("/{driver_id}/analytics", NewAnalyticsEndpoint(mockService).ServeHTTP)

			ts := httptest.NewServer(r)
			defer ts.Close()

			url := ts.URL + "/" + tc.driverID + "/analytics?"
			if tc.startTime != "" {
				url += "startTime=" + tc.startTime + "&"
			}
			if tc.endTime != "" {
				url += "endTime=" + tc.endTime + "&"
			}
			for _, g := range tc.groupBy {
				url += "groupBy=" + g + "&"
			}
			if tc.granularity != "" {
				url += "granularity=" + tc.granularity + "&"
			}
			for _, s := range tc.seriesID {
				url += "seriesId=" + s + "&"
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

func TestParseInt64Slice(t *testing.T) {
	testCases := []struct {
		name            string
		values          []string
		expectedInts    []int64
		expectedInvalid []string
	}{
		{
			name:         "empty input",
			values:       []string{},
			expectedInts: nil,
		},
		{
			name:         "valid integers",
			values:       []string{"1", "2", "3"},
			expectedInts: []int64{1, 2, 3},
		},
		{
			name:            "mixed valid and invalid",
			values:          []string{"1", "abc", "3", "def"},
			expectedInts:    []int64{1, 3},
			expectedInvalid: []string{"abc", "def"},
		},
		{
			name:            "all invalid",
			values:          []string{"abc", "def"},
			expectedInvalid: []string{"abc", "def"},
		},
		{
			name:         "large int64 values",
			values:       []string{"9223372036854775807"},
			expectedInts: []int64{9223372036854775807},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ints, invalid := parseInt64Slice(tc.values)
			assert.Equal(t, tc.expectedInts, ints)
			assert.Equal(t, tc.expectedInvalid, invalid)
		})
	}
}