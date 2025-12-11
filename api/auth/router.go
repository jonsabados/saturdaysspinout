package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(authService Service) http.Handler {
	r := chi.NewRouter()

	r.Post("/ir/callback", api.WrapWithSegment("authCallbackEndpoint", NewAuthCallbackEndpoint(authService)).ServeHTTP)

	return r
}
