package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDriverRaceIDFromTime(t *testing.T) {
	t.Run("converts time to unix timestamp", func(t *testing.T) {
		input := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
		result := DriverRaceIDFromTime(input)
		assert.Equal(t, input.Unix(), result)
	})

	t.Run("handles zero time", func(t *testing.T) {
		input := time.Time{}
		result := DriverRaceIDFromTime(input)
		assert.Equal(t, input.Unix(), result)
	})

	t.Run("handles unix epoch", func(t *testing.T) {
		input := time.Unix(0, 0).UTC()
		result := DriverRaceIDFromTime(input)
		assert.Equal(t, int64(0), result)
	})

	t.Run("truncates nanoseconds", func(t *testing.T) {
		withNanos := time.Date(2024, 6, 15, 14, 30, 0, 999999999, time.UTC)
		withoutNanos := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

		resultWithNanos := DriverRaceIDFromTime(withNanos)
		resultWithoutNanos := DriverRaceIDFromTime(withoutNanos)

		assert.Equal(t, resultWithoutNanos, resultWithNanos, "nanoseconds should not affect the result")
	})
}

func TestTimeFromDriverRaceID(t *testing.T) {
	t.Run("converts unix timestamp to time", func(t *testing.T) {
		input := int64(1718461800)
		result := TimeFromDriverRaceID(input)
		assert.Equal(t, input, result.Unix())
	})

	t.Run("handles zero", func(t *testing.T) {
		result := TimeFromDriverRaceID(0)
		assert.Equal(t, int64(0), result.Unix())
	})

	t.Run("handles negative timestamp", func(t *testing.T) {
		input := int64(-86400)
		result := TimeFromDriverRaceID(input)
		assert.Equal(t, input, result.Unix())
	})
}

func TestRoundTrip(t *testing.T) {
	original := time.Date(2024, 12, 15, 20, 0, 0, 0, time.UTC)

	raceID := DriverRaceIDFromTime(original)
	recovered := TimeFromDriverRaceID(raceID)

	assert.True(t, original.Equal(recovered), "round trip failed: original %v, recovered %v", original, recovered)
}

func TestRoundTripTruncatesNanoseconds(t *testing.T) {
	original := time.Date(2024, 12, 15, 20, 0, 0, 500000000, time.UTC)
	expectedAfterRoundTrip := time.Date(2024, 12, 15, 20, 0, 0, 0, time.UTC)

	raceID := DriverRaceIDFromTime(original)
	recovered := TimeFromDriverRaceID(raceID)

	assert.False(t, original.Equal(recovered), "nanoseconds should be lost in round trip")
	assert.True(t, expectedAfterRoundTrip.Equal(recovered), "recovered time should have zero nanoseconds")
}