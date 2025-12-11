package health

import (
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
)

func NewPingEndpoint() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		api.DoOKResponse(request.Context(), "Pong", writer)
	})
}
