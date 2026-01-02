package driver

import (
	"time"

	"github.com/jonsabados/saturdaysspinout/journal"
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
	SeriesID              int64     `json:"seriesId"`
	SeriesName            string    `json:"seriesName"`
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
	OldLicenseLevel       int       `json:"oldLicenseLevel"`
	NewLicenseLevel       int       `json:"newLicenseLevel"`
	OldSubLevel           int       `json:"oldSubLevel"`
	NewSubLevel           int       `json:"newSubLevel"`
	ReasonOut             string    `json:"reasonOut"`
}

func raceFromDriverSession(session store.DriverSession) Race {
	return Race{
		ID:                    session.StartTime.Unix(),
		SubsessionID:          session.SubsessionID,
		TrackID:               session.TrackID,
		SeriesID:              session.SeriesID,
		SeriesName:            session.SeriesName,
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
		OldLicenseLevel:       session.OldLicenseLevel,
		NewLicenseLevel:       session.NewLicenseLevel,
		OldSubLevel:           session.OldSubLevel,
		NewSubLevel:           session.NewSubLevel,
		ReasonOut:             session.ReasonOut,
	}
}

// SaveJournalEntryRequest is the request body for creating/updating a journal entry.
type SaveJournalEntryRequest struct {
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`
}

// JournalEntry is the API response model for a journal entry with joined race context.
type JournalEntry struct {
	RaceID    int64     `json:"raceId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Notes     string    `json:"notes"`
	Tags      []string  `json:"tags"`
	Race      *Race     `json:"race,omitempty"`
}

func journalEntryFromStore(entry store.RaceJournalEntry, session *store.DriverSession) JournalEntry {
	result := JournalEntry{
		RaceID:    entry.RaceID,
		CreatedAt: entry.CreatedAt.UTC(),
		UpdatedAt: entry.UpdatedAt.UTC(),
		Notes:     entry.Notes,
		Tags:      entry.Tags,
	}
	if result.Tags == nil {
		result.Tags = []string{}
	}
	if session != nil {
		race := raceFromDriverSession(*session)
		result.Race = &race
	}
	return result
}

func journalEntryFromServiceEntry(entry journal.Entry) JournalEntry {
	result := JournalEntry{
		RaceID:    entry.RaceID,
		CreatedAt: entry.CreatedAt.UTC(),
		UpdatedAt: entry.UpdatedAt.UTC(),
		Notes:     entry.Notes,
		Tags:      entry.Tags,
	}
	if result.Tags == nil {
		result.Tags = []string{}
	}
	if entry.Race != nil {
		race := raceFromDriverSession(*entry.Race)
		result.Race = &race
	}
	return result
}
