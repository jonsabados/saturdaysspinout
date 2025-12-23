package ingestion

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

type RaceIngestionRequest struct {
	NotifyConnectionID string `json:"notifyConnectionId"`
}

type EventDispatcher interface {
	PublishEvent(ctx context.Context, event any) error
}

type Store interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
}

func NewRaceIngestionEndpoint(driverStore Store, dispatcher EventDispatcher) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logger := zerolog.Ctx(ctx)

		sessionClaims := api.SessionClaimsFromContext(ctx)
		if sessionClaims == nil {
			api.DoUnauthorizedResponse(ctx, "missing session claims", writer)
			return
		}

		sensitiveClaims := api.SensitiveClaimsFromContext(ctx)
		if sensitiveClaims == nil {
			api.DoUnauthorizedResponse(ctx, "missing sensitive claims", writer)
			return
		}

		driver, err := driverStore.GetDriver(ctx, sessionClaims.IRacingUserID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get driver")
			api.DoErrorResponse(ctx, writer)
			return
		}
		if driver == nil {
			api.DoNotFoundResponse(ctx, "driver not found", writer)
			return
		}

		now := time.Now()
		if driver.IngestionBlockedUntil != nil && driver.IngestionBlockedUntil.After(now) {
			retryAfter := int(driver.IngestionBlockedUntil.Sub(now).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			api.DoTooManyRequestsResponse(ctx, "race ingestion already in progress", retryAfter, writer)
			return
		}

		var req RaceIngestionRequest
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			api.DoBadRequestResponse(ctx, api.NewRequestErrors().WithError("invalid request body"), writer)
			return
		}

		errs := api.NewRequestErrors()
		if req.NotifyConnectionID == "" {
			errs = errs.WithFieldError("notifyConnectionId", "required")
		}
		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, writer)
			return
		}

		if err := dispatcher.PublishEvent(ctx, ingestion.RaceIngestionRequest{
			DriverID:           sessionClaims.IRacingUserID,
			IRacingAccessToken: sensitiveClaims.IRacingAccessToken,
			NotifyConnectionID: req.NotifyConnectionID,
		}); err != nil {
			logger.Error().Err(err).Msg("failed to publish race ingestion event")
			api.DoErrorResponse(ctx, writer)
			return
		}

		logger.Info().Int64("driverId", sessionClaims.IRacingUserID).Msg("race ingestion request queued")

		api.DoAcceptedResponse(ctx, map[string]string{"status": "queued"}, writer)
	})
}
