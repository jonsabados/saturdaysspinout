package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
)

func NewRouter(authService Service, authMiddleware func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()

	r.Post("/ir/callback", api.WrapWithSegment("authCallbackEndpoint", NewAuthCallbackEndpoint(authService)).ServeHTTP)

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/refresh", api.WrapWithSegment("authRefreshEndpoint", NewAuthRefreshEndpoint(authService)).ServeHTTP)
	})

	return r
}
