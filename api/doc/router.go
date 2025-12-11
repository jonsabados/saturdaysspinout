package doc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(docFetcher Fetcher) http.Handler {
	r := chi.NewRouter()

	r.Get("/iracing-api/*", api.WrapWithSegment("iracingDocProxyEndpoint", NewIRacingDocProxyEndpoint(docFetcher)).ServeHTTP)

	return r
}
