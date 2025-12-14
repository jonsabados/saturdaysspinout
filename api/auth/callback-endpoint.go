package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/auth"
)

type CallbackRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

type CallbackResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	UserID    int64  `json:"user_id"`
	UserName  string `json:"user_name"`
}

type Service interface {
	HandleCallback(ctx context.Context, code, codeVerifier, redirectURI string) (*auth.Result, error)
	HandleRefresh(ctx context.Context, userID int64, userName string, refreshToken string) (*auth.Result, error)
}

func NewAuthCallbackEndpoint(authService Service) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		logger := zerolog.Ctx(ctx)

		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req CallbackRequest
		if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
			logger.Warn().Err(err).Msg("failed to decode request body")
			api.DoBadRequestResponse(ctx, api.RequestErrors{}.WithError("invalid request body"), writer)
			return
		}

		var errs api.RequestErrors
		if req.Code == "" {
			errs = errs.WithFieldError("code", "required")
		}
		if req.CodeVerifier == "" {
			errs = errs.WithFieldError("code_verifier", "required")
		}
		if req.RedirectURI == "" {
			errs = errs.WithFieldError("redirect_uri", "required")
		}
		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, writer)
			return
		}

		result, err := authService.HandleCallback(ctx, req.Code, req.CodeVerifier, req.RedirectURI)
		if err != nil {
			logger.Error().Err(err).Msg("authentication failed")
			api.DoErrorResponse(ctx, writer)
			return
		}

		logger.Info().Int64("user_id", result.UserID).Str("user_name", result.UserName).Msg("user authenticated successfully")

		api.DoOKResponse(ctx, CallbackResponse{
			Token:     result.Token,
			ExpiresAt: result.ExpiresAt.Unix(),
			UserID:    result.UserID,
			UserName:  result.UserName,
		}, writer)
	})
}
