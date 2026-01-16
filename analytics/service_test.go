package analytics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_GetDimensions(t *testing.T) {
	testSessions := []store.DriverSession{
		{SeriesID: 42, CarID: 10, TrackID: 100},
		{SeriesID: 42, CarID: 11, TrackID: 101},
		{SeriesID: 43, CarID: 10, TrackID: 100},
	}

	type storeCall struct {
		sessions []store.DriverSession
		err      error
	}

	testCases := []struct {
		name string

		storeCall *storeCall

		expectedDimensions *Dimensions
		expectedErr        error
	}{
		{
			name: "success with unique dimensions",
			storeCall: &storeCall{
				sessions: testSessions,
			},
			expectedDimensions: &Dimensions{
				SeriesIDs: []int64{42, 43},
				CarIDs:    []int64{10, 11},
				TrackIDs:  []int64{100, 101},
			},
		},
		{
			name: "empty sessions",
			storeCall: &storeCall{
				sessions: []store.DriverSession{},
			},
			expectedDimensions: &Dimensions{
				SeriesIDs: []int64{},
				CarIDs:    []int64{},
				TrackIDs:  []int64{},
			},
		},
		{
			name: "store error",
			storeCall: &storeCall{
				err: errors.New("database error"),
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)

			from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			to := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

			if tc.storeCall != nil {
				mockStore.EXPECT().GetDriverSessionsByTimeRange(mock.Anything, int64(12345), from, to).
					Return(tc.storeCall.sessions, tc.storeCall.err)
			}

			svc := NewService(mockStore)
			dims, err := svc.GetDimensions(context.Background(), 12345, from, to)

			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedDimensions, dims)
		})
	}
}

