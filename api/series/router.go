package series

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(svc SeriesService, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", api.WrapWithSegment("getSeries", NewGetSeriesEndpoint(svc)).ServeHTTP)

	return r
}