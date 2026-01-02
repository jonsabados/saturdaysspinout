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
	GetDriverSessionsByTimeRange(ctx context.Context, driverID int64, from, to time.Time) ([]store.DriverSession, error)
}

func NewGetRacesEndpoint(raceStore GetRacesStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := zerolog.Ctx(ctx)

		errs := api.NewRequestErrors()

		driverID, err := strconv.ParseInt(chi.URLParam(r, api.DriverIDPathParam), 10, 64)
		if err != nil {
			errs = errs.WithFieldError(api.DriverIDPathParam, "must be a valid integer")
		}

		var startTime, endTime time.Time

		startTimeStr := r.URL.Query().Get(api.StartTimeQueryParam)
		if startTimeStr == "" {
			errs = errs.WithFieldError(api.StartTimeQueryParam, "required")
		} else {
			startTime, err = time.Parse(time.RFC3339, startTimeStr)
			if err != nil {
				errs = errs.WithFieldError(api.StartTimeQueryParam, "must be a valid ISO-8601 timestamp")
			}
		}

		endTimeStr := r.URL.Query().Get(api.EndTimeQueryParam)
		if endTimeStr == "" {
			errs = errs.WithFieldError(api.EndTimeQueryParam, "required")
		} else {
			endTime, err = time.Parse(time.RFC3339, endTimeStr)
			if err != nil {
				errs = errs.WithFieldError(api.EndTimeQueryParam, "must be a valid ISO-8601 timestamp")
			}
		}

		page := 1
		if pageStr := r.URL.Query().Get(api.PageQueryParam); pageStr != "" {
			page, err = strconv.Atoi(pageStr)
			if err != nil || page < 1 {
				errs = errs.WithFieldError(api.PageQueryParam, "must be a positive integer")
			}
		}

		resultsPerPage := api.DefaultResultsPerPage
		if rppStr := r.URL.Query().Get(api.ResultsPerPageParam); rppStr != "" {
			resultsPerPage, err = strconv.Atoi(rppStr)
			if err != nil || resultsPerPage < 1 {
				errs = errs.WithFieldError(api.ResultsPerPageParam, "must be a positive integer")
			}
		}

		if errs.HasAnyError() {
			api.DoBadRequestResponse(ctx, errs, w)
			return
		}

		sessions, err := raceStore.GetDriverSessionsByTimeRange(ctx, driverID, startTime, endTime)
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