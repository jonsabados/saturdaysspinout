package driver

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

type Store interface {
	GetDriverStore
	GetRacesStore
	GetRaceStore
	DeleteRacesStore
}

type JournalService interface {
	JournalServiceForSave
	GetJournalEntryStore
	ListJournalEntriesStore
	DeleteJournalEntryStore
}

func NewRouter(raceStore Store, journalService JournalService, analyticsService AnalyticsService, authMiddleware, developerMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Route(fmt.Sprintf("/{%s}", api.DriverIDPathParam), func(r chi.Router) {
		r.Use(api.DriverOwnershipMiddleware(api.DriverIDPathParam))

		r.Get("/", api.WrapWithSegment("getDriver", NewGetDriverEndpoint(raceStore)).ServeHTTP)
		r.Get("/races", api.WrapWithSegment("getDriverRaces", NewGetRacesEndpoint(raceStore)).ServeHTTP)
		r.Get("/races/{driver_race_id}", api.WrapWithSegment("getDriverRace", NewGetRaceEndpoint(raceStore)).ServeHTTP)
		r.Get("/races/{driver_race_id}/journal", api.WrapWithSegment("getJournalEntry", NewGetJournalEntryEndpoint(journalService)).ServeHTTP)
		r.Put("/races/{driver_race_id}/journal", api.WrapWithSegment("saveJournalEntry", NewSaveJournalEndpoint(journalService)).ServeHTTP)
		r.Delete("/races/{driver_race_id}/journal", api.WrapWithSegment("deleteJournalEntry", NewDeleteJournalEntryEndpoint(journalService)).ServeHTTP)
		r.Get("/journal", api.WrapWithSegment("listJournalEntries", NewListJournalEntriesEndpoint(journalService)).ServeHTTP)

		// Analytics endpoints
		r.Get("/analytics/dimensions", api.WrapWithSegment("getAnalyticsDimensions", NewAnalyticsDimensionsEndpoint(analyticsService)).ServeHTTP)
		r.Get("/analytics", api.WrapWithSegment("getAnalytics", NewAnalyticsEndpoint(analyticsService)).ServeHTTP)

		// Developer-only endpoints
		r.With(developerMiddleware).Delete("/races", api.WrapWithSegment("deleteDriverRaces", NewDeleteRacesEndpoint(raceStore)).ServeHTTP)
	})

	return r
}
