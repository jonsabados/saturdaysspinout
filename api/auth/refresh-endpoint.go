package auth

import (
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/rs/zerolog"
)

func NewAuthRefreshEndpoint(authService Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logger := zerolog.Ctx(ctx)

		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		sessionClaims := api.SessionClaimsFromContext(ctx)
		sensitiveClaims := api.SensitiveClaimsFromContext(ctx)
		if sessionClaims == nil || sensitiveClaims == nil {
			logger.Error().Msg("claims not found in context")
			api.DoErrorResponse(ctx, writer)
			return
		}

		result, err := authService.HandleRefresh(ctx, sessionClaims.IRacingUserID, sessionClaims.IRacingUserName, sessionClaims.Entitlements, sensitiveClaims.IRacingRefreshToken)
		if err != nil {
			logger.Error().Err(err).Msg("token refresh failed")
			api.DoErrorResponse(ctx, writer)
			return
		}

		logger.Info().Int64("user_id", result.UserID).Msg("token refreshed successfully")

		api.DoOKResponse(ctx, CallbackResponse{
			Token:     result.Token,
			ExpiresAt: result.ExpiresAt.Unix(),
			UserID:    result.UserID,
			UserName:  result.UserName,
		}, writer)
	})
}