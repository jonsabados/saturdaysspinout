package analytics

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jonsabados/saturdaysspinout/store"
)

// Store defines the data access interface needed by the analytics service.
type Store interface {
	GetDriverSessionsByTimeRange(ctx context.Context, driverID int64, from, to time.Time) ([]store.DriverSession, error)
}

// Dimensions contains the unique series, cars, and tracks a driver has raced.
type Dimensions struct {
	SeriesIDs []int64
	CarIDs    []int64
	TrackIDs  []int64
}

// Service provides analytics computations over race data.
type Service struct {
	store Store
}

// NewService creates a new analytics service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// GetDimensions returns the unique series, cars, and tracks the driver has raced
// within the specified time range.
func (s *Service) GetDimensions(ctx context.Context, driverID int64, from, to time.Time) (*Dimensions, error) {
	sessions, err := s.store.GetDriverSessionsByTimeRange(ctx, driverID, from, to)
	if err != nil {
		return nil, err
	}

	seriesSet := make(map[int64]struct{})
	carSet := make(map[int64]struct{})
	trackSet := make(map[int64]struct{})

	for _, session := range sessions {
		seriesSet[session.SeriesID] = struct{}{}
		carSet[session.CarID] = struct{}{}
		trackSet[session.TrackID] = struct{}{}
	}

	dims := &Dimensions{
		SeriesIDs: make([]int64, 0, len(seriesSet)),
		CarIDs:    make([]int64, 0, len(carSet)),
		TrackIDs:  make([]int64, 0, len(trackSet)),
	}

	for id := range seriesSet {
		dims.SeriesIDs = append(dims.SeriesIDs, id)
	}
	for id := range carSet {
		dims.CarIDs = append(dims.CarIDs, id)
	}
	for id := range trackSet {
		dims.TrackIDs = append(dims.TrackIDs, id)
	}

	// Sort by ID for consistent ordering
	sort.Slice(dims.SeriesIDs, func(i, j int) bool { return dims.SeriesIDs[i] < dims.SeriesIDs[j] })
	sort.Slice(dims.CarIDs, func(i, j int) bool { return dims.CarIDs[i] < dims.CarIDs[j] })
	sort.Slice(dims.TrackIDs, func(i, j int) bool { return dims.TrackIDs[i] < dims.TrackIDs[j] })

	return dims, nil
}

// Summary contains aggregated statistics for a set of races.
type Summary struct {
	RaceCount int

	// iRating
	IRatingStart int
	IRatingEnd   int
	IRatingDelta int
	IRatingGain  int
	IRatingLoss  int

	// CPI (Compliance Points Index)
	CPIStart float64
	CPIEnd   float64
	CPIDelta float64
	CPIGain  float64
	CPILoss  float64

	// Position stats
	Podiums           int
	Top5Finishes      int
	Wins              int
	AvgFinishPosition float64
	AvgStartPosition  float64
	PositionsGained   float64

	// Incidents
	TotalIncidents int
	AvgIncidents   float64
}

// GroupedSummary contains stats for a specific dimension grouping.
type GroupedSummary struct {
	SeriesID *int64
	CarID    *int64
	TrackID  *int64
	Summary  Summary
}

// PeriodSummary contains stats for a time period.
type PeriodSummary struct {
	Period  string
	Summary Summary
}

// Granularity represents the time bucketing level for time series data.
type Granularity string

const (
	GranularityDay   Granularity = "day"
	GranularityWeek  Granularity = "week"
	GranularityMonth Granularity = "month"
	GranularityYear  Granularity = "year"
)

// IsValid checks if the granularity value is valid.
func (g Granularity) IsValid() bool {
	switch g {
	case GranularityDay, GranularityWeek, GranularityMonth, GranularityYear:
		return true
	}
	return false
}

// GroupByDimension represents a dimension to group results by.
type GroupByDimension string

const (
	GroupBySeries GroupByDimension = "series"
	GroupByCar    GroupByDimension = "car"
	GroupByTrack  GroupByDimension = "track"
)

// IsValid checks if the groupBy dimension value is valid.
func (g GroupByDimension) IsValid() bool {
	switch g {
	case GroupBySeries, GroupByCar, GroupByTrack:
		return true
	}
	return false
}

// AnalyticsRequest contains the parameters for an analytics query.
type AnalyticsRequest struct {
	DriverID    int64
	From        time.Time
	To          time.Time
	GroupBy     []GroupByDimension
	Granularity Granularity
	SeriesIDs   []int64
	CarIDs      []int64
	TrackIDs    []int64
}

// AnalyticsResult contains the computed analytics.
type AnalyticsResult struct {
	Summary    Summary
	GroupedBy  []GroupedSummary
	TimeSeries []PeriodSummary
}

// GetAnalytics computes analytics for the given request.
func (s *Service) GetAnalytics(ctx context.Context, req AnalyticsRequest) (*AnalyticsResult, error) {
	sessions, err := s.store.GetDriverSessionsByTimeRange(ctx, req.DriverID, req.From, req.To)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := filterSessions(sessions, req.SeriesIDs, req.CarIDs, req.TrackIDs)

	// Sort by start time for chronological processing
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].StartTime.Before(filtered[j].StartTime)
	})

	result := &AnalyticsResult{
		Summary: computeSummary(filtered),
	}

	// Compute grouped stats if groupBy specified
	if len(req.GroupBy) > 0 {
		result.GroupedBy = computeGroupedStats(filtered, req.GroupBy)
	}

	// Compute time series if granularity specified
	if req.Granularity != "" && req.Granularity.IsValid() {
		result.TimeSeries = computeTimeSeries(filtered, req.Granularity)
	}

	return result, nil
}

