package tracks

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(svc TracksService, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", api.WrapWithSegment("getTracks", NewGetTracksEndpoint(svc)).ServeHTTP)

	return r
}