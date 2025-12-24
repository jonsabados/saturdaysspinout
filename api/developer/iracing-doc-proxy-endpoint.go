package developer

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/iracing"
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
			if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
				logger.Warn().Err(err).Str("path", path).Msg("iracing token expired")
				api.DoUnauthorizedResponse(ctx, "iRacing access token expired", w)
				return
			}
			logger.Error().Err(err).Str("path", path).Msg("failed to fetch iracing doc")
			api.DoErrorResponse(ctx, w)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})
}
