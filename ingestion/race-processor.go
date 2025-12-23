package ingestion

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

const DefaultRaceConsumptionConcurrency = 2
const DefaultLapConsumptionConcurrency = 2 // note, this is per race consumption thread, so with 2 and 2 we could be making 4 simultaneous calls to iRacing

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
	GetSession(ctx context.Context, subsessionID int64) (*store.Session, error)
	GetSessionDrivers(ctx context.Context, subsessionID int64) ([]store.SessionDriver, error)
	UpdateDriverRacesIngestedTo(ctx context.Context, driverID int64, racesIngestedTo time.Time) error
	PersistSessionData(ctx context.Context, data store.SessionDataInsertion) error
	AcquireIngestionLock(ctx context.Context, driverID int64, lockDuration time.Duration) (bool, error)
	ReleaseIngestionLock(ctx context.Context, driverID int64) error
}

type IRacingClient interface {
	SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...iracing.SearchOption) ([]iracing.SeriesResult, error)
	GetSessionResults(ctx context.Context, accessToken string, subsessionID int64, opts ...iracing.GetSessionResultsOption) (*iracing.SessionResult, error)
	GetLapData(ctx context.Context, accessToken string, subsessionID int64, simsessionNumber int, opts ...iracing.GetLapDataOption) (*iracing.LapDataResponse, error)
}

type Pusher interface {
	Push(ctx context.Context, connectionID string, actionType string, payload any) (bool, error)
	Broadcast(ctx context.Context, driverID int64, actionType string, payload any) error
}

