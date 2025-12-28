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

type DriverNote struct {
	DriverID  int64
	Timestamp time.Time
	SessionID int64
	LapNumber int64
	IsMistake bool
	Category  string
	Notes     string
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

type GlobalCounters struct {
	Drivers int64
}

type WebSocketConnection struct {
	DriverID     int64
	ConnectionID string
	ConnectedAt  time.Time
}

