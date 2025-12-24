package api

import (
	"net/http"
	"slices"
)

func EntitlementMiddleware(requiredEntitlement string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			sessionClaims := SessionClaimsFromContext(ctx)
			if sessionClaims == nil {
				DoUnauthorizedResponse(ctx, "missing session claims", w)
				return
			}

			if !slices.Contains(sessionClaims.Entitlements, requiredEntitlement) {
				DoForbiddenResponse(ctx, "insufficient entitlements", w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}