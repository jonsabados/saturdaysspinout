package ingestion

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/metrics"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

const DefaultRaceConsumptionConcurrency = 2

const mainEventSessionNumber = 0
const actionIngestionFailedStaleCredentials = "ingestionFailedStaleCredentials"

type RaceReadyMsg struct {
	RaceID int64 `json:"raceId"`
}

type ChunkCompleteMsg struct {
	IngestedTo time.Time `json:"ingestedTo"`
}

type Store interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
	GetDriverSession(ctx context.Context, driverID int64, startTime time.Time) (*store.DriverSession, error)
	UpdateDriverRacesIngestedTo(ctx context.Context, driverID int64, racesIngestedTo time.Time) error
	SaveDriverSessions(ctx context.Context, sessions []store.DriverSession) error
	AcquireIngestionLock(ctx context.Context, driverID int64, lockDuration time.Duration) (bool, error)
	ReleaseIngestionLock(ctx context.Context, driverID int64) error
}

type IRacingClient interface {
	SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...iracing.SearchOption) ([]iracing.SeriesResult, error)
	GetSessionResults(ctx context.Context, accessToken string, subsessionID int64, opts ...iracing.GetSessionResultsOption) (*iracing.SessionResult, error)
}

type Pusher interface {
	Push(ctx context.Context, connectionID string, actionType string, payload any) (bool, error)
	Broadcast(ctx context.Context, driverID int64, actionType string, payload any) error
}

type EventDispatcher interface {
	PublishEvent(ctx context.Context, event any) error
}

type MetricsClient interface {
	EmitCount(ctx context.Context, name string, count int) error
}

type RaceProcessorOption func(*RaceProcessor)

func WithSearchWindowInDays(days int) RaceProcessorOption {
	return func(r *RaceProcessor) {
		r.searchWindowDuration = time.Hour * 24 * time.Duration(days)
	}
}

func WithRaceConsumptionConcurrency(n int) RaceProcessorOption {
	return func(r *RaceProcessor) {
		r.raceConsumptionConcurrency = n
	}
}

type RaceProcessor struct {
	store                      Store
	iracingClient              IRacingClient
	searchWindowDuration       time.Duration
	pusher                     Pusher
	eventDispatcher            EventDispatcher
	metricsClient              MetricsClient
	raceConsumptionConcurrency int
	lockDuration               time.Duration
	now                        func() time.Time
}

func NewRaceProcessor(store Store, iracingClient IRacingClient, pusher Pusher, eventDispatcher EventDispatcher, metricsClient MetricsClient, lockDuration time.Duration, opts ...RaceProcessorOption) *RaceProcessor {
	r := &RaceProcessor{
		store:                      store,
		iracingClient:              iracingClient,
		pusher:                     pusher,
		eventDispatcher:            eventDispatcher,
		metricsClient:              metricsClient,
		searchWindowDuration:       time.Hour * 24 * 10,
		raceConsumptionConcurrency: DefaultRaceConsumptionConcurrency,
		lockDuration:               lockDuration,
		now:                        time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *RaceProcessor) IngestRaces(ctx context.Context, request RaceIngestionRequest) error {
	logger := zerolog.Ctx(ctx)

	acquired, err := r.store.AcquireIngestionLock(ctx, request.DriverID, r.lockDuration)
	if err != nil {
		return fmt.Errorf("acquiring ingestion lock: %w", err)
	}
	if !acquired {
		logger.Warn().Int64("driverID", request.DriverID).Msg("ingestion lock already held, skipping")
		return nil
	}

	needsRecursion, err := r.doIngestRaces(ctx, request)
	if err != nil {
		// Release lock so SQS backoff can handle retry (or client can retry immediately for stale credentials)
		if releaseErr := r.store.ReleaseIngestionLock(ctx, request.DriverID); releaseErr != nil {
			logger.Err(releaseErr).Msg("failed to release ingestion lock after error")
		}
		if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
			r.notifyStaleCredentials(ctx, request.NotifyConnectionID)
			return nil
		}
		return err
	}

	if needsRecursion {
		if err := r.store.ReleaseIngestionLock(ctx, request.DriverID); err != nil {
			return fmt.Errorf("releasing ingestion lock: %w", err)
		}
		logger.Info().Msg("more races to ingest, dispatching another round")
		if err := r.eventDispatcher.PublishEvent(ctx, request); err != nil {
			return fmt.Errorf("dispatching next ingestion round: %w", err)
		}
	}
	// If up to date, let the lock expire naturally (cooldown period)

	return nil
}

