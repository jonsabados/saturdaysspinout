package cars

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(svc CarsService, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(authMiddleware)

	r.Get("/", api.WrapWithSegment("getCars", NewGetCarsEndpoint(svc)).ServeHTTP)

	return r
}