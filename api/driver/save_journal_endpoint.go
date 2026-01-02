package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/journal"
	"github.com/rs/zerolog"
)

type JournalServiceForSave interface {
	ValidateRaceExists(ctx context.Context, driverID, raceID int64) (bool, error)
	Save(ctx context.Context, input journal.SaveInput) (*journal.Entry, error)
}

func NewSaveJournalEndpoint(journalService JournalServiceForSave) http.Handler {
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

		var req SaveJournalEntryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			errs = errs.WithError("invalid JSON body")
		}

		// Validate tags
		for _, v := range journal.ValidateTags(req.Tags) {
			errs = errs.WithFieldErrorCode(v.Field, v.Code, v.Params)
		}

		// Check if the race exists (only if we have valid IDs)
		if !errs.HasAnyError() {
			exists, err := journalService.ValidateRaceExists(ctx, driverID, raceID)
			if err != nil {
				logger.Error().Err(err).Int64("driverId", driverID).Int64("raceId", raceID).Msg("failed to validate race exists")
				api.DoErrorResponse(ctx, w)
				return
			}
			if !exists {
				errs = errs.WithFieldErrorCode("driver_race_id", "race_not_found", nil)
			}
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		entry, err := journalService.Save(ctx, journal.SaveInput{
			DriverID: driverID,
			RaceID:   raceID,
			Notes:    req.Notes,
			Tags:     req.Tags,
		})
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Int64("raceId", raceID).Msg("failed to save journal entry")
			api.DoErrorResponse(ctx, w)
			return
		}

		api.DoOKResponse(ctx, journalEntryFromServiceEntry(*entry), w)
	})
}