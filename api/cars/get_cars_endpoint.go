package cars

import (
	"context"
	"errors"
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/cars"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/rs/zerolog"
)

type CarsService interface {
	GetAll(ctx context.Context, accessToken string) ([]cars.Car, error)
}

func NewGetCarsEndpoint(svc CarsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		claims := api.SensitiveClaimsFromContext(ctx)
		if claims == nil {
			api.DoErrorResponse(ctx, w)
			return
		}

		carList, err := svc.GetAll(ctx, claims.IRacingAccessToken)
		if err != nil {
			if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
				logger.Warn().Err(err).Msg("iRacing token expired while fetching cars")
				api.DoUnauthorizedResponse(ctx, "iRacing access token expired", w)
				return
			}
			logger.Error().Err(err).Msg("failed to fetch cars")
			api.DoErrorResponse(ctx, w)
			return
		}

		response := make([]Car, len(carList))
		for i, c := range carList {
			response[i] = carFromDomain(c)
		}

		api.DoOKResponse(ctx, response, w)
	})
}