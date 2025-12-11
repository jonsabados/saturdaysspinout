package doc

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
)

type Fetcher interface {
	Fetch(ctx context.Context, accessToken string, path string) ([]byte, string, error)
}

func NewIRacingDocProxyEndpoint(fetcher Fetcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		claims := api.SensitiveClaimsFromContext(ctx)
		if claims == nil {
			api.DoErrorResponse(ctx, w)
			return
		}

		path := "/" + chi.URLParam(r, "*")

		body, contentType, err := fetcher.Fetch(ctx, claims.IRacingAccessToken, path)
		if err != nil {
			logger.Error().Err(err).Str("path", path).Msg("failed to fetch iracing doc")
			api.DoErrorResponse(ctx, w)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})
}
