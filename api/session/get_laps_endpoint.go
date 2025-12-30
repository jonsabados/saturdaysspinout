package session

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/iracing"
)

const (
	SimsessionPathParam = "simsession"
	DriverIDPathParam   = "driver_id"
)

type LapDataClient interface {
	GetLapData(ctx context.Context, accessToken string, subsessionID int64, simsessionNumber int, opts ...iracing.GetLapDataOption) (*iracing.LapDataResponse, error)
}

func NewGetLapsEndpoint(client LapDataClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		subsessionIDStr := chi.URLParam(r, SubsessionIDPathParam)
		if subsessionIDStr == "" {
			errs = errs.WithFieldError(SubsessionIDPathParam, "required")
		}

		simsessionStr := chi.URLParam(r, SimsessionPathParam)
		if simsessionStr == "" {
			errs = errs.WithFieldError(SimsessionPathParam, "required")
		}

		driverIDStr := chi.URLParam(r, DriverIDPathParam)
		if driverIDStr == "" {
			errs = errs.WithFieldError(DriverIDPathParam, "required")
		}

		var subsessionID int64
		var simsession int
		var driverID int64
		var err error

		if subsessionIDStr != "" {
			subsessionID, err = strconv.ParseInt(subsessionIDStr, 10, 64)
			if err != nil {
				errs = errs.WithFieldError(SubsessionIDPathParam, "must be a valid integer")
			}
		}

		if simsessionStr != "" {
			simsession64, err := strconv.ParseInt(simsessionStr, 10, 32)
			if err != nil {
				errs = errs.WithFieldError(SimsessionPathParam, "must be a valid integer")
			}
			simsession = int(simsession64)
		}

		if driverIDStr != "" {
			driverID, err = strconv.ParseInt(driverIDStr, 10, 64)
			if err != nil {
				errs = errs.WithFieldError(DriverIDPathParam, "must be a valid integer")
			}
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		claims := api.SensitiveClaimsFromContext(ctx)
		if claims == nil {
			logger.Error().Msg("sensitive claims not found in context")
			api.DoErrorResponse(ctx, w)
			return
		}

		result, err := client.GetLapData(ctx, claims.IRacingAccessToken, subsessionID, simsession, iracing.WithCustomerIDLap(driverID))
		if err != nil {
			if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
				logger.Warn().Err(err).Msg("iRacing token expired while fetching lap data")
				api.DoUnauthorizedResponse(ctx, "iRacing access token expired", w)
				return
			}
			logger.Error().Err(err).
				Int64("subsessionId", subsessionID).
				Int("simsession", simsession).
				Int64("driverId", driverID).
				Msg("failed to fetch lap data")
			api.DoErrorResponse(ctx, w)
			return
		}

		api.DoOKResponse(ctx, lapDataResponseFromIRacing(result), w)
	})
}