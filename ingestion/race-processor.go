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

const mainEventSessionNumber = 0
const actionIngestionFailedStaleCredentials = "ingestionFailedStaleCredentials"

type persistResult int

const (
	persistOK persistResult = iota
	persistDisconnected
)

type raceResult int

const (
	raceProcessed raceResult = iota
	raceSkipped
	raceDisconnected
)

type RaceReadyMsg struct {
	RaceID int64 `json:"raceId"`
}

type Store interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
	GetDriverSession(ctx context.Context, driverID int64, startTime time.Time) (*store.DriverSession, error)
	GetSession(ctx context.Context, subsessionID int64) (*store.Session, error)
	GetSessionDrivers(ctx context.Context, subsessionID int64) ([]store.SessionDriver, error)
	UpdateDriverRacesIngestedTo(ctx context.Context, driverID int64, racesIngestedTo time.Time) error
	PersistSessionData(ctx context.Context, data store.SessionDataInsertion) error
}

type IRacingClient interface {
	SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...iracing.SearchOption) ([]iracing.SeriesResult, error)
	GetSessionResults(ctx context.Context, accessToken string, subsessionID int64, opts ...iracing.GetSessionResultsOption) (*iracing.SessionResult, error)
	GetLapData(ctx context.Context, accessToken string, subsessionID int64, simsessionNumber int, opts ...iracing.GetLapDataOption) (*iracing.LapDataResponse, error)
}

type Pusher interface {
	Push(ctx context.Context, connectionID string, actionType string, payload any) (bool, error)
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
	raceConsumptionConcurrency int
	now                        func() time.Time
}

func NewRaceProcessor(store Store, iracingClient IRacingClient, pusher Pusher, opts ...RaceProcessorOption) *RaceProcessor {
	r := &RaceProcessor{
		store:                      store,
		iracingClient:              iracingClient,
		pusher:                     pusher,
		searchWindowDuration:       time.Hour * 24 * 10,
		raceConsumptionConcurrency: DefaultRaceConsumptionConcurrency,
		now:                        time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *RaceProcessor) IngestRaces(ctx context.Context, request RaceIngestionRequest) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := zerolog.Ctx(ctx)

	driver, err := r.store.GetDriver(ctx, request.DriverID)
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
	rangeEnd := rangeBegin.Add(r.searchWindowDuration)
	if rangeEnd.After(now) {
		rangeEnd = now
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
			return nil
		}
		return fmt.Errorf("searching series results: %w", err)
	}

	raceCount := 0
	newRaceCount := 0
	var errs []error

	collectionChan := make(chan collectionResult)
	collectorDone := sync.WaitGroup{}

	collectorDone.Add(1)
	go func() {
		defer collectorDone.Done()
		for {
			select {
			case result, ok := <-collectionChan:
				if !ok {
					// channels closed, were done
					return
				}
				raceCount += result.race
				newRaceCount += result.newRace
				if result.err != nil {
					logger.Err(result.err).Msg("error during ingestion, bailing out")
					errs = append(errs, result.err)
					// any errors == bail out
					cancel()
				}
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
		return errors.Join(errs...)
	}

	err = r.store.UpdateDriverRacesIngestedTo(ctx, driver.DriverID, rangeEnd)
	if err != nil {
		return fmt.Errorf("updating driver ingested to: %w", err)
	}
	// todo - trigger another round ingestion if our range is in the past
	logger.Info().Int("raceCount", raceCount).Int("newRaceCount", newRaceCount).Msg("ingested races")
	return nil
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
		result, err := r.processExistingSession(ctx, existingSession, driver.DriverID, request.NotifyConnectionID)
		if err != nil {
			segmentErr = err
			collectorChan <- collectionResult{err: err}
			return
		}
		if result == raceDisconnected {
			logger.Warn().Int64("driverID", driver.DriverID).Msg("user disconnected during ingestion, discontinuing")
			segmentErr = fmt.Errorf("driver %d disconnected during ingestion", driver.DriverID)
			collectorChan <- collectionResult{err: segmentErr}
			return
		}
		return
	}

	collectorChan <- collectionResult{newRace: 1}
	result, err := r.processNewSession(ctx, race.SubsessionID, driver.DriverID, request.IRacingAccessToken, request.NotifyConnectionID)
	if err != nil {
		segmentErr = err
		collectorChan <- collectionResult{err: err}
		return
	}
	if result == raceDisconnected {
		logger.Warn().Int64("driverID", driver.DriverID).Msg("user disconnected during ingestion, discontinuing")
		segmentErr = fmt.Errorf("driver %d disconnected during ingestion", driver.DriverID)
		collectorChan <- collectionResult{err: segmentErr}
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

func (r *RaceProcessor) persistAndNotify(ctx context.Context, insertions store.SessionDataInsertion, connectionID string) (persistResult, error) {
	if !insertions.HasData() {
		return persistOK, nil
	}

	if err := r.store.PersistSessionData(ctx, insertions); err != nil {
		return persistOK, fmt.Errorf("persisting data: %w", err)
	}

	for _, driverSession := range insertions.DriverSessionEntries {
		raceID := store.DriverRaceIDFromTime(driverSession.StartTime)
		connected, err := r.pusher.Push(ctx, connectionID, "raceIngested", RaceReadyMsg{raceID})
		if err != nil {
			return persistOK, fmt.Errorf("notifying race ingested: %w", err)
		}
		if !connected {
			return persistDisconnected, nil
		}
	}
	return persistOK, nil
}

func (r *RaceProcessor) processExistingSession(ctx context.Context, existingSession *store.Session, driverID int64, connectionID string) (res raceResult, err error) {
	ctx, segment := xray.BeginSubsegment(ctx, "ProcessExistingSession")
	defer func() { segment.Close(err) }()

	logger := zerolog.Ctx(ctx)

	existingDriverSession, err := r.store.GetDriverSession(ctx, driverID, existingSession.StartTime)
	if err != nil {
		return raceProcessed, fmt.Errorf("checking if driver session already exists: %w", err)
	}
	if existingDriverSession != nil {
		logger.Info().Int64("sessionID", existingSession.SubsessionID).Int64("driverID", driverID).Msg("driver session already ingested")
		return raceSkipped, nil
	}

	sessionDrivers, err := r.store.GetSessionDrivers(ctx, existingSession.SubsessionID)
	if err != nil {
		return raceProcessed, fmt.Errorf("getting session drivers: %w", err)
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

	result, err := r.persistAndNotify(ctx, insertions, connectionID)
	if err != nil {
		return raceProcessed, err
	}
	if result == persistDisconnected {
		return raceDisconnected, nil
	}
	return raceProcessed, nil
}

func (r *RaceProcessor) processNewSession(ctx context.Context, subsessionID int64, driverID int64, accessToken, connectionID string) (raceResult, error) {
	logger := zerolog.Ctx(ctx)

	sessionResult, err := r.iracingClient.GetSessionResults(ctx, accessToken, subsessionID, iracing.WithIncludeLicenses(true))
	if err != nil {
		if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
			r.notifyStaleCredentials(ctx, connectionID)
			return raceDisconnected, nil
		}
		return raceProcessed, fmt.Errorf("pulling session results: %w", err)
	}
	logger.Trace().Interface("result", sessionResult).Msg("got session result")

	var insertions store.SessionDataInsertion

	for _, simSession := range sessionResult.SessionResults {
		if simSession.SimsessionNumber != mainEventSessionNumber {
			continue
		}

		insertions.SessionEntries = append(insertions.SessionEntries, store.Session{
			SubsessionID: sessionResult.SubsessionID,
			TrackID:      sessionResult.Track.TrackID,
			StartTime:    sessionResult.StartTime,
			CarClasses:   mapCarClasses(sessionResult.SubsessionID, sessionResult.CarClasses),
		})

		for _, driverResult := range simSession.Results {
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
					return raceProcessed, fmt.Errorf("checking if driver session already exists: %w", err)
				}

				if existingDriverSession != nil {
					logger.Info().Int64("sessionID", subsessionID).Int64("driverID", driverResult.CustID).Msg("driver session already ingested")
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

			// note, team events are going to throw a wrinkle at things since you have to look up laps by team for those, but I only race solo so it is what it is for now
			var laps *iracing.LapDataResponse
			err = xray.Capture(ctx, "FetchDriverLaps", func(lapCtx context.Context) error {
				_ = xray.AddAnnotation(lapCtx, "driverID", driverResult.CustID)
				var lapErr error
				laps, lapErr = r.iracingClient.GetLapData(lapCtx, accessToken, subsessionID, simSession.SimsessionNumber, iracing.WithCustomerIDLap(driverResult.CustID))
				return lapErr
			})
			if err != nil {
				if errors.Is(err, iracing.ErrUpstreamUnauthorized) {
					r.notifyStaleCredentials(ctx, connectionID)
					return raceDisconnected, nil
				}
				return raceProcessed, fmt.Errorf("pulling driver laps: %w", err)
			}
			logger.Trace().Interface("lapData", laps).Msg("got laps result")

			for _, lap := range laps.Laps {
				insertions.SessionDriverLapEntries = append(insertions.SessionDriverLapEntries, store.SessionDriverLap{
					SubsessionID: sessionResult.SubsessionID,
					DriverID:     driverResult.CustID,
					LapNumber:    lap.LapNumber,
					LapTime:      iracing.LapTimeToDuration(lap.LapTime),
					Flags:        lap.Flags,
					Incident:     lap.Incident,
					LapEvents:    lap.LapEvents,
				})
			}
		}

		result, err := r.persistAndNotify(ctx, insertions, connectionID)
		if err != nil {
			return raceProcessed, err
		}
		if result == persistDisconnected {
			return raceDisconnected, nil
		}
	}
	return raceProcessed, nil
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
