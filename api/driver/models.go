package driver

import (
	"time"

	"github.com/jonsabados/saturdaysspinout/store"
)

type DriverInfo struct {
	DriverID              int64      `json:"driverId"`
	DriverName            string     `json:"driverName"`
	MemberSince           time.Time  `json:"memberSince"`
	RacesIngestedTo       *time.Time `json:"racesIngestedTo"`
	IngestionBlockedUntil *time.Time `json:"ingestionBlockedUntil"`
	FirstLogin            time.Time  `json:"firstLogin"`
	LastLogin             time.Time  `json:"lastLogin"`
	LoginCount            int64      `json:"loginCount"`
	SessionCount          int64      `json:"sessionCount"`
}

func driverInfoFromDriver(driver store.Driver) DriverInfo {
	info := DriverInfo{
		DriverID:     driver.DriverID,
		DriverName:   driver.DriverName,
		MemberSince:  driver.MemberSince.UTC(),
		FirstLogin:   driver.FirstLogin.UTC(),
		LastLogin:    driver.LastLogin.UTC(),
		LoginCount:   driver.LoginCount,
		SessionCount: driver.SessionCount,
	}
	if driver.RacesIngestedTo != nil {
		t := driver.RacesIngestedTo.UTC()
		info.RacesIngestedTo = &t
	}
	if driver.IngestionBlockedUntil != nil {
		t := driver.IngestionBlockedUntil.UTC()
		info.IngestionBlockedUntil = &t
	}
	return info
}

type Race struct {
	ID                    int64     `json:"id"`
	SubsessionID          int64     `json:"subsessionId"`
	TrackID               int64     `json:"trackId"`
	CarID                 int64     `json:"carId"`
	StartTime             time.Time `json:"startTime"`
	StartPosition         int       `json:"startPosition"`
	StartPositionInClass  int       `json:"startPositionInClass"`
	FinishPosition        int       `json:"finishPosition"`
	FinishPositionInClass int       `json:"finishPositionInClass"`
	Incidents             int       `json:"incidents"`
	OldCPI                float64   `json:"oldCpi"`
	NewCPI                float64   `json:"newCpi"`
	OldIRating            int       `json:"oldIrating"`
	NewIRating            int       `json:"newIrating"`
	ReasonOut             string    `json:"reasonOut"`
}

func raceFromDriverSession(session store.DriverSession) Race {
	return Race{
		ID:                    session.StartTime.Unix(),
		SubsessionID:          session.SubsessionID,
		TrackID:               session.TrackID,
		CarID:                 session.CarID,
		StartTime:             session.StartTime.UTC(),
		StartPosition:         session.StartPosition,
		StartPositionInClass:  session.StartPositionInClass,
		FinishPosition:        session.FinishPosition,
		FinishPositionInClass: session.FinishPositionInClass,
		Incidents:             session.Incidents,
		OldCPI:                session.OldCPI,
		NewCPI:                session.NewCPI,
		OldIRating:            session.OldIRating,
		NewIRating:            session.NewIRating,
		ReasonOut:             session.ReasonOut,
	}
}
