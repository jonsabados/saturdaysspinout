package driver

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/journal"
	"github.com/rs/zerolog"
)

type GetJournalEntryStore interface {
	Get(ctx context.Context, driverID, raceID int64) (*journal.Entry, error)
}

func NewGetJournalEntryEndpoint(journalService GetJournalEntryStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		driverID, err := strconv.ParseInt(chi.URLParam(r, api.DriverIDPathParam), 10, 64)
		if err != nil {
			errs = errs.WithFieldError(api.DriverIDPathParam, "must be a valid integer")
		}

		var raceID int64
		raceIDStr := chi.URLParam(r, "driver_race_id")
		if raceIDStr == "" {
			errs = errs.WithFieldError("driver_race_id", "required")
		} else {
			raceID, err = strconv.ParseInt(raceIDStr, 10, 64)
			if err != nil {
				errs = errs.WithFieldError("driver_race_id", "must be a valid integer")
			}
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		entry, err := journalService.Get(ctx, driverID, raceID)
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Int64("raceId", raceID).Msg("failed to get journal entry")
			api.DoErrorResponse(ctx, w)
			return
		}

		if entry == nil {
			api.DoNotFoundResponse(ctx, "no journal entry found for this race", w)
			return
		}

		api.DoOKResponse(ctx, journalEntryFromServiceEntry(*entry), w)
	})
}