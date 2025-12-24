package driver

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/rs/zerolog"
)

type DeleteRacesStore interface {
	DeleteDriverRaces(ctx context.Context, driverID int64) error
}

func NewDeleteRacesEndpoint(store DeleteRacesStore) http.Handler {
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

		err = store.DeleteDriverRaces(ctx, driverID)
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Msg("failed to delete driver races")
			api.DoErrorResponse(ctx, w)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}