func filterSessions(sessions []store.DriverSession, seriesIDs, carIDs, trackIDs []int64) []store.DriverSession {
	if len(seriesIDs) == 0 && len(carIDs) == 0 && len(trackIDs) == 0 {
		return sessions
	}

	seriesSet := int64SetFromSlice(seriesIDs)
	carSet := int64SetFromSlice(carIDs)
	trackSet := int64SetFromSlice(trackIDs)

	var filtered []store.DriverSession
	for _, session := range sessions {
		// AND across dimensions, OR within dimension
		if len(seriesSet) > 0 {
			if _, ok := seriesSet[session.SeriesID]; !ok {
				continue
			}
		}
		if len(carSet) > 0 {
			if _, ok := carSet[session.CarID]; !ok {
				continue
			}
		}
		if len(trackSet) > 0 {
			if _, ok := trackSet[session.TrackID]; !ok {
				continue
			}
		}
		filtered = append(filtered, session)
	}
	return filtered
}

func int64SetFromSlice(ids []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func computeSummary(sessions []store.DriverSession) Summary {
	if len(sessions) == 0 {
		return Summary{}
	}

	summary := Summary{
		RaceCount:    len(sessions),
		IRatingStart: sessions[0].OldIRating,
		IRatingEnd:   sessions[len(sessions)-1].NewIRating,
		CPIStart:     sessions[0].OldCPI,
		CPIEnd:       sessions[len(sessions)-1].NewCPI,
	}

	var totalFinishPos, totalStartPos int
	var positionsGainedSum int

	for _, session := range sessions {
		// iRating changes
		irDelta := session.NewIRating - session.OldIRating
		if irDelta > 0 {
			summary.IRatingGain += irDelta
		} else {
			summary.IRatingLoss += -irDelta // store as positive
		}

		// CPI changes
		cpiDelta := session.NewCPI - session.OldCPI
		if cpiDelta > 0 {
			summary.CPIGain += cpiDelta
		} else {
			summary.CPILoss += -cpiDelta // store as positive
		}

		// Position stats
		if session.FinishPosition <= 3 {
			summary.Podiums++
		}
		if session.FinishPosition <= 5 {
			summary.Top5Finishes++
		}
		if session.FinishPosition == 1 {
			summary.Wins++
		}

		totalFinishPos += session.FinishPosition
		totalStartPos += session.StartPosition
		positionsGainedSum += session.StartPosition - session.FinishPosition

		// Incidents
		summary.TotalIncidents += session.Incidents
	}

	summary.IRatingDelta = summary.IRatingEnd - summary.IRatingStart
	summary.CPIDelta = summary.CPIEnd - summary.CPIStart
	summary.AvgFinishPosition = float64(totalFinishPos) / float64(len(sessions))
	summary.AvgStartPosition = float64(totalStartPos) / float64(len(sessions))
	summary.PositionsGained = float64(positionsGainedSum) / float64(len(sessions))
	summary.AvgIncidents = float64(summary.TotalIncidents) / float64(len(sessions))

	return summary
}

type groupKey struct {
	seriesID *int64
	carID    *int64
	trackID  *int64
}

func computeGroupedStats(sessions []store.DriverSession, groupBy []GroupByDimension) []GroupedSummary {
	groups := make(map[string][]store.DriverSession)
	keyMap := make(map[string]groupKey)

	for _, session := range sessions {
		key := groupKey{}
		keyStr := ""

		for _, dim := range groupBy {
			switch dim {
			case GroupBySeries:
				id := session.SeriesID
				key.seriesID = &id
				keyStr += fmt.Sprintf("s:%d|", id)
			case GroupByCar:
				id := session.CarID
				key.carID = &id
				keyStr += fmt.Sprintf("c:%d|", id)
			case GroupByTrack:
				id := session.TrackID
				key.trackID = &id
				keyStr += fmt.Sprintf("t:%d|", id)
			}
		}

		groups[keyStr] = append(groups[keyStr], session)
		keyMap[keyStr] = key
	}

	var results []GroupedSummary
	for keyStr, groupSessions := range groups {
		// Sort sessions within group by time
		sort.Slice(groupSessions, func(i, j int) bool {
			return groupSessions[i].StartTime.Before(groupSessions[j].StartTime)
		})

		key := keyMap[keyStr]
		results = append(results, GroupedSummary{
			SeriesID: key.seriesID,
			CarID:    key.carID,
			TrackID:  key.trackID,
			Summary:  computeSummary(groupSessions),
		})
	}

	// Sort by race count descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Summary.RaceCount > results[j].Summary.RaceCount
	})

	return results
}

func computeTimeSeries(sessions []store.DriverSession, granularity Granularity) []PeriodSummary {
	if len(sessions) == 0 {
		return nil
	}

	groups := make(map[string][]store.DriverSession)

	for _, session := range sessions {
		period := formatPeriod(session.StartTime, granularity)
		groups[period] = append(groups[period], session)
	}

	var results []PeriodSummary
	for period, groupSessions := range groups {
		// Sort sessions within period by time
		sort.Slice(groupSessions, func(i, j int) bool {
			return groupSessions[i].StartTime.Before(groupSessions[j].StartTime)
		})

		results = append(results, PeriodSummary{
			Period:  period,
			Summary: computeSummary(groupSessions),
		})
	}

	// Sort chronologically by period string (works for ISO formats)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Period < results[j].Period
	})

	return results
}

func formatPeriod(t time.Time, granularity Granularity) string {
	switch granularity {
	case GranularityDay:
		return t.Format("2006-01-02")
	case GranularityWeek:
		year, week := t.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case GranularityMonth:
		return t.Format("2006-01")
	case GranularityYear:
		return t.Format("2006")
	default:
		return t.Format("2006-01-02")
	}
}