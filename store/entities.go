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
	Tracks int64
	Notes  int64
}
