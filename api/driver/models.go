package driver

import (
	"time"

	"github.com/jonsabados/saturdaysspinout/store"
)

type Race struct {
	ID                    int64   `json:"id"`
	SubsessionID          int64   `json:"subsessionId"`
	TrackID               int64   `json:"trackId"`
	CarID                 int64   `json:"carId"`
	StartTime             string  `json:"startTime"`
	StartPosition         int     `json:"startPosition"`
	StartPositionInClass  int     `json:"startPositionInClass"`
	FinishPosition        int     `json:"finishPosition"`
	FinishPositionInClass int     `json:"finishPositionInClass"`
	Incidents             int     `json:"incidents"`
	OldCPI                float64 `json:"oldCpi"`
	NewCPI                float64 `json:"newCpi"`
	OldIRating            int     `json:"oldIrating"`
	NewIRating            int     `json:"newIrating"`
	ReasonOut             string  `json:"reasonOut"`
}

func raceFromDriverSession(session store.DriverSession) Race {
	return Race{
		ID:                    session.StartTime.Unix(),
		SubsessionID:          session.SubsessionID,
		TrackID:               session.TrackID,
		CarID:                 session.CarID,
		StartTime:             session.StartTime.UTC().Format(time.RFC3339),
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