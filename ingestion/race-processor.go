package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

const maxSearchWindow = 90 * 24 * time.Hour

type DriverStore interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
}

type IRacingClient interface {
	SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...iracing.SearchOption) ([]iracing.SessionResult, error)
}

type RaceProcessor struct {
	driverStore   DriverStore
	iracingClient IRacingClient
	now           func() time.Time
}

func NewRaceProcessor(driverStore DriverStore, iracingClient IRacingClient) *RaceProcessor {
	return &RaceProcessor{
		driverStore:   driverStore,
		iracingClient: iracingClient,
		now:           time.Now,
	}
}

func (r *RaceProcessor) IngestRaces(ctx context.Context, request RaceIngestionRequest) error {
	logger := zerolog.Ctx(ctx)

	driver, err := r.driverStore.GetDriver(ctx, request.DriverID)
	if err != nil {
		return fmt.Errorf("getting driver: %w", err)
	}
	if driver == nil {
		return fmt.Errorf("driver %d not found", request.DriverID)
	}

	rangeBegin := driver.MemberSince
	if driver.RacesIngestedTo != nil {
		rangeBegin = *driver.RacesIngestedTo
	}

	now := r.now()
	rangeEnd := rangeBegin.Add(maxSearchWindow)
	if rangeEnd.After(now) {
		rangeEnd = now
	}

	logger.Debug().
		Int64("driverID", request.DriverID).
		Time("rangeBegin", rangeBegin).
		Time("rangeEnd", rangeEnd).
		Msg("searching for race results")

	results, err := r.iracingClient.SearchSeriesResults(ctx, request.IRacingAccessToken, rangeBegin, rangeEnd,
		iracing.WithCustomerID(request.DriverID),
		iracing.WithEventTypes(iracing.EventTypeRace),
	)
	if err != nil {
		return fmt.Errorf("searching series results: %w", err)
	}

	previewResults := results
	if len(previewResults) > 5 {
		previewResults = previewResults[:5]
	}
	logger.Info().
		Int64("driverID", request.DriverID).
		Int("resultsCount", len(results)).
		Interface("resultsPreview", previewResults).
		Msg("received race results")

	return nil
}
