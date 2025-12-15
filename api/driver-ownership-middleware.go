package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func DriverOwnershipMiddleware(pathParam string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			sessionClaims := SessionClaimsFromContext(ctx)
			if sessionClaims == nil {
				DoUnauthorizedResponse(ctx, "missing session claims", w)
				return
			}

			driverIDParam := chi.URLParam(r, pathParam)
			if driverIDParam == "" {
				DoBadRequestResponse(ctx, NewRequestErrors().WithFieldError(pathParam, "required"), w)
				return
			}

			driverID, err := strconv.ParseInt(driverIDParam, 10, 64)
			if err != nil {
				DoBadRequestResponse(ctx, NewRequestErrors().WithFieldError(pathParam, "must be a valid integer"), w)
				return
			}

			if driverID != sessionClaims.IRacingUserID {
				DoForbiddenResponse(ctx, "access denied", w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}