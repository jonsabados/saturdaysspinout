package session

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

// CombinedClient combines all iRacing client methods needed by session endpoints.
type CombinedClient interface {
	IRacingClient
	LapDataClient
}

func NewRouter(client CombinedClient, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/{"+SubsessionIDPathParam+"}", api.WrapWithSegment("getSession", NewGetSessionEndpoint(client)).ServeHTTP)
	r.Get("/{"+SubsessionIDPathParam+"}/simsession/{"+SimsessionPathParam+"}/driver/{"+DriverIDPathParam+"}/laps", api.WrapWithSegment("getLaps", NewGetLapsEndpoint(client)).ServeHTTP)

	return r
}