func (r *RaceProcessor) doIngestRaces(ctx context.Context, request RaceIngestionRequest) (needsRecursion bool, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := zerolog.Ctx(ctx)

	driver, err := r.store.GetDriver(ctx, request.DriverID)
	if err != nil {
		return false, fmt.Errorf("getting driver: %w", err)
	}
	if driver == nil {
		return false, fmt.Errorf("driver %d not found", request.DriverID)
	}

	rangeBegin := driver.MemberSince
	if driver.RacesIngestedTo != nil {
		rangeBegin = *driver.RacesIngestedTo
	}

	willBeUpToDate := false
	now := r.now()
	rangeEnd := rangeBegin.Add(r.searchWindowDuration)
	if rangeEnd.After(now) {
		rangeEnd = now
		willBeUpToDate = true
	}

	logger.Info().
		Int64("driverID", request.DriverID).
		Time("rangeBegin", rangeBegin).
		Time("rangeEnd", rangeEnd).
		Msg("searching for race results")

	results, err := r.iracingClient.SearchSeriesResults(ctx, request.IRacingAccessToken, rangeBegin, rangeEnd,
		iracing.WithCustomerID(request.DriverID),
		iracing.WithEventTypes(iracing.EventTypeRace),
	)
	if err != nil {
		return false, fmt.Errorf("searching series results: %w", err)
	}

	raceCount := 0
	newRaceCount := 0
	var errs []error

	collectionChan := make(chan collectionResult)
	collectorDone := sync.WaitGroup{}

	collectorDone.Add(1)
	go func() {
		defer collectorDone.Done()
		for result := range collectionChan {
			raceCount += result.race
			newRaceCount += result.newRace
			if result.err != nil {
				logger.Err(result.err).Msg("error during ingestion, bailing out")
				errs = append(errs, result.err)
				// any errors == bail out
				cancel()
			}
		}
	}()

	racesChan := make(chan iracing.SeriesResult)
	racesDone := sync.WaitGroup{}

	insertionMutex := sync.Mutex{}

	for i := 0; i < r.raceConsumptionConcurrency; i++ {
		racesDone.Add(1)
		go func() {
			defer racesDone.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case race, ok := <-racesChan:
					if !ok {
						return
					}
					r.ingestRace(ctx, &insertionMutex, driver, request, race, collectionChan)
				}
			}
		}()
	}

	running := true
	for _, race := range results {
		if !running {
			break
		}
		select {
		case racesChan <- race:
		case <-ctx.Done():
			running = false
		}
	}

	close(racesChan)
	racesDone.Wait()

	close(collectionChan)
	collectorDone.Wait()

	if len(errs) > 0 {
		return false, errors.Join(errs...)
	}

	if err := r.store.UpdateDriverRacesIngestedTo(ctx, driver.DriverID, rangeEnd); err != nil {
		return false, fmt.Errorf("updating driver ingested to: %w", err)
	}
	if err := r.pusher.Broadcast(ctx, driver.DriverID, "ingestionChunkComplete", ChunkCompleteMsg{IngestedTo: rangeEnd}); err != nil {
		return false, fmt.Errorf("pushing chunk complete notification: %w", err)
	}

	logger.Info().Int("raceCount", raceCount).Int("newRaceCount", newRaceCount).Bool("willBeUpToDate", willBeUpToDate).Msg("ingested races")

	return !willBeUpToDate, nil
}

