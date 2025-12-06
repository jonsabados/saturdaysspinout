package api

import (
	"net/http"
)

func NewPingEndpoint() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		DoOKResponse(request.Context(), "Pong", writer)
	})
}
