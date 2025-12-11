package health

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/ping", NewPingEndpoint().ServeHTTP)

	return r
}
