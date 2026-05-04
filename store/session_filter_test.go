package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterBySeriesIDs(t *testing.T) {
	sessions := []DriverSession{
		{SeriesID: 42, CarID: 10, TrackID: 100},
		{SeriesID: 42, CarID: 11, TrackID: 101},
		{SeriesID: 43, CarID: 10, TrackID: 100},
		{SeriesID: 44, CarID: 12, TrackID: 102},
	}

	t.Run("single id", func(t *testing.T) {
		result := FilterBySeriesIDs([]int64{42})(sessions)
		assert.Len(t, result, 2)
		for _, s := range result {
			assert.Equal(t, int64(42), s.SeriesID)
		}
	})

	t.Run("multiple ids OR within", func(t *testing.T) {
		result := FilterBySeriesIDs([]int64{42, 43})(sessions)
		assert.Len(t, result, 3)
	})

	t.Run("no matches", func(t *testing.T) {
		result := FilterBySeriesIDs([]int64{999})(sessions)
		assert.Empty(t, result)
	})

	t.Run("panics on empty", func(t *testing.T) {
		assert.Panics(t, func() { FilterBySeriesIDs(nil) })
		assert.Panics(t, func() { FilterBySeriesIDs([]int64{}) })
	})
}

func TestFilterByCarIDs(t *testing.T) {
	sessions := []DriverSession{
		{SeriesID: 42, CarID: 10, TrackID: 100},
		{SeriesID: 42, CarID: 11, TrackID: 101},
		{SeriesID: 43, CarID: 10, TrackID: 100},
		{SeriesID: 44, CarID: 12, TrackID: 102},
	}

	t.Run("single id", func(t *testing.T) {
		result := FilterByCarIDs([]int64{10})(sessions)
		assert.Len(t, result, 2)
		for _, s := range result {
			assert.Equal(t, int64(10), s.CarID)
		}
	})

	t.Run("multiple ids OR within", func(t *testing.T) {
		result := FilterByCarIDs([]int64{10, 12})(sessions)
		assert.Len(t, result, 3)
	})

	t.Run("no matches", func(t *testing.T) {
		result := FilterByCarIDs([]int64{999})(sessions)
		assert.Empty(t, result)
	})

	t.Run("panics on empty", func(t *testing.T) {
		assert.Panics(t, func() { FilterByCarIDs(nil) })
		assert.Panics(t, func() { FilterByCarIDs([]int64{}) })
	})
}

func TestFilterByTrackIDs(t *testing.T) {
	sessions := []DriverSession{
		{SeriesID: 42, CarID: 10, TrackID: 100},
		{SeriesID: 42, CarID: 11, TrackID: 101},
		{SeriesID: 43, CarID: 10, TrackID: 100},
		{SeriesID: 44, CarID: 12, TrackID: 102},
	}

	t.Run("single id", func(t *testing.T) {
		result := FilterByTrackIDs([]int64{100})(sessions)
		assert.Len(t, result, 2)
		for _, s := range result {
			assert.Equal(t, int64(100), s.TrackID)
		}
	})

	t.Run("multiple ids OR within", func(t *testing.T) {
		result := FilterByTrackIDs([]int64{100, 102})(sessions)
		assert.Len(t, result, 3)
	})

	t.Run("no matches", func(t *testing.T) {
		result := FilterByTrackIDs([]int64{999})(sessions)
		assert.Empty(t, result)
	})

	t.Run("panics on empty", func(t *testing.T) {
		assert.Panics(t, func() { FilterByTrackIDs(nil) })
		assert.Panics(t, func() { FilterByTrackIDs([]int64{}) })
	})
}

func TestSessionFilters_ANDAcross(t *testing.T) {
	sessions := []DriverSession{
		{SeriesID: 42, CarID: 10, TrackID: 100},
		{SeriesID: 42, CarID: 11, TrackID: 101},
		{SeriesID: 43, CarID: 10, TrackID: 100},
		{SeriesID: 44, CarID: 12, TrackID: 102},
	}

	// Composing filters in sequence ANDs across dimensions: only sessions matching every filter survive.
	result := FilterBySeriesIDs([]int64{42})(sessions)
	result = FilterByCarIDs([]int64{10})(result)
	assert.Len(t, result, 1)
	assert.Equal(t, int64(42), result[0].SeriesID)
	assert.Equal(t, int64(10), result[0].CarID)
}