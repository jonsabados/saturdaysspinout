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

// AnalyticsSummary contains aggregated statistics for a set of races.
type AnalyticsSummary struct {
	RaceCount int `json:"raceCount"`

	// iRating
	IRatingStart int `json:"iRatingStart"` // first race in range
	IRatingEnd   int `json:"iRatingEnd"`   // last race in range
	IRatingDelta int `json:"iRatingDelta"` // net change
	IRatingGain  int `json:"iRatingGain"`  // sum of positive deltas
	IRatingLoss  int `json:"iRatingLoss"`  // sum of negative deltas (as positive number)

	// CPI (Compliance Points Index) - frontend formats as SR for display
	CPIStart float64 `json:"cpiStart"` // CPI at first race
	CPIEnd   float64 `json:"cpiEnd"`   // CPI at last race
	CPIDelta float64 `json:"cpiDelta"` // net CPI change
	CPIGain  float64 `json:"cpiGain"`  // sum of positive CPI deltas
	CPILoss  float64 `json:"cpiLoss"`  // sum of negative CPI deltas

	// Position stats
	Podiums           int     `json:"podiums"`           // finish <= 3
	Top5Finishes      int     `json:"top5Finishes"`      // finish <= 5
	Wins              int     `json:"wins"`              // finish == 1
	AvgFinishPosition float64 `json:"avgFinishPosition"` // average finish position
	AvgStartPosition  float64 `json:"avgStartPosition"`  // average qualifying position
	PositionsGained   float64 `json:"positionsGained"`   // avg (start - finish), positive = gained

	// Incidents
	TotalIncidents int     `json:"totalIncidents"`
	AvgIncidents   float64 `json:"avgIncidents"`
}

// AnalyticsGroup represents aggregated stats for a specific dimension combination.
type AnalyticsGroup struct {
	// Dimension identifiers - populated based on groupBy dimensions
	// Frontend uses reference endpoints (/series, /cars, /tracks) for names
	SeriesID *int64 `json:"seriesId,omitempty"`
	CarID    *int64 `json:"carId,omitempty"`
	TrackID  *int64 `json:"trackId,omitempty"`

	Summary AnalyticsSummary `json:"summary"`
}

// AnalyticsPeriod represents aggregated stats for a time period.
type AnalyticsPeriod struct {
	Period  string           `json:"period"` // Format based on granularity: "2024-01-15", "2024-W03", "2024-01", "2024"
	Summary AnalyticsSummary `json:"summary"`
}

// AnalyticsResponse is the response for the analytics endpoint.
type AnalyticsResponse struct {
	Summary    AnalyticsSummary  `json:"summary"`
	GroupedBy  []AnalyticsGroup  `json:"groupedBy,omitempty"`  // if groupBy specified
	TimeSeries []AnalyticsPeriod `json:"timeSeries,omitempty"` // if granularity specified
}

// DimensionsResponse is the response for the dimensions endpoint.
// Returns IDs only - frontend uses reference endpoints (/series, /cars, /tracks) for details.
type DimensionsResponse struct {
	Series []int64 `json:"series"`
	Cars   []int64 `json:"cars"`
	Tracks []int64 `json:"tracks"`
}
