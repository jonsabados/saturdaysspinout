package ingestion

import (
	"context"
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/rs/zerolog"
)

type RaceIngestionRequest struct {
	DriverID           int64  `json:"driverId"`
	IRacingAccessToken string `json:"IRacingAccessToken"`
}

type EventDispatcher interface {
	PublishEvent(ctx context.Context, event any) error
}

func NewRaceIngestionEndpoint(dispatcher EventDispatcher) http.Handler {
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

		event := RaceIngestionRequest{
			DriverID:           sessionClaims.IRacingUserID,
			IRacingAccessToken: sensitiveClaims.IRacingAccessToken,
		}

		if err := dispatcher.PublishEvent(ctx, event); err != nil {
			logger.Error().Err(err).Msg("failed to publish race ingestion event")
			api.DoErrorResponse(ctx, writer)
			return
		}

		logger.Info().Int64("driverId", sessionClaims.IRacingUserID).Msg("race ingestion request queued")

		api.DoAcceptedResponse(ctx, map[string]string{"status": "queued"}, writer)
	})
}
