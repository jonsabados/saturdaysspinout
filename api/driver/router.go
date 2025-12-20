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
}

func NewRouter(raceStore Store, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Route(fmt.Sprintf("/{%s}", api.DriverIDPathParam), func(r chi.Router) {
		r.Use(api.DriverOwnershipMiddleware(api.DriverIDPathParam))

		r.Get("/", api.WrapWithSegment("getDriver", NewGetDriverEndpoint(raceStore)).ServeHTTP)
		r.Get("/races", api.WrapWithSegment("getDriverRaces", NewGetRacesEndpoint(raceStore)).ServeHTTP)
		r.Get("/races/{driver_race_id}", api.WrapWithSegment("getDriverRace", NewGetRaceEndpoint(raceStore)).ServeHTTP)
	})

	return r
}
