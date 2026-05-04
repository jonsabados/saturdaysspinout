package store

// SessionFilter narrows a slice of DriverSession to those matching some criteria.
// Filters are composed by applying them in sequence, so multiple filters AND together.
type SessionFilter func(sessions []DriverSession) []DriverSession

// FilterBySeriesIDs returns a SessionFilter that keeps sessions whose SeriesID is in seriesIDs (OR within).
// Panics if seriesIDs is empty: an empty filter is meaningless — callers must decide explicitly whether to filter.
func FilterBySeriesIDs(seriesIDs []int64) SessionFilter {
	if len(seriesIDs) == 0 {
		panic("store.FilterBySeriesIDs: seriesIDs must not be empty")
	}
	allowed := int64Set(seriesIDs)
	return func(sessions []DriverSession) []DriverSession {
		filtered := make([]DriverSession, 0, len(sessions))
		for _, session := range sessions {
			if _, ok := allowed[session.SeriesID]; ok {
				filtered = append(filtered, session)
			}
		}
		return filtered
	}
}

// FilterByCarIDs returns a SessionFilter that keeps sessions whose CarID is in carIDs (OR within).
// Panics if carIDs is empty: an empty filter is meaningless — callers must decide explicitly whether to filter.
func FilterByCarIDs(carIDs []int64) SessionFilter {
	if len(carIDs) == 0 {
		panic("store.FilterByCarIDs: carIDs must not be empty")
	}
	allowed := int64Set(carIDs)
	return func(sessions []DriverSession) []DriverSession {
		filtered := make([]DriverSession, 0, len(sessions))
		for _, session := range sessions {
			if _, ok := allowed[session.CarID]; ok {
				filtered = append(filtered, session)
			}
		}
		return filtered
	}
}

// FilterByTrackIDs returns a SessionFilter that keeps sessions whose TrackID is in trackIDs (OR within).
// Panics if trackIDs is empty: an empty filter is meaningless — callers must decide explicitly whether to filter.
func FilterByTrackIDs(trackIDs []int64) SessionFilter {
	if len(trackIDs) == 0 {
		panic("store.FilterByTrackIDs: trackIDs must not be empty")
	}
	allowed := int64Set(trackIDs)
	return func(sessions []DriverSession) []DriverSession {
		filtered := make([]DriverSession, 0, len(sessions))
		for _, session := range sessions {
			if _, ok := allowed[session.TrackID]; ok {
				filtered = append(filtered, session)
			}
		}
		return filtered
	}
}

func int64Set(ids []int64) map[int64]struct{} {
	set := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}