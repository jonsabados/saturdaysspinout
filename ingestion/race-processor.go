package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

const mainEventSessionNumber = 0

type Store interface {
	GetDriver(ctx context.Context, driverID int64) (*store.Driver, error)
	GetDriverSession(ctx context.Context, driverID int64, startTime time.Time) (*store.DriverSession, error)
	GetSession(ctx context.Context, subsessionID int64) (*store.Session, error)
	UpdateDriverRacesIngestedTo(ctx context.Context, driverID int64, racesIngestedTo time.Time) error
	PersistSessionData(ctx context.Context, data store.SessionDataInsertion) error
}

type IRacingClient interface {
	SearchSeriesResults(ctx context.Context, accessToken string, finishRangeBegin, finishRangeEnd time.Time, opts ...iracing.SearchOption) ([]iracing.SeriesResult, error)
	GetSessionResults(ctx context.Context, accessToken string, subsessionID int64, opts ...iracing.GetSessionResultsOption) (*iracing.SessionResult, error)
	GetLapData(ctx context.Context, accessToken string, subsessionID int64, simsessionNumber int, opts ...iracing.GetLapDataOption) (*iracing.LapDataResponse, error)
}

type RaceProcessorOption func(*RaceProcessor)

func WithSearchWindowInDays(days int) RaceProcessorOption {
	return func(r *RaceProcessor) {
		r.searchWindowDuration = time.Hour * 24 * time.Duration(days)
	}
}

type RaceProcessor struct {
	store                Store
	iracingClient        IRacingClient
	searchWindowDuration time.Duration
	now                  func() time.Time
}

func NewRaceProcessor(store Store, iracingClient IRacingClient, opts ...RaceProcessorOption) *RaceProcessor {
	r := &RaceProcessor{
		store:                store,
		iracingClient:        iracingClient,
		searchWindowDuration: time.Hour * 24 * 10,
		now:                  time.Now,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *RaceProcessor) IngestRaces(ctx context.Context, request RaceIngestionRequest) error {
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
		return fmt.Errorf("searching series results: %w", err)
	}

	raceCount := 0
	newRaceCount := 0

	for _, race := range results {
		raceCount++
		logger.Trace().Interface("race", race).Msg("processing race")

		existingRecord, err := r.store.GetSession(ctx, race.SubsessionID)
		if err != nil {
			return fmt.Errorf("checking session record: %w", err)
		}
		// it's possible the session itself will have been ingested for a prior driver
		insertSessionData := existingRecord == nil
		if !insertSessionData {
			logger.Info().Int64("sessionID", race.SubsessionID).Msg("session already ingested")
		} else {
			newRaceCount++
		}

		sessionResult, err := r.iracingClient.GetSessionResults(ctx, request.IRacingAccessToken, race.SubsessionID, iracing.WithIncludeLicenses(true))
		// TODO for this and other errors, check for ErrUpstreamUnauthorized to tell the frontend to refresh and try again
		if err != nil {
			return fmt.Errorf("pulling session results: %w", err)
		}
		logger.Trace().Interface("result", sessionResult).Msg("got session result")

		var insertions store.SessionDataInsertion

		for _, simSession := range sessionResult.SessionResults {
			// we only care about the actual race
			if simSession.SimsessionNumber != mainEventSessionNumber {
				continue
			}
			if insertSessionData {
				insertions.SessionEntries = append(insertions.SessionEntries, store.Session{
					SubsessionID: sessionResult.SubsessionID,
					TrackID:      sessionResult.Track.TrackID,
					StartTime:    sessionResult.StartTime,
					CarClasses:   mapCarClasses(sessionResult.SubsessionID, sessionResult.CarClasses),
				})
			}

			for _, driverResult := range simSession.Results {
				if insertSessionData {
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
				}

				if driverResult.CustID == driver.DriverID {
					existingRecord, err := r.store.GetDriverSession(ctx, driver.DriverID, sessionResult.StartTime)
					if err != nil {
						return fmt.Errorf("checking if driver session already exists: %w", err)
					}

					if existingRecord != nil {
						logger.Info().Int64("sessionID", race.SubsessionID).Int64("driverID", driverResult.CustID).Msg("driver session already ingested")
					} else {
						// TODO - record this for broadcasting to the front end via webhooks after save
						insertions.DriverSessionEntries = append(insertions.DriverSessionEntries, store.DriverSession{
							DriverID:              driver.DriverID,
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

				if insertSessionData {
					// note, team events are going to throw a wrinkle at things since you have to look up laps by team for those, but I only race solo so it is what it is for now
					laps, err := r.iracingClient.GetLapData(ctx, request.IRacingAccessToken, race.SubsessionID, simSession.SimsessionNumber, iracing.WithCustomerIDLap(driverResult.CustID))
					if err != nil {
						return fmt.Errorf("pulling driver laps: %w", err)
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
			}

			err := r.store.PersistSessionData(ctx, insertions)
			if err != nil {
				return fmt.Errorf("persisting data: %w", err)
			}
		}
	}

	err = r.store.UpdateDriverRacesIngestedTo(ctx, driver.DriverID, rangeEnd)
	if err != nil {
		return fmt.Errorf("updating driver ingested to: %w", err)
	}
	// todo - trigger another round ingestion if our range is in the past
	logger.Info().Int("raceCount", raceCount).Int("newRaceCount", newRaceCount).Msg("ingested races")
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
