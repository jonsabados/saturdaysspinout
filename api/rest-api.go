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

type RootRouters struct {
	HealthRouter    http.Handler
	AuthRouter      http.Handler
	DeveloperRouter http.Handler
	IngestionRouter http.Handler
	DriverRouter    http.Handler
	TracksRouter    http.Handler
	CarsRouter      http.Handler
}

type RestAPIConfig struct {
	CORSAllowedOrigins []string
	// DeadlineBuffer is subtracted from existing context deadlines to leave room for cleanup.
	DeadlineBuffer time.Duration
}

func NewRestAPI(logger zerolog.Logger, correlationIDGenerator correlation.IDGenerator, routers RootRouters, cfg RestAPIConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Correlation-ID"},
		ExposedHeaders:   []string{"X-Correlation-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(ZerologLogAttachMiddleware(logger))
	r.Use(correlation.Middleware(correlationIDGenerator))
	r.Use(ReduceDeadlineMiddleware(cfg.DeadlineBuffer))
	r.Use(RequestLoggingMiddleware())

	r.Mount("/health", routers.HealthRouter)
	r.Mount("/auth", routers.AuthRouter)
	r.Mount("/ingestion", routers.IngestionRouter)
	r.Mount("/developer", routers.DeveloperRouter)
	r.Mount("/driver", routers.DriverRouter)
	r.Mount("/tracks", routers.TracksRouter)
	r.Mount("/cars", routers.CarsRouter)

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

// ReduceDeadlineMiddleware reduces existing context deadlines by the specified buffer.
// If no deadline exists, the context is passed through unchanged.
func ReduceDeadlineMiddleware(buffer time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()

			deadline, ok := ctx.Deadline()
			if !ok {
				next.ServeHTTP(writer, request)
				return
			}

			newDeadline := deadline.Add(-buffer)
			zerolog.Ctx(ctx).Debug().
				Time("original", deadline).
				Time("new", newDeadline).
				Msg("reducing context deadline")

			ctx, cancel := context.WithDeadline(ctx, newDeadline)
			defer cancel()

			next.ServeHTTP(writer, request.WithContext(ctx))
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
