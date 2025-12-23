package ingestion

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(driverStore Store, dispatcher EventDispatcher, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Post("/race", api.WrapWithSegment("raceIngestionEndpoint", NewRaceIngestionEndpoint(driverStore, dispatcher)).ServeHTTP)

	return r
}
