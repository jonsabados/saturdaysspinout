package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/auth"
)

// AuthCallbackRequest represents the request from the frontend
type AuthCallbackRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
}

// AuthCallbackResponse represents the response to the frontend
type AuthCallbackResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	HandleCallback(ctx context.Context, code, codeVerifier, redirectURI string) (*auth.AuthResult, error)
}

// AuthCallbackEndpoint handles the OAuth callback HTTP transport
type AuthCallbackEndpoint struct {
	authService AuthService
}

// NewAuthCallbackEndpoint creates a new AuthCallbackEndpoint
func NewAuthCallbackEndpoint(authService AuthService) *AuthCallbackEndpoint {
	return &AuthCallbackEndpoint{
		authService: authService,
	}
}

func (e *AuthCallbackEndpoint) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	logger := zerolog.Ctx(ctx)

	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req AuthCallbackRequest
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		logger.Warn().Err(err).Msg("failed to decode request body")
		DoBadRequestResponse(ctx, []string{"invalid request body"}, nil, writer)
		return
	}

	if req.Code == "" {
		DoBadRequestResponse(ctx, []string{"code is required"}, nil, writer)
		return
	}
	if req.CodeVerifier == "" {
		DoBadRequestResponse(ctx, []string{"code_verifier is required"}, nil, writer)
		return
	}
	if req.RedirectURI == "" {
		DoBadRequestResponse(ctx, []string{"redirect_uri is required"}, nil, writer)
		return
	}

	result, err := e.authService.HandleCallback(ctx, req.Code, req.CodeVerifier, req.RedirectURI)
	if err != nil {
		logger.Error().Err(err).Msg("authentication failed")
		DoErrorResponse(ctx, writer)
		return
	}

	logger.Info().Int("user_id", result.UserID).Str("user_name", result.UserName).Msg("user authenticated successfully")

	DoOKResponse(ctx, AuthCallbackResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Unix(),
	}, writer)
}