func (r *RaceProcessor) ingestRace(ctx context.Context, insertionMutex *sync.Mutex, driver *store.Driver, request RaceIngestionRequest, race iracing.SeriesResult, collectorChan chan collectionResult) {
	ctx, segment := xray.BeginSubsegment(ctx, "IngestRace")
	var segmentErr error
	defer func() { segment.Close(segmentErr) }()
	_ = xray.AddAnnotation(ctx, "subsessionID", race.SubsessionID)

	logger := zerolog.Ctx(ctx)
	logger.Trace().Interface("race", race).Msg("processing race")

	if race.DriverChanges {
		logger.Warn().Int64("subsessionID", race.SubsessionID).Msg("skipping team event - team event ingestion not yet supported")
		return
	}

	collectorChan <- collectionResult{race: 1}

	// Fetch session results from iRacing to get this driver's detailed stats
	sessionResult, err := r.iracingClient.GetSessionResults(ctx, request.IRacingAccessToken, race.SubsessionID, iracing.WithIncludeLicenses(true))
	if err != nil {
		segmentErr = err
		collectorChan <- collectionResult{err: fmt.Errorf("pulling session results: %w", err)}
		return
	}

	// Check if we already have this driver's session record
	existingDriverSession, err := r.store.GetDriverSession(ctx, driver.DriverID, sessionResult.StartTime)
	if err != nil {
		segmentErr = err
		collectorChan <- collectionResult{err: fmt.Errorf("checking driver session: %w", err)}
		return
	}
	if existingDriverSession != nil {
		logger.Info().Int64("sessionID", race.SubsessionID).Msg("driver session already ingested")
		return
	}

	collectorChan <- collectionResult{newRace: 1}

	raceSession := findRaceSession(sessionResult.SessionResults)
	if raceSession == nil {
		logger.Warn().Int64("subsessionID", race.SubsessionID).Msg("no race session found in session results")
		return
	}

	// Find this driver's result
	var driverResult *iracing.DriverResult
	for i := range raceSession.Results {
		if raceSession.Results[i].CustID == driver.DriverID {
			driverResult = &raceSession.Results[i]
			break
		}
	}
	if driverResult == nil {
		logger.Warn().Int64("subsessionID", race.SubsessionID).Int64("driverID", driver.DriverID).Msg("driver not found in session results")
		return
	}

	driverSession := store.DriverSession{
		DriverID:              driver.DriverID,
		SubsessionID:          sessionResult.SubsessionID,
		TrackID:               sessionResult.Track.TrackID,
		SeriesID:              int64(sessionResult.SeriesID),
		SeriesName:            sessionResult.SeriesName,
		CarID:                 driverResult.CarID,
		StartTime:             sessionResult.StartTime,
		StartPosition:         driverResult.StartingPosition,
		StartPositionInClass:  driverResult.StartingPositionInClass,
		FinishPosition:        driverResult.FinishPosition,
		FinishPositionInClass: driverResult.FinishPositionInClass,
		Incidents:             driverResult.Incidents,
		OldCPI:                driverResult.OldCPI,
		NewCPI:                driverResult.NewCPI,
		OldIRating:            driverResult.OldIRating,
		NewIRating:            driverResult.NewIRating,
		OldLicenseLevel:       driverResult.OldLicenseLevel,
		NewLicenseLevel:       driverResult.NewLicenseLevel,
		OldSubLevel:           driverResult.OldSubLevel,
		NewSubLevel:           driverResult.NewSubLevel,
		ReasonOut:             driverResult.ReasonOut,
	}

	insertionMutex.Lock()
	if err := r.store.SaveDriverSessions(ctx, []store.DriverSession{driverSession}); err != nil {
		insertionMutex.Unlock()
		segmentErr = err
		collectorChan <- collectionResult{err: fmt.Errorf("saving driver session: %w", err)}
		return
	}
	insertionMutex.Unlock()

	if err := r.metricsClient.EmitCount(ctx, metrics.DriverSessionsIngested, 1); err != nil {
		logger.Warn().Err(err).Msg("failed to emit driver sessions ingested metric")
	}

	raceID := store.DriverRaceIDFromTime(driverSession.StartTime)
	if err := r.pusher.Broadcast(ctx, driver.DriverID, "raceIngested", RaceReadyMsg{raceID}); err != nil {
		segmentErr = err
		collectorChan <- collectionResult{err: fmt.Errorf("broadcasting race ingested: %w", err)}
		return
	}
}

func (r *RaceProcessor) notifyStaleCredentials(ctx context.Context, connectionID string) {
	logger := zerolog.Ctx(ctx)
	if connectionID == "" {
		logger.Warn().Msg("no connection ID to notify of stale credentials")
		return
	}
	_, err := r.pusher.Push(ctx, connectionID, actionIngestionFailedStaleCredentials, nil)
	if err != nil {
		logger.Error().Err(err).Msg("failed to notify client of stale credentials")
	}
}

func findRaceSession(sessions []iracing.SimSessionResult) *iracing.SimSessionResult {
	for i := range sessions {
		if sessions[i].SimsessionNumber == mainEventSessionNumber {
			return &sessions[i]
		}
	}
	return nil
}

type collectionResult struct {
	newRace int
	race    int
	err     error
}
