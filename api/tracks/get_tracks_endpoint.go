package tracks

import (
	"context"
	"errors"
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/tracks"
	"github.com/rs/zerolog"
)

type TracksService interface {
	GetAll(ctx context.Context, accessToken string) ([]tracks.Track, error)
}

func NewGetTracksEndpoint(svc TracksService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		claims := api.SensitiveClaimsFromContext(ctx)
		if claims == nil {
			api.DoErrorResponse(ctx, w)
			return
		}

		trackList, err := svc.GetAll(ctx, claims.IRacingAccessToken)
		if err != nil {
			if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
				logger.Warn().Err(err).Msg("iRacing token expired while fetching tracks")
				api.DoUnauthorizedResponse(ctx, "iRacing access token expired", w)
				return
			}
			logger.Error().Err(err).Msg("failed to fetch tracks")
			api.DoErrorResponse(ctx, w)
			return
		}

		response := make([]Track, len(trackList))
		for i, t := range trackList {
			response[i] = trackFromDomain(t)
		}

		api.DoOKResponse(ctx, response, w)
	})
}