type EventDispatcher interface {
	PublishEvent(ctx context.Context, event any) error
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

func WithLapConsumptionConcurrency(n int) RaceProcessorOption {
	return func(r *RaceProcessor) {
		r.lapConsumptionConcurrency = n
	}
}

type RaceProcessor struct {
	store                      Store
	iracingClient              IRacingClient
	searchWindowDuration       time.Duration
	pusher                     Pusher
	eventDispatcher            EventDispatcher
	raceConsumptionConcurrency int
	lapConsumptionConcurrency  int
	lockDuration               time.Duration
	now                        func() time.Time
}

func NewRaceProcessor(store Store, iracingClient IRacingClient, pusher Pusher, eventDispatcher EventDispatcher, lockDuration time.Duration, opts ...RaceProcessorOption) *RaceProcessor {
	r := &RaceProcessor{
		store:                      store,
		iracingClient:              iracingClient,
		pusher:                     pusher,
		eventDispatcher:            eventDispatcher,
		searchWindowDuration:       time.Hour * 24 * 10,
		raceConsumptionConcurrency: DefaultRaceConsumptionConcurrency,
		lapConsumptionConcurrency:  DefaultLapConsumptionConcurrency,
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
		// Release lock so SQS backoff can handle retry
		if releaseErr := r.store.ReleaseIngestionLock(ctx, request.DriverID); releaseErr != nil {
			logger.Err(releaseErr).Msg("failed to release ingestion lock after error")
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
		if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
			r.notifyStaleCredentials(ctx, request.NotifyConnectionID)
			return false, nil
		}
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
					r.ingestRace(ctx, driver, request, race, collectionChan)
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

func (r *RaceProcessor) ingestRace(ctx context.Context, driver *store.Driver, request RaceIngestionRequest, race iracing.SeriesResult, collectorChan chan collectionResult) {
	ctx, segment := xray.BeginSubsegment(ctx, "IngestRace")
	var segmentErr error
	defer func() { segment.Close(segmentErr) }()
	_ = xray.AddAnnotation(ctx, "subsessionID", race.SubsessionID)

	logger := zerolog.Ctx(ctx)
	logger.Trace().Interface("race", race).Msg("processing race")

	collectorChan <- collectionResult{race: 1}

	existingSession, err := r.store.GetSession(ctx, race.SubsessionID)
	if err != nil {
		segmentErr = err
		collectorChan <- collectionResult{
			err: fmt.Errorf("checking session record: %w", err),
		}
		return
	}

	if existingSession != nil {
		logger.Info().Int64("sessionID", race.SubsessionID).Msg("session already ingested")
		if err := r.processExistingSession(ctx, existingSession, driver.DriverID); err != nil {
			segmentErr = err
			collectorChan <- collectionResult{err: err}
		}
		return
	}

	collectorChan <- collectionResult{newRace: 1}
	if err := r.processNewSession(ctx, request, race.SubsessionID); err != nil {
		segmentErr = err
		collectorChan <- collectionResult{err: err}
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

func (r *RaceProcessor) persistAndNotify(ctx context.Context, insertions store.SessionDataInsertion) error {
	if !insertions.HasData() {
		return nil
	}

	if err := r.store.PersistSessionData(ctx, insertions); err != nil {
		return fmt.Errorf("persisting data: %w", err)
	}

	for _, driverSession := range insertions.DriverSessionEntries {
		raceID := store.DriverRaceIDFromTime(driverSession.StartTime)
		if err := r.pusher.Broadcast(ctx, driverSession.DriverID, "raceIngested", RaceReadyMsg{raceID}); err != nil {
			return fmt.Errorf("broadcasting race ingested: %w", err)
		}
	}
	return nil
}

func (r *RaceProcessor) processExistingSession(ctx context.Context, existingSession *store.Session, driverID int64) error {
	var err error
	ctx, segment := xray.BeginSubsegment(ctx, "ProcessExistingSession")
	defer func() { segment.Close(err) }()

	logger := zerolog.Ctx(ctx)

	existingDriverSession, err := r.store.GetDriverSession(ctx, driverID, existingSession.StartTime)
	if err != nil {
		return fmt.Errorf("checking if driver session already exists: %w", err)
	}
	if existingDriverSession != nil {
		logger.Info().Int64("sessionID", existingSession.SubsessionID).Int64("driverID", driverID).Msg("driver session already ingested")
		return nil
	}

	sessionDrivers, err := r.store.GetSessionDrivers(ctx, existingSession.SubsessionID)
	if err != nil {
		return fmt.Errorf("getting session drivers: %w", err)
	}

	var insertions store.SessionDataInsertion
	for _, sd := range sessionDrivers {
		if sd.DriverID == driverID {
			insertions.DriverSessionEntries = append(insertions.DriverSessionEntries, store.DriverSession{
				DriverID:              driverID,
				SubsessionID:          existingSession.SubsessionID,
				TrackID:               existingSession.TrackID,
				CarID:                 sd.CarID,
				StartTime:             existingSession.StartTime,
				StartPosition:         sd.StartPosition,
				StartPositionInClass:  sd.StartPositionInClass,
				FinishPosition:        sd.FinishPosition,
				FinishPositionInClass: sd.FinishPositionInClass,
				Incidents:             sd.Incidents,
				OldCPI:                sd.OldCPI,
				NewCPI:                sd.NewCPI,
				OldIRating:            sd.OldIRating,
				NewIRating:            sd.NewIRating,
				ReasonOut:             sd.ReasonOut,
			})
			break
		}
	}

	err = r.persistAndNotify(ctx, insertions)
	return err
}

func findRaceSession(sessions []iracing.SimSessionResult) *iracing.SimSessionResult {
	for i := range sessions {
		if sessions[i].SimsessionNumber == mainEventSessionNumber {
			return &sessions[i]
		}
	}
	return nil
}

func (r *RaceProcessor) processNewSession(ctx context.Context, request RaceIngestionRequest, subsessionID int64) error {
	logger := zerolog.Ctx(ctx)

	sessionResult, err := r.iracingClient.GetSessionResults(ctx, request.IRacingAccessToken, subsessionID, iracing.WithIncludeLicenses(true))
	if err != nil {
		if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
			r.notifyStaleCredentials(ctx, request.NotifyConnectionID)
			return nil
		}
		return fmt.Errorf("pulling session results: %w", err)
	}
	logger.Trace().Interface("result", sessionResult).Msg("got session result")

	raceSession := findRaceSession(sessionResult.SessionResults)
	if raceSession == nil {
		logger.Warn().Int64("subsessionID", subsessionID).Msg("no race session found in session results")
		return nil
	}

	var insertions store.SessionDataInsertion

	insertions.SessionEntries = append(insertions.SessionEntries, store.Session{
		SubsessionID: sessionResult.SubsessionID,
		TrackID:      sessionResult.Track.TrackID,
		StartTime:    sessionResult.StartTime,
		CarClasses:   mapCarClasses(sessionResult.SubsessionID, sessionResult.CarClasses),
	})

	// first lets process the session level stats - these are all just in memory operations
	for _, driverResult := range raceSession.Results {
		err := r.processDriverSessionResults(ctx, subsessionID, request.DriverID, &insertions, sessionResult, driverResult)
		if err != nil {
			return err
		}
	}

	// now laps, which is going to involve some concurrency
	laps, err := r.processLaps(ctx, request, raceSession, sessionResult)
	if err != nil {
		if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
			r.notifyStaleCredentials(ctx, request.NotifyConnectionID)
			return nil
		}
		return err
	}
	insertions.SessionDriverLapEntries = append(insertions.SessionDriverLapEntries, laps...)

	return r.persistAndNotify(ctx, insertions)
}

func (r *RaceProcessor) processLaps(ctx context.Context, request RaceIngestionRequest, raceSession *iracing.SimSessionResult, sessionResult *iracing.SessionResult) ([]store.SessionDriverLap, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var ret []store.SessionDriverLap
	var errs []error

	resultChan := make(chan lapCollectionResult)
	collectorDone := sync.WaitGroup{}
	collectorDone.Add(1)
	go func() {
		defer collectorDone.Done()
		for result := range resultChan {
			if result.err != nil {
				cancel()
				errs = append(errs, result.err)
			} else {
				ret = append(ret, result.laps...)
			}
		}
	}()

	workChan := make(chan iracing.DriverResult)
	workersDone := sync.WaitGroup{}

	for i := 0; i < r.lapConsumptionConcurrency; i++ {
		workersDone.Add(1)
		go func() {
			defer workersDone.Done()
			for {
				select {
				case driverResult, ok := <-workChan:
					if !ok {
						return
					}
					driversLaps, err := r.fetchDriverLaps(ctx, request, sessionResult.SubsessionID, raceSession.SimsessionNumber, driverResult)
					resultChan <- lapCollectionResult{
						laps: driversLaps,
						err:  err,
					}
				case <-ctx.Done():
					resultChan <- lapCollectionResult{err: ctx.Err()}
					return
				}
			}
		}()
	}

	running := true
	for _, driverResult := range raceSession.Results {
		if !running {
			break
		}
		select {
		case workChan <- driverResult:
		case <-ctx.Done():
			running = false
		}
	}
	close(workChan)
	workersDone.Wait()
	close(resultChan)
	collectorDone.Wait()

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return ret, nil
}

func (r *RaceProcessor) fetchDriverLaps(ctx context.Context, request RaceIngestionRequest, subsessionID int64, sessionNumber int, driverResult iracing.DriverResult) ([]store.SessionDriverLap, error) {
	logger := zerolog.Ctx(ctx)
	var ret []store.SessionDriverLap

	// note, team events are going to throw a wrinkle at things since you have to look up laps by team for those, but I only race solo so it is what it is for now
	var laps *iracing.LapDataResponse
	err := xray.Capture(ctx, "FetchDriverLaps", func(lapCtx context.Context) error {
		_ = xray.AddAnnotation(lapCtx, "driverID", driverResult.CustID)
		var lapErr error
		laps, lapErr = r.iracingClient.GetLapData(lapCtx, request.IRacingAccessToken, subsessionID, sessionNumber, iracing.WithCustomerIDLap(driverResult.CustID))
		return lapErr
	})
	if err != nil {
		return nil, err
	}
	logger.Trace().Interface("lapData", laps).Msg("got laps result")

	for _, lap := range laps.Laps {
		ret = append(ret, store.SessionDriverLap{
			SubsessionID: subsessionID,
			DriverID:     driverResult.CustID,
			LapNumber:    lap.LapNumber,
			LapTime:      iracing.LapTimeToDuration(lap.LapTime),
			Flags:        lap.Flags,
			Incident:     lap.Incident,
			LapEvents:    lap.LapEvents,
		})
	}
	return ret, nil
}

func (r *RaceProcessor) processDriverSessionResults(ctx context.Context, subsessionID int64, driverID int64, insertions *store.SessionDataInsertion, sessionResult *iracing.SessionResult, driverResult iracing.DriverResult) error {
	insertions.SessionDriverEntries = append(insertions.SessionDriverEntries, store.SessionDriver{
		SubsessionID:          sessionResult.SubsessionID,
		DriverID:              driverResult.CustID,
		CarID:                 driverResult.CarID,
		StartPosition:         driverResult.StartingPosition,
		StartPositionInClass:  driverResult.StartingPositionInClass,
		FinishPosition:        driverResult.FinishPosition,
		FinishPositionInClass: driverResult.FinishPositionInClass,
		Incidents:             driverResult.Incidents,
		OldCPI:                driverResult.OldCPI,
		NewCPI:                driverResult.NewCPI,
		OldIRating:            driverResult.OldIRating,
		NewIRating:            driverResult.NewIRating,
		ReasonOut:             driverResult.ReasonOut,
		AI:                    driverResult.AI,
	})

	if driverResult.CustID == driverID {
		existingDriverSession, err := r.store.GetDriverSession(ctx, driverID, sessionResult.StartTime)
		if err != nil {
			return fmt.Errorf("checking if driver session already exists: %w", err)
		}

		if existingDriverSession != nil {
			zerolog.Ctx(ctx).Info().Int64("sessionID", subsessionID).Int64("driverID", driverResult.CustID).Msg("driver session already ingested")
		} else {
			insertions.DriverSessionEntries = append(insertions.DriverSessionEntries, store.DriverSession{
				DriverID:              driverID,
				SubsessionID:          sessionResult.SubsessionID,
				TrackID:               sessionResult.Track.TrackID,
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
				ReasonOut:             driverResult.ReasonOut,
			})
		}
	}
	return nil
}

func mapCarClasses(subsessionID int64, classes []iracing.CarClass) []store.SessionCarClass {
	result := make([]store.SessionCarClass, len(classes))
	for i, class := range classes {
		cars := make([]store.SessionCarClassCar, len(class.CarsInClass))
		for j, car := range class.CarsInClass {
			cars[j] = store.SessionCarClassCar{
				SubsessionID: subsessionID,
				CarClassID:   int64(class.CarClassID),
				CarID:        int64(car.CarID),
			}
		}
		result[i] = store.SessionCarClass{
			SubsessionID:    subsessionID,
			CarClassID:      int64(class.CarClassID),
			StrengthOfField: class.StrengthOfField,
			NumberOfEntries: class.NumEntries,
			Cars:            cars,
		}
	}
	return result
}

type collectionResult struct {
	newRace int
	race    int
	err     error
}

type lapCollectionResult struct {
	laps []store.SessionDriverLap
	err  error
}
