package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/auth"
)

type TokenValidator interface {
	ValidateToken(ctx context.Context, tokenString string) (*auth.SessionClaims, *auth.SensitiveClaims, error)
}

type sessionClaimsKeyType string
type sensitiveClaimsKeyType string

const sessionClaimsKey = sessionClaimsKeyType("sessionClaims")
const sensitiveClaimsKey = sensitiveClaimsKeyType("sensitiveClaims")

func AuthMiddleware(validator TokenValidator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				DoUnauthorizedResponse(ctx, "missing authorization header", w)
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				DoUnauthorizedResponse(ctx, "invalid authorization header format", w)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			sessionClaims, sensitiveClaims, err := validator.ValidateToken(ctx, token)
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("token validation failed")
				DoUnauthorizedResponse(ctx, "invalid token", w)
				return
			}

			ctx = context.WithValue(ctx, sessionClaimsKey, sessionClaims)
			ctx = context.WithValue(ctx, sensitiveClaimsKey, sensitiveClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func SessionClaimsFromContext(ctx context.Context) *auth.SessionClaims {
	if claims, ok := ctx.Value(sessionClaimsKey).(*auth.SessionClaims); ok {
		return claims
	}
	return nil
}

func SensitiveClaimsFromContext(ctx context.Context) *auth.SensitiveClaims {
	if claims, ok := ctx.Value(sensitiveClaimsKey).(*auth.SensitiveClaims); ok {
		return claims
	}
	return nil
}
