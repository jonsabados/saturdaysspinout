package driver

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

type GetDriverStore interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
}

func NewGetDriverEndpoint(driverStore GetDriverStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		driverID, err := strconv.ParseInt(chi.URLParam(r, api.DriverIDPathParam), 10, 64)
		if err != nil {
			errs = errs.WithFieldError(api.DriverIDPathParam, "must be a valid integer")
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		driver, err := driverStore.GetDriver(ctx, driverID)
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Msg("failed to fetch driver")
			api.DoErrorResponse(ctx, w)
			return
		}

		if driver == nil {
			api.DoNotFoundResponse(ctx, "driver not found", w)
			return
		}

		api.DoOKResponse(ctx, driverInfoFromDriver(*driver), w)
	})
}