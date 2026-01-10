package series

import (
	"context"
	"errors"
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/series"
	"github.com/rs/zerolog"
)

type SeriesService interface {
	GetAll(ctx context.Context, accessToken string) ([]series.Series, error)
}

func NewGetSeriesEndpoint(svc SeriesService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		claims := api.SensitiveClaimsFromContext(ctx)
		if claims == nil {
			api.DoErrorResponse(ctx, w)
			return
		}

		seriesList, err := svc.GetAll(ctx, claims.IRacingAccessToken)
		if err != nil {
			if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
				logger.Warn().Err(err).Msg("iRacing token expired while fetching series")
				api.DoUnauthorizedResponse(ctx, "iRacing access token expired", w)
				return
			}
			logger.Error().Err(err).Msg("failed to fetch series")
			api.DoErrorResponse(ctx, w)
			return
		}

		response := make([]Series, len(seriesList))
		for i, s := range seriesList {
			response[i] = seriesFromDomain(s)
		}

		api.DoOKResponse(ctx, response, w)
	})
}