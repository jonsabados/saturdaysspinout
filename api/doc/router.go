package doc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(docFetcher Fetcher, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/iracing-api/*", api.WrapWithSegment("iracingDocProxyEndpoint", NewIRacingDocProxyEndpoint(docFetcher)).ServeHTTP)

	return r
}
