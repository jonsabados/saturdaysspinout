package driver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/journal"
	"github.com/rs/zerolog"
)

type ListJournalEntriesStore interface {
	List(ctx context.Context, input journal.ListInput) ([]journal.Entry, error)
}

func NewListJournalEntriesEndpoint(journalService ListJournalEntriesStore) http.Handler {
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

		entries, err := journalService.List(ctx, journal.ListInput{
			DriverID: driverID,
			From:     startTime,
			To:       endTime,
		})
		if err != nil {
			logger.Error().Err(err).Int64("driverId", driverID).Msg("failed to list journal entries")
			api.DoErrorResponse(ctx, w)
			return
		}

		// Apply pagination
		totalResults := len(entries)
		start := (page - 1) * resultsPerPage
		end := start + resultsPerPage
		if start > totalResults {
			start = totalResults
		}
		if end > totalResults {
			end = totalResults
		}

		pageItems := entries[start:end]
		items := make([]JournalEntry, len(pageItems))
		for i, entry := range pageItems {
			items[i] = journalEntryFromServiceEntry(entry)
		}

		api.DoOKListResponse(ctx, items, page, resultsPerPage, totalResults, w)
	})
}