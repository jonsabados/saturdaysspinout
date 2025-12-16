package driver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

type GetRaceStore interface {
	GetDriverSession(ctx context.Context, driverID int64, startTime time.Time) (*store.DriverSession, error)
}

func NewGetRaceEndpoint(raceStore GetRaceStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		driverID, err := strconv.ParseInt(chi.URLParam(r, api.DriverIDPathParam), 10, 64)
		if err != nil {
			errs = errs.WithFieldError(api.DriverIDPathParam, "must be a valid integer")
		}

		var driverRaceID int64

		driverRaceIDStr := chi.URLParam(r, "driver_race_id")
		if driverRaceIDStr == "" {
			errs = errs.WithFieldError("driver_race_id", "required")
		} else {
			driverRaceID, err = strconv.ParseInt(driverRaceIDStr, 10, 64)
			if err != nil {
				errs = errs.WithFieldError("driver_race_id", "must be a valid integer")
			}
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		session, err := raceStore.GetDriverSession(ctx, driverID, store.TimeFromDriverRaceID(driverRaceID))
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Int64("driverRaceId", driverRaceID).Msg("failed to fetch driver session")
			api.DoErrorResponse(ctx, w)
			return
		}

		if session == nil {
			api.DoNotFoundResponse(ctx, "race not found", w)
			return
		}

		api.DoOKResponse(ctx, raceFromDriverSession(*session), w)
	})
}