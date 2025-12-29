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

const SubsessionIDPathParam = "subsession_id"

type IRacingClient interface {
	GetSessionResults(ctx context.Context, accessToken string, subsessionID int64, opts ...iracing.GetSessionResultsOption) (*iracing.SessionResult, error)
}

func NewGetSessionEndpoint(client IRacingClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		subsessionIDStr := chi.URLParam(r, SubsessionIDPathParam)
		if subsessionIDStr == "" {
			errs = errs.WithFieldError(SubsessionIDPathParam, "required")
		}

		var subsessionID int64
		var err error
		if subsessionIDStr != "" {
			subsessionID, err = strconv.ParseInt(subsessionIDStr, 10, 64)
			if err != nil {
				errs = errs.WithFieldError(SubsessionIDPathParam, "must be a valid integer")
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

		result, err := client.GetSessionResults(ctx, claims.IRacingAccessToken, subsessionID, iracing.WithIncludeLicenses(true))
		if err != nil {
			if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
				logger.Warn().Err(err).Msg("iRacing token expired while fetching session results")
				api.DoUnauthorizedResponse(ctx, "iRacing access token expired", w)
				return
			}
			logger.Error().Err(err).Int64("subsessionId", subsessionID).Msg("failed to fetch session results")
			api.DoErrorResponse(ctx, w)
			return
		}

		api.DoOKResponse(ctx, sessionResponseFromIRacing(result), w)
	})
}
