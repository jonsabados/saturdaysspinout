package session

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(client IRacingClient, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/{"+SubsessionIDPathParam+"}", api.WrapWithSegment("getSession", NewGetSessionEndpoint(client)).ServeHTTP)

	return r
}