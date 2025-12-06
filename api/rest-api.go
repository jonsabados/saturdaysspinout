package api

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/correlation"
)

func NewRestAPI(logger zerolog.Logger, correlationIDGenerator correlation.IDGenerator, corsAllowedOrigins []string, pingEndpoint http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Correlation-ID"},
		ExposedHeaders:   []string{"X-Correlation-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(ZerologLogAttachMiddleware(logger))
	r.Use(correlation.Middleware(correlationIDGenerator))
	r.Use(RequestLoggingMiddleware())

	r.Get("/health/ping", WrapWithSegment("pingEndpoint", pingEndpoint).ServeHTTP)

	return xray.Handler(xray.NewFixedSegmentNamer("processHttpRequest"), r)
}

func ZerologLogAttachMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			ctx = logger.WithContext(ctx)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func RequestLoggingMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ww := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

			t1 := time.Now()
			defer func() {
				zerolog.Ctx(request.Context()).Info().
					Int("status", ww.Status()).
					Int("bytesWritten", ww.BytesWritten()).
					Dur("duration", time.Since(t1)).
					Msg("request processed")
			}()

			next.ServeHTTP(ww, request)
		})
	}
}

func WrapWithSegment(segmentName string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = xray.Capture(request.Context(), segmentName, func(ctx context.Context) error {
			handler.ServeHTTP(writer, request.WithContext(ctx))
			return nil
		})
	})
}
