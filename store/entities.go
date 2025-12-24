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

type Session struct {
	SubsessionID    int64
	TrackID         int64
	SeriesID        int64
	SeriesName      string
	LicenseCategory string
	StartTime       time.Time
	CarClasses      []SessionCarClass
}

type SessionCarClass struct {
	SubsessionID    int64
	CarClassID      int64
	StrengthOfField int
	NumberOfEntries int
	Cars            []SessionCarClassCar
}

type SessionCarClassCar struct {
	SubsessionID int64
	CarClassID   int64
	CarID        int64
}

type SessionDriver struct {
	SubsessionID          int64
	DriverID              int64
	CarID                 int64
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
	AI                    bool
}

type SessionDriverLap struct {
	SubsessionID int64
	DriverID     int64
	LapNumber    int
	LapTime      time.Duration
	Flags        int
	Incident     bool
	LapEvents    []string
}

type GlobalCounters struct {
	Drivers  int64
	Notes    int64
	Sessions int64
	Laps     int64
}

type WebSocketConnection struct {
	DriverID     int64
	ConnectionID string
	ConnectedAt  time.Time
}

type SessionDataInsertion struct {
	SessionEntries          []Session
	SessionDriverEntries    []SessionDriver
	SessionDriverLapEntries []SessionDriverLap
	DriverSessionEntries    []DriverSession
}

func (s SessionDataInsertion) HasData() bool {
	return len(s.SessionEntries) > 0 || len(s.SessionDriverEntries) > 0 || len(s.SessionDriverLapEntries) > 0 || len(s.DriverSessionEntries) > 0
}
