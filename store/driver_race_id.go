package store

import "time"

// DriverRaceIDFromTime converts a race start time to the external driver race ID.
// Driver race IDs are scoped to a specific driver - different drivers may have
// different races with the same ID.
func DriverRaceIDFromTime(t time.Time) int64 {
	return t.Unix()
}

// TimeFromDriverRaceID converts an external driver race ID back to a time.Time.
func TimeFromDriverRaceID(raceID int64) time.Time {
	return time.Unix(raceID, 0)
}