func TestService_GetAnalytics(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	testSessions := []store.DriverSession{
		{
			SeriesID:       42,
			CarID:          10,
			TrackID:        100,
			StartTime:      baseTime,
			OldIRating:     1500,
			NewIRating:     1550,
			OldCPI:         3.0,
			NewCPI:         3.1,
			StartPosition:  5,
			FinishPosition: 2,
			Incidents:      2,
		},
		{
			SeriesID:       42,
			CarID:          11,
			TrackID:        101,
			StartTime:      baseTime.Add(24 * time.Hour),
			OldIRating:     1550,
			NewIRating:     1520,
			OldCPI:         3.1,
			NewCPI:         2.9,
			StartPosition:  3,
			FinishPosition: 8,
			Incidents:      4,
		},
		{
			SeriesID:       43,
			CarID:          10,
			TrackID:        100,
			StartTime:      baseTime.Add(48 * time.Hour),
			OldIRating:     1520,
			NewIRating:     1600,
			OldCPI:         2.9,
			NewCPI:         3.2,
			StartPosition:  10,
			FinishPosition: 0, // 0-based: 0 = 1st place (win)
			Incidents:      0,
		},
	}

	type storeCall struct {
		sessions []store.DriverSession
		err      error
	}

	testCases := []struct {
		name string

		request   AnalyticsRequest
		storeCall *storeCall

		expectedRaceCount    int
		expectedIRatingStart int
		expectedIRatingEnd   int
		expectedIRatingDelta int
		expectedWins         int
		expectedPodiums      int
		expectedGroupCount   int
		expectedTimeSeriesCount int
		expectedErr          error
	}{
		{
			name: "basic summary",
			request: AnalyticsRequest{
				DriverID: 12345,
				From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			storeCall: &storeCall{
				sessions: testSessions,
			},
			expectedRaceCount:    3,
			expectedIRatingStart: 1500,
			expectedIRatingEnd:   1600,
			expectedIRatingDelta: 100,
			expectedWins:         1,
			expectedPodiums:      2,
		},
		{
			name: "with groupBy series",
			request: AnalyticsRequest{
				DriverID: 12345,
				From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
				GroupBy:  []GroupByDimension{GroupBySeries},
			},
			storeCall: &storeCall{
				sessions: testSessions,
			},
			expectedRaceCount:  3,
			expectedGroupCount: 2, // series 42 and 43
		},
		{
			name: "with granularity day",
			request: AnalyticsRequest{
				DriverID:    12345,
				From:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:          time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
				Granularity: GranularityDay,
			},
			storeCall: &storeCall{
				sessions: testSessions,
			},
			expectedRaceCount:       3,
			expectedTimeSeriesCount: 3, // 3 different days
		},
		{
			name: "with filter by series",
			request: AnalyticsRequest{
				DriverID:  12345,
				From:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:        time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
				SeriesIDs: []int64{42},
			},
			storeCall: &storeCall{
				sessions: testSessions,
			},
			expectedRaceCount:    2, // only series 42
			expectedIRatingStart: 1500,
			expectedIRatingEnd:   1520,
		},
		{
			name: "empty sessions",
			request: AnalyticsRequest{
				DriverID: 12345,
				From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			storeCall: &storeCall{
				sessions: []store.DriverSession{},
			},
			expectedRaceCount: 0,
		},
		{
			name: "store error",
			request: AnalyticsRequest{
				DriverID: 12345,
				From:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				To:       time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			storeCall: &storeCall{
				err: errors.New("database error"),
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)

			if tc.storeCall != nil {
				mockStore.EXPECT().GetDriverSessionsByTimeRange(mock.Anything, tc.request.DriverID, tc.request.From, tc.request.To).
					Return(tc.storeCall.sessions, tc.storeCall.err)
			}

			svc := NewService(mockStore)
			result, err := svc.GetAnalytics(context.Background(), tc.request)

			if tc.expectedErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedRaceCount, result.Summary.RaceCount)

			if tc.expectedIRatingStart != 0 {
				assert.Equal(t, tc.expectedIRatingStart, result.Summary.IRatingStart)
			}
			if tc.expectedIRatingEnd != 0 {
				assert.Equal(t, tc.expectedIRatingEnd, result.Summary.IRatingEnd)
			}
			if tc.expectedIRatingDelta != 0 {
				assert.Equal(t, tc.expectedIRatingDelta, result.Summary.IRatingDelta)
			}
			if tc.expectedWins != 0 {
				assert.Equal(t, tc.expectedWins, result.Summary.Wins)
			}
			if tc.expectedPodiums != 0 {
				assert.Equal(t, tc.expectedPodiums, result.Summary.Podiums)
			}
			if tc.expectedGroupCount != 0 {
				assert.Len(t, result.GroupedBy, tc.expectedGroupCount)
			}
			if tc.expectedTimeSeriesCount != 0 {
				assert.Len(t, result.TimeSeries, tc.expectedTimeSeriesCount)
			}
		})
	}
}

func TestComputeSummary(t *testing.T) {
	testCases := []struct {
		name     string
		sessions []store.DriverSession
		expected Summary
	}{
		{
			name:     "empty sessions",
			sessions: []store.DriverSession{},
			expected: Summary{},
		},
		{
			name: "single session",
			sessions: []store.DriverSession{
				{
					OldIRating:     1500,
					NewIRating:     1550,
					OldCPI:         3.0,
					NewCPI:         3.2,
					StartPosition:  5,
					FinishPosition: 2,
					Incidents:      3,
				},
			},
			expected: Summary{
				RaceCount:         1,
				IRatingStart:      1500,
				IRatingEnd:        1550,
				IRatingDelta:      50,
				IRatingGain:       50,
				IRatingLoss:       0,
				CPIStart:          3.0,
				CPIEnd:            3.2,
				CPIDelta:          0.2,
				CPIGain:           0.2,
				CPILoss:           0,
				Podiums:           1,
				Top5Finishes:      1,
				Wins:              0,
				AvgFinishPosition: 2.0,
				AvgStartPosition:  5.0,
				PositionsGained:   3.0,
				TotalIncidents:    3,
				AvgIncidents:      3.0,
			},
		},
		{
			name: "win counts correctly",
			sessions: []store.DriverSession{
				{
					OldIRating:     1500,
					NewIRating:     1600,
					StartPosition:  3,
					FinishPosition: 0, // 0-based: 0 = 1st place (win)
				},
			},
			expected: Summary{
				RaceCount:         1,
				IRatingStart:      1500,
				IRatingEnd:        1600,
				IRatingDelta:      100,
				IRatingGain:       100,
				Podiums:           1,
				Top5Finishes:      1,
				Wins:              1,
				AvgFinishPosition: 0.0,
				AvgStartPosition:  3.0,
				PositionsGained:   3.0,
			},
		},
		{
			name: "loss tracking",
			sessions: []store.DriverSession{
				{
					OldIRating:     1600,
					NewIRating:     1500,
					OldCPI:         3.5,
					NewCPI:         3.0,
					StartPosition:  5,
					FinishPosition: 10,
				},
			},
			expected: Summary{
				RaceCount:         1,
				IRatingStart:      1600,
				IRatingEnd:        1500,
				IRatingDelta:      -100,
				IRatingGain:       0,
				IRatingLoss:       100,
				CPIStart:          3.5,
				CPIEnd:            3.0,
				CPIDelta:          -0.5,
				CPIGain:           0,
				CPILoss:           0.5,
				AvgFinishPosition: 10.0,
				AvgStartPosition:  5.0,
				PositionsGained:   -5.0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := computeSummary(tc.sessions)
			assert.InDelta(t, tc.expected.RaceCount, result.RaceCount, 0)
			assert.InDelta(t, tc.expected.IRatingStart, result.IRatingStart, 0)
			assert.InDelta(t, tc.expected.IRatingEnd, result.IRatingEnd, 0)
			assert.InDelta(t, tc.expected.IRatingDelta, result.IRatingDelta, 0)
			assert.InDelta(t, tc.expected.IRatingGain, result.IRatingGain, 0)
			assert.InDelta(t, tc.expected.IRatingLoss, result.IRatingLoss, 0)
			assert.InDelta(t, tc.expected.CPIStart, result.CPIStart, 0.001)
			assert.InDelta(t, tc.expected.CPIEnd, result.CPIEnd, 0.001)
			assert.InDelta(t, tc.expected.CPIDelta, result.CPIDelta, 0.001)
			assert.InDelta(t, tc.expected.CPIGain, result.CPIGain, 0.001)
			assert.InDelta(t, tc.expected.CPILoss, result.CPILoss, 0.001)
			assert.InDelta(t, tc.expected.Podiums, result.Podiums, 0)
			assert.InDelta(t, tc.expected.Top5Finishes, result.Top5Finishes, 0)
			assert.InDelta(t, tc.expected.Wins, result.Wins, 0)
			assert.InDelta(t, tc.expected.AvgFinishPosition, result.AvgFinishPosition, 0.001)
			assert.InDelta(t, tc.expected.AvgStartPosition, result.AvgStartPosition, 0.001)
			assert.InDelta(t, tc.expected.PositionsGained, result.PositionsGained, 0.001)
			assert.InDelta(t, tc.expected.TotalIncidents, result.TotalIncidents, 0)
			assert.InDelta(t, tc.expected.AvgIncidents, result.AvgIncidents, 0.001)
		})
	}
}

func TestFilterSessions(t *testing.T) {
	sessions := []store.DriverSession{
		{SeriesID: 42, CarID: 10, TrackID: 100},
		{SeriesID: 42, CarID: 11, TrackID: 101},
		{SeriesID: 43, CarID: 10, TrackID: 100},
		{SeriesID: 44, CarID: 12, TrackID: 102},
	}

	testCases := []struct {
		name      string
		seriesIDs []int64
		carIDs    []int64
		trackIDs  []int64
		expected  int
	}{
		{
			name:     "no filters",
			expected: 4,
		},
		{
			name:      "filter by single series",
			seriesIDs: []int64{42},
			expected:  2,
		},
		{
			name:      "filter by multiple series (OR)",
			seriesIDs: []int64{42, 43},
			expected:  3,
		},
		{
			name:     "filter by car",
			carIDs:   []int64{10},
			expected: 2,
		},
		{
			name:     "filter by track",
			trackIDs: []int64{100},
			expected: 2,
		},
		{
			name:      "filter by series AND car (cross-dimension)",
			seriesIDs: []int64{42},
			carIDs:    []int64{10},
			expected:  1,
		},
		{
			name:      "filter with no matches",
			seriesIDs: []int64{999},
			expected:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterSessions(sessions, tc.seriesIDs, tc.carIDs, tc.trackIDs)
			assert.Len(t, result, tc.expected)
		})
	}
}

func TestFormatPeriod(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	testCases := []struct {
		name        string
		granularity Granularity
		expected    string
	}{
		{
			name:        "day",
			granularity: GranularityDay,
			expected:    "2024-01-15",
		},
		{
			name:        "week",
			granularity: GranularityWeek,
			expected:    "2024-W03",
		},
		{
			name:        "month",
			granularity: GranularityMonth,
			expected:    "2024-01",
		},
		{
			name:        "year",
			granularity: GranularityYear,
			expected:    "2024",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatPeriod(testTime, tc.granularity)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGranularity_IsValid(t *testing.T) {
	testCases := []struct {
		granularity Granularity
		expected    bool
	}{
		{GranularityDay, true},
		{GranularityWeek, true},
		{GranularityMonth, true},
		{GranularityYear, true},
		{Granularity("invalid"), false},
		{Granularity(""), false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.granularity), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.granularity.IsValid())
		})
	}
}

func TestGroupByDimension_IsValid(t *testing.T) {
	testCases := []struct {
		dimension GroupByDimension
		expected  bool
	}{
		{GroupBySeries, true},
		{GroupByCar, true},
		{GroupByTrack, true},
		{GroupByDimension("invalid"), false},
		{GroupByDimension(""), false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.dimension), func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.dimension.IsValid())
		})
	}
}