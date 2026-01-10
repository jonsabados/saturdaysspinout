package driver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/analytics"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/rs/zerolog"
)

// NewAnalyticsEndpoint creates the handler for GET /driver/{driver_id}/analytics
func NewAnalyticsEndpoint(svc AnalyticsService) http.Handler {
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

		// Parse groupBy (repeated param)
		var groupBy []analytics.GroupByDimension
		for _, g := range r.URL.Query()[api.GroupByQueryParam] {
			dim := analytics.GroupByDimension(g)
			if !dim.IsValid() {
				errs = errs.WithFieldErrorCode(api.GroupByQueryParam, ErrCodeInvalidValue, map[string]string{
					"value":   g,
					"allowed": "series, car, track",
				})
			} else {
				groupBy = append(groupBy, dim)
			}
		}

		// Parse granularity
		var granularity analytics.Granularity
		if g := r.URL.Query().Get(api.GranularityQueryParam); g != "" {
			granularity = analytics.Granularity(g)
			if !granularity.IsValid() {
				errs = errs.WithFieldErrorCode(api.GranularityQueryParam, ErrCodeInvalidValue, map[string]string{
					"value":   g,
					"allowed": "day, week, month, year",
				})
			}
		}

		// groupBy and granularity are mutually exclusive
		if len(groupBy) > 0 && granularity != "" {
			errs = errs.WithFieldErrorCode(api.GroupByQueryParam, ErrCodeMutualExclusive, map[string]string{
				"other": api.GranularityQueryParam,
			})
		}

		// Parse filter params (repeated params for OR within dimension)
		seriesIDs, seriesErrs := parseInt64Slice(r.URL.Query()[api.SeriesIDQueryParam])
		if len(seriesErrs) > 0 {
			for _, e := range seriesErrs {
				errs = errs.WithFieldErrorCode(api.SeriesIDQueryParam, ErrCodeInvalidInteger, map[string]string{"value": e})
			}
		}

		carIDs, carErrs := parseInt64Slice(r.URL.Query()[api.CarIDQueryParam])
		if len(carErrs) > 0 {
			for _, e := range carErrs {
				errs = errs.WithFieldErrorCode(api.CarIDQueryParam, ErrCodeInvalidInteger, map[string]string{"value": e})
			}
		}

		trackIDs, trackErrs := parseInt64Slice(r.URL.Query()[api.TrackIDQueryParam])
		if len(trackErrs) > 0 {
			for _, e := range trackErrs {
				errs = errs.WithFieldErrorCode(api.TrackIDQueryParam, ErrCodeInvalidInteger, map[string]string{"value": e})
			}
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		// Build request and call service
		req := analytics.AnalyticsRequest{
			DriverID:    driverID,
			From:        startTime,
			To:          endTime,
			GroupBy:     groupBy,
			Granularity: granularity,
			SeriesIDs:   seriesIDs,
			CarIDs:      carIDs,
			TrackIDs:    trackIDs,
		}

		result, err := svc.GetAnalytics(ctx, req)
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Msg("failed to get analytics")
			api.DoErrorResponse(ctx, w)
			return
		}

		// Convert domain result to API response
		response := AnalyticsResponse{
			Summary: summaryFromDomain(result.Summary),
		}

		if len(result.GroupedBy) > 0 {
			response.GroupedBy = make([]AnalyticsGroup, len(result.GroupedBy))
			for i, g := range result.GroupedBy {
				response.GroupedBy[i] = AnalyticsGroup{
					SeriesID: g.SeriesID,
					CarID:    g.CarID,
					TrackID:  g.TrackID,
					Summary:  summaryFromDomain(g.Summary),
				}
			}
		}

		if len(result.TimeSeries) > 0 {
			response.TimeSeries = make([]AnalyticsPeriod, len(result.TimeSeries))
			for i, p := range result.TimeSeries {
				response.TimeSeries[i] = AnalyticsPeriod{
					Period:  p.Period,
					Summary: summaryFromDomain(p.Summary),
				}
			}
		}

		api.DoOKResponse(ctx, response, w)
	})
}

// parseInt64Slice parses a slice of strings to int64s, returning invalid values separately
func parseInt64Slice(values []string) ([]int64, []string) {
	var ints []int64
	var invalid []string

	for _, v := range values {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			invalid = append(invalid, v)
		} else {
			ints = append(ints, i)
		}
	}

	return ints, invalid
}

func summaryFromDomain(s analytics.Summary) AnalyticsSummary {
	return AnalyticsSummary{
		RaceCount:         s.RaceCount,
		IRatingStart:      s.IRatingStart,
		IRatingEnd:        s.IRatingEnd,
		IRatingDelta:      s.IRatingDelta,
		IRatingGain:       s.IRatingGain,
		IRatingLoss:       s.IRatingLoss,
		CPIStart:          s.CPIStart,
		CPIEnd:            s.CPIEnd,
		CPIDelta:          s.CPIDelta,
		CPIGain:           s.CPIGain,
		CPILoss:           s.CPILoss,
		Podiums:           s.Podiums,
		Top5Finishes:      s.Top5Finishes,
		Wins:              s.Wins,
		AvgFinishPosition: s.AvgFinishPosition,
		AvgStartPosition:  s.AvgStartPosition,
		PositionsGained:   s.PositionsGained,
		TotalIncidents:    s.TotalIncidents,
		AvgIncidents:      s.AvgIncidents,
	}
}