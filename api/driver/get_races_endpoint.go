package driver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

type GetRacesStore interface {
	GetDriverSessionsByTimeRange(ctx context.Context, driverID int64, from, to time.Time, filters ...store.SessionFilter) ([]store.DriverSession, error)
}

func NewGetRacesEndpoint(raceStore GetRacesStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		driverID, err := strconv.ParseInt(chi.URLParam(r, api.DriverIDPathParam), 10, 64)
		if err != nil {
			errs = errs.WithFieldErrorCode(api.DriverIDPathParam, ErrCodeInvalidInteger, nil)
		}

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

		if !startTime.IsZero() && !endTime.IsZero() && endTime.Before(startTime) {
			errs = errs.WithFieldErrorCode(api.EndTimeQueryParam, ErrCodeEndBeforeStart, nil)
		}

		page := 1
		if pageStr := r.URL.Query().Get(api.PageQueryParam); pageStr != "" {
			page, err = strconv.Atoi(pageStr)
			if err != nil || page < 1 {
				errs = errs.WithFieldErrorCode(api.PageQueryParam, ErrCodePositiveInteger, nil)
			}
		}

		resultsPerPage := api.DefaultResultsPerPage
		if rppStr := r.URL.Query().Get(api.ResultsPerPageParam); rppStr != "" {
			resultsPerPage, err = strconv.Atoi(rppStr)
			if err != nil || resultsPerPage < 1 {
				errs = errs.WithFieldErrorCode(api.ResultsPerPageParam, ErrCodePositiveInteger, nil)
			}
		}

		seriesIDs, seriesErrs := parseInt64Slice(r.URL.Query()[api.SeriesIDQueryParam])
		for _, e := range seriesErrs {
			errs = errs.WithFieldErrorCode(api.SeriesIDQueryParam, ErrCodeInvalidInteger, map[string]string{"value": e})
		}

		carIDs, carErrs := parseInt64Slice(r.URL.Query()[api.CarIDQueryParam])
		for _, e := range carErrs {
			errs = errs.WithFieldErrorCode(api.CarIDQueryParam, ErrCodeInvalidInteger, map[string]string{"value": e})
		}

		trackIDs, trackErrs := parseInt64Slice(r.URL.Query()[api.TrackIDQueryParam])
		for _, e := range trackErrs {
			errs = errs.WithFieldErrorCode(api.TrackIDQueryParam, ErrCodeInvalidInteger, map[string]string{"value": e})
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		var filters []store.SessionFilter
		if len(seriesIDs) > 0 {
			filters = append(filters, store.FilterBySeriesIDs(seriesIDs))
		}
		if len(carIDs) > 0 {
			filters = append(filters, store.FilterByCarIDs(carIDs))
		}
		if len(trackIDs) > 0 {
			filters = append(filters, store.FilterByTrackIDs(trackIDs))
		}

		sessions, err := raceStore.GetDriverSessionsByTimeRange(ctx, driverID, startTime, endTime, filters...)
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Msg("failed to fetch driver sessions")
			api.DoErrorResponse(ctx, w)
			return
		}

		totalResults := len(sessions)
		start := (page - 1) * resultsPerPage
		end := start + resultsPerPage
		if start > totalResults {
			start = totalResults
		}
		if end > totalResults {
			end = totalResults
		}

		pageItems := sessions[start:end]
		items := make([]Race, len(pageItems))
		for i, session := range pageItems {
			items[i] = raceFromDriverSession(session)
		}

		api.DoOKListResponse(ctx, items, page, resultsPerPage, totalResults, w)
	})
}