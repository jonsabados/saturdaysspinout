package developer

import (
	"net/http"

	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
}

func NewIRacingTokenEndpoint() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		claims := api.SensitiveClaimsFromContext(ctx)
		if claims == nil {
			logger.Error().Msg("sensitive claims not found in context")
			api.DoErrorResponse(ctx, w)
			return
		}

		api.DoOKResponse(ctx, TokenResponse{
			AccessToken: claims.IRacingAccessToken,
		}, w)
	})
}
