package store

import (
	"errors"
	"time"
)

var ErrEntityAlreadyExists = errors.New("entity already exists")

type Track struct {
	ID   int64
	Name string
}

type Driver struct {
	DriverID        int64
	DriverName      string
	MemberSince     time.Time
	RacesIngestedTo *time.Time
	FirstLogin      time.Time
	LastLogin       time.Time
	LoginCount      int64
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

type GlobalCounters struct {
	Drivers int64
	Tracks  int64
	Notes   int64
}

type WebSocketConnection struct {
	DriverID     int64
	ConnectionID string
	ConnectedAt  time.Time
}
