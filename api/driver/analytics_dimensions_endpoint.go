package driver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/analytics"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/rs/zerolog"
)

// AnalyticsService defines the interface for analytics operations.
type AnalyticsService interface {
	GetDimensions(ctx context.Context, driverID int64, from, to time.Time) (*analytics.Dimensions, error)
	GetAnalytics(ctx context.Context, req analytics.AnalyticsRequest) (*analytics.AnalyticsResult, error)
}

// Error codes for i18n support
const (
	ErrCodeRequired        = "required"
	ErrCodeInvalidInteger  = "invalid_integer"
	ErrCodeInvalidISO8601  = "invalid_iso8601"
	ErrCodeEndBeforeStart  = "end_before_start"
	ErrCodeInvalidValue    = "invalid_value"
	ErrCodeMutualExclusive = "mutual_exclusive"
)

// NewAnalyticsDimensionsEndpoint creates the handler for GET /driver/{driver_id}/analytics/dimensions
func NewAnalyticsDimensionsEndpoint(svc AnalyticsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		// Parse driver ID from path
		driverID, err := strconv.ParseInt(chi.URLParam(r, api.DriverIDPathParam), 10, 64)
		if err != nil {
			errs = errs.WithFieldErrorCode(api.DriverIDPathParam, ErrCodeInvalidInteger, nil)
		}

		// Parse time range from query params
		var startTime, endTime time.Time

		startTimeStr := r.URL.Query().Get(api.StartTimeQueryParam)
		if startTimeStr == "" {
			errs = errs.WithFieldErrorCode(api.StartTimeQueryParam, ErrCodeRequired, nil)
		} else {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				errs = errs.WithFieldErrorCode(api.StartTimeQueryParam, ErrCodeInvalidISO8601, nil)
			}
		}

		endTimeStr := r.URL.Query().Get(api.EndTimeQueryParam)
		if endTimeStr == "" {
			errs = errs.WithFieldErrorCode(api.EndTimeQueryParam, ErrCodeRequired, nil)
		} else {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				errs = errs.WithFieldErrorCode(api.EndTimeQueryParam, ErrCodeInvalidISO8601, nil)
			}
		}

		// Cross-field validation: endTime must be after startTime
		if !startTime.IsZero() && !endTime.IsZero() && endTime.Before(startTime) {
			errs = errs.WithFieldErrorCode(api.EndTimeQueryParam, ErrCodeEndBeforeStart, nil)
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		// Get dimensions from service
		dims, err := svc.GetDimensions(ctx, driverID, startTime, endTime)
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Msg("failed to get dimensions")
			api.DoErrorResponse(ctx, w)
			return
		}

		response := DimensionsResponse{
			Series: dims.SeriesIDs,
			Cars:   dims.CarIDs,
			Tracks: dims.TrackIDs,
		}

		// Ensure non-nil slices for JSON
		if response.Series == nil {
			response.Series = []int64{}
		}
		if response.Cars == nil {
			response.Cars = []int64{}
		}
		if response.Tracks == nil {
			response.Tracks = []int64{}
		}

		api.DoOKResponse(ctx, response, w)
	})
}