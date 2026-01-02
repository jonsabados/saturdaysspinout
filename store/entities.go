package store

import (
	"errors"
	"time"
)

var ErrEntityAlreadyExists = errors.New("entity already exists")

type Driver struct {
	DriverID              int64
	DriverName            string
	MemberSince           time.Time
	RacesIngestedTo       *time.Time
	IngestionBlockedUntil *time.Time
	FirstLogin            time.Time
	LastLogin             time.Time
	LoginCount            int64
	SessionCount          int64
	Entitlements          []string
}

// DriverSession represents drivers records of sessions (for use in list views of races)
type DriverSession struct {
	DriverID              int64
	SubsessionID          int64
	TrackID               int64
	CarID                 int64
	SeriesID              int64
	SeriesName            string
	StartTime             time.Time
	StartPosition         int
	StartPositionInClass  int
	FinishPosition        int
	FinishPositionInClass int
	Incidents             int
	OldCPI                float64
	NewCPI                float64
	OldIRating            int
	NewIRating            int
	OldLicenseLevel       int
	NewLicenseLevel       int
	OldSubLevel           int
	NewSubLevel           int
	ReasonOut             string
}

// RaceJournalEntry represents a user's journal entry for a specific race.
// Race context is fetched separately via DriverSession and joined at query time.
type RaceJournalEntry struct {
	DriverID  int64
	RaceID    int64 // driver_race_id (unix timestamp of race start)
	CreatedAt time.Time
	UpdatedAt time.Time

	// User-provided content
	Notes string
	Tags  []string // Unified tags: plain ("podium") or key:value ("sentiment:good")
}

type GlobalCounters struct {
	Drivers int64
}

type WebSocketConnection struct {
	DriverID     int64
	ConnectionID string
	ConnectedAt  time.Time
}

