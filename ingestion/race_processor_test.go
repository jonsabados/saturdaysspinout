package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/metrics"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type getDriverCall struct {
	driverID int64
	result   *store.Driver
	err      error
}

type searchSeriesResultsCall struct {
	finishRangeBegin time.Time
	finishRangeEnd   time.Time
	result           []iracing.SeriesResult
	err              error
}

type getSessionCall struct {
	subsessionID int64
	result       *store.Session
	err          error
}

type getSessionDriversCall struct {
	subsessionID int64
	result       []store.SessionDriver
	err          error
}

type getSessionResultsCall struct {
	subsessionID int64
	result       *iracing.SessionResult
	err          error
}

type getDriverSessionCall struct {
	driverID  int64
	startTime time.Time
	result    *store.DriverSession
	err       error
}

type getLapDataCall struct {
	subsessionID     int64
	simsessionNumber int
	result           *iracing.LapDataResponse
	err              error
}

type persistSessionDataCall struct {
	validate func(t *testing.T, data store.SessionDataInsertion)
	err      error
}

type pushCall struct {
	connectionID string
	actionType   string
	result       bool
	err          error
}

type broadcastCall struct {
	driverID   int64
	actionType string
	payload    any
	err        error
}

type updateDriverRacesIngestedToCall struct {
	driverID        int64
	racesIngestedTo time.Time
	err             error
}

type publishEventCall struct {
	event any
	err   error
}

type emitCountCall struct {
	name  string
	count int
	err   error
}

type acquireIngestionLockCall struct {
	driverID int64
	acquired bool
	err      error
}

type releaseIngestionLockCall struct {
	driverID int64
	err      error
}

func TestRaceProcessor_IngestRaces(t *testing.T) {
	driverID := int64(12345)
	subsessionID := int64(99999)
	memberSince := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	sessionStartTime := time.Date(2024, 6, 10, 18, 0, 0, 0, time.UTC)
	rangeEnd := memberSince.Add(time.Hour * 24 * 10) // default search window
	lockDuration := 15 * time.Minute

	testCases := []struct {
		name string

		request RaceIngestionRequest

		acquireIngestionLockCall        acquireIngestionLockCall
		releaseIngestionLockCall        *releaseIngestionLockCall
		getDriverCall                   *getDriverCall
		searchSeriesResultsCall         *searchSeriesResultsCall
		getSessionCalls                 []getSessionCall
		getSessionDriversCalls          []getSessionDriversCall
		getSessionResultsCalls          []getSessionResultsCall
		getDriverSessionCalls           []getDriverSessionCall
		getLapDataCalls                 []getLapDataCall
		persistSessionDataCalls         []persistSessionDataCall
		emitCountCalls                  []emitCountCall
		pushCalls                       []pushCall
		broadcastCalls                  []broadcastCall
		updateDriverRacesIngestedToCall *updateDriverRacesIngestedToCall
		publishEventCall                *publishEventCall

		expectedErr string
	}{
		{
			name: "session exists but driver session does not - uses stored data to create driver session",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			releaseIngestionLockCall: &releaseIngestionLockCall{driverID: driverID},
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result: &store.Driver{
					DriverID:    driverID,
					MemberSince: memberSince,
				},
			},
			searchSeriesResultsCall: &searchSeriesResultsCall{
				finishRangeBegin: memberSince,
				finishRangeEnd:   rangeEnd,
				result: []iracing.SeriesResult{
					{SubsessionID: subsessionID},
				},
			},
			getSessionCalls: []getSessionCall{
				{
					subsessionID: subsessionID,
					result: &store.Session{
						SubsessionID:    subsessionID,
						TrackID:         123,
						SeriesID:        42,
						SeriesName:      "Test Series",
						LicenseCategory: "Road",
						StartTime:       sessionStartTime,
					},
				},
			},
			// No iRacing API calls - use stored data instead
			getSessionResultsCalls: []getSessionResultsCall{},
			getLapDataCalls:        []getLapDataCall{},
			// Check if driver session exists
			getDriverSessionCalls: []getDriverSessionCall{
				{
					driverID:  driverID,
					startTime: sessionStartTime,
					result:    nil, // doesn't exist yet for this driver
				},
			},
			// Query stored SessionDriver data to build DriverSession
			getSessionDriversCalls: []getSessionDriversCall{
				{
					subsessionID: subsessionID,
					result: []store.SessionDriver{
						{
							SubsessionID:          subsessionID,
							DriverID:              driverID,
							CarID:                 10,
							StartPosition:         5,
							StartPositionInClass:  5,
							FinishPosition:        3,
							FinishPositionInClass: 3,
							Incidents:             2,
							OldIRating:            1400,
							NewIRating:            1450,
							OldLicenseLevel:       17,
							NewLicenseLevel:       18,
							OldSubLevel:           381,
							NewSubLevel:           399,
						},
						{
							SubsessionID: subsessionID,
							DriverID:     99999, // another driver
							CarID:        11,
						},
					},
				},
			},
			// Should persist the new DriverSession
			persistSessionDataCalls: []persistSessionDataCall{
				{
					validate: func(t *testing.T, data store.SessionDataInsertion) {
						// Only DriverSession, no Session/SessionDriver/Laps
						assert.Empty(t, data.SessionEntries)
						assert.Empty(t, data.SessionDriverEntries)
						assert.Empty(t, data.SessionDriverLapEntries)

						require.Len(t, data.DriverSessionEntries, 1)
						ds := data.DriverSessionEntries[0]
						assert.Equal(t, driverID, ds.DriverID)
						assert.Equal(t, subsessionID, ds.SubsessionID)
						assert.Equal(t, int64(123), ds.TrackID)
						assert.Equal(t, int64(42), ds.SeriesID)
						assert.Equal(t, "Test Series", ds.SeriesName)
						assert.Equal(t, int64(10), ds.CarID)
						assert.Equal(t, 5, ds.StartPosition)
						assert.Equal(t, 3, ds.FinishPosition)
						assert.Equal(t, 2, ds.Incidents)
						assert.Equal(t, 1400, ds.OldIRating)
						assert.Equal(t, 1450, ds.NewIRating)
						assert.Equal(t, 17, ds.OldLicenseLevel)
						assert.Equal(t, 18, ds.NewLicenseLevel)
						assert.Equal(t, 381, ds.OldSubLevel)
						assert.Equal(t, 399, ds.NewSubLevel)
					},
				},
			},
			// Should broadcast to all driver's connections
			broadcastCalls: []broadcastCall{
				{
					driverID:   driverID,
					actionType: "raceIngested",
					payload:    RaceReadyMsg{RaceID: sessionStartTime.Unix()},
				},
				{
					driverID:   driverID,
					actionType: "ingestionChunkComplete",
					payload:    ChunkCompleteMsg{IngestedTo: rangeEnd},
				},
			},
			updateDriverRacesIngestedToCall: &updateDriverRacesIngestedToCall{
				driverID:        driverID,
				racesIngestedTo: rangeEnd,
			},
			publishEventCall: &publishEventCall{
				event: RaceIngestionRequest{
					DriverID:           driverID,
					IRacingAccessToken: "test-token",
					NotifyConnectionID: "conn-123",
				},
			},
		},
		{
			name: "driver session already exists - should not persist or notify",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			releaseIngestionLockCall: &releaseIngestionLockCall{driverID: driverID},
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result: &store.Driver{
					DriverID:    driverID,
					MemberSince: memberSince,
				},
			},
			searchSeriesResultsCall: &searchSeriesResultsCall{
				finishRangeBegin: memberSince,
				finishRangeEnd:   rangeEnd,
				result: []iracing.SeriesResult{
					{SubsessionID: subsessionID},
				},
			},
			getSessionCalls: []getSessionCall{
				{
					subsessionID: subsessionID,
					result: &store.Session{
						SubsessionID: subsessionID,
						TrackID:      123,
						StartTime:    sessionStartTime,
					},
				},
			},
			// Session exists, so no iRacing API calls
			getSessionResultsCalls: []getSessionResultsCall{},
			getLapDataCalls:        []getLapDataCall{},
			// Driver session already exists
			getDriverSessionCalls: []getDriverSessionCall{
				{
					driverID:  driverID,
					startTime: sessionStartTime,
					result: &store.DriverSession{
						DriverID:     driverID,
						SubsessionID: subsessionID,
						StartTime:    sessionStartTime,
					},
				},
			},
			// No persist calls - nothing new to save
			persistSessionDataCalls: []persistSessionDataCall{},
			// No push calls - nothing new to notify
			pushCalls: []pushCall{},
			// Should still broadcast chunk complete
			broadcastCalls: []broadcastCall{
				{
					driverID:   driverID,
					actionType: "ingestionChunkComplete",
					payload:    ChunkCompleteMsg{IngestedTo: rangeEnd},
				},
			},
			// Should still update ingested-to marker
			updateDriverRacesIngestedToCall: &updateDriverRacesIngestedToCall{
				driverID:        driverID,
				racesIngestedTo: rangeEnd,
			},
			publishEventCall: &publishEventCall{
				event: RaceIngestionRequest{
					DriverID:           driverID,
					IRacingAccessToken: "test-token",
					NotifyConnectionID: "conn-123",
				},
			},
		},
		{
			name: "happy path - new session and driver session - calls all APIs and persists",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			releaseIngestionLockCall: &releaseIngestionLockCall{driverID: driverID},
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result: &store.Driver{
					DriverID:    driverID,
					MemberSince: memberSince,
				},
			},
			searchSeriesResultsCall: &searchSeriesResultsCall{
				finishRangeBegin: memberSince,
				finishRangeEnd:   rangeEnd,
				result: []iracing.SeriesResult{
					{SubsessionID: subsessionID},
				},
			},
			getSessionCalls: []getSessionCall{
				{
					subsessionID: subsessionID,
					result:       nil, // session doesn't exist
				},
			},
			getSessionResultsCalls: []getSessionResultsCall{
				{
					subsessionID: subsessionID,
					result: &iracing.SessionResult{
						SubsessionID:    subsessionID,
						SeriesID:        42,
						SeriesName:      "Test Series",
						LicenseCategory: "Road",
						Track:           iracing.Track{TrackID: 123},
						StartTime:       sessionStartTime,
						CarClasses: []iracing.CarClass{
							{
								CarClassID:      1,
								StrengthOfField: 1500,
								NumEntries:      20,
								CarsInClass:     []iracing.CarInClass{{CarID: 10}},
							},
						},
						SessionResults: []iracing.SimSessionResult{
							{
								SimsessionNumber: 0, // main event
								Results: []iracing.DriverResult{
									{
										CustID:                  driverID,
										CarID:                   10,
										StartingPosition:        5,
										StartingPositionInClass: 5,
										FinishPosition:          3,
										FinishPositionInClass:   3,
										Incidents:               2,
										OldIRating:              1400,
										NewIRating:              1450,
										OldLicenseLevel:         17,
										NewLicenseLevel:         18,
										OldSubLevel:             381,
										NewSubLevel:             399,
									},
								},
							},
						},
					},
				},
			},
			getDriverSessionCalls: []getDriverSessionCall{
				{
					driverID:  driverID,
					startTime: sessionStartTime,
					result:    nil, // doesn't exist
				},
			},
			getLapDataCalls: []getLapDataCall{
				{
					subsessionID:     subsessionID,
					simsessionNumber: 0,
					result: &iracing.LapDataResponse{
						Laps: []iracing.Lap{
							{LapNumber: 1, LapTime: 60000, Flags: 0},
							{LapNumber: 2, LapTime: 59500, Flags: 0},
						},
					},
				},
			},
			persistSessionDataCalls: []persistSessionDataCall{
				{
					validate: func(t *testing.T, data store.SessionDataInsertion) {
						require.Len(t, data.SessionEntries, 1)
						session := data.SessionEntries[0]
						assert.Equal(t, subsessionID, session.SubsessionID)
						assert.Equal(t, int64(123), session.TrackID)
						assert.Equal(t, int64(42), session.SeriesID)
						assert.Equal(t, "Test Series", session.SeriesName)
						assert.Equal(t, "Road", session.LicenseCategory)

						require.Len(t, data.SessionDriverEntries, 1)
						sd := data.SessionDriverEntries[0]
						assert.Equal(t, driverID, sd.DriverID)
						assert.Equal(t, int64(10), sd.CarID)
						assert.Equal(t, 17, sd.OldLicenseLevel)
						assert.Equal(t, 18, sd.NewLicenseLevel)
						assert.Equal(t, 381, sd.OldSubLevel)
						assert.Equal(t, 399, sd.NewSubLevel)

						require.Len(t, data.DriverSessionEntries, 1)
						ds := data.DriverSessionEntries[0]
						assert.Equal(t, driverID, ds.DriverID)
						assert.Equal(t, 5, ds.StartPosition)
						assert.Equal(t, 3, ds.FinishPosition)
						assert.Equal(t, int64(42), ds.SeriesID)
						assert.Equal(t, "Test Series", ds.SeriesName)
						assert.Equal(t, 17, ds.OldLicenseLevel)
						assert.Equal(t, 18, ds.NewLicenseLevel)
						assert.Equal(t, 381, ds.OldSubLevel)
						assert.Equal(t, 399, ds.NewSubLevel)

						require.Len(t, data.SessionDriverLapEntries, 2)
					},
				},
			},
			emitCountCalls: []emitCountCall{
				{name: metrics.SessionsIngested, count: 1},
				{name: metrics.LapsIngested, count: 2},
			},
			broadcastCalls: []broadcastCall{
				{
					driverID:   driverID,
					actionType: "raceIngested",
					payload:    RaceReadyMsg{RaceID: sessionStartTime.Unix()},
				},
				{
					driverID:   driverID,
					actionType: "ingestionChunkComplete",
					payload:    ChunkCompleteMsg{IngestedTo: rangeEnd},
				},
			},
			updateDriverRacesIngestedToCall: &updateDriverRacesIngestedToCall{
				driverID:        driverID,
				racesIngestedTo: rangeEnd,
			},
			publishEventCall: &publishEventCall{
				event: RaceIngestionRequest{
					DriverID:           driverID,
					IRacingAccessToken: "test-token",
					NotifyConnectionID: "conn-123",
				},
			},
		},
		{
			name: "stale credentials on SearchSeriesResults - notifies and returns without error",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "stale-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			releaseIngestionLockCall: &releaseIngestionLockCall{driverID: driverID}, // Release so client can retry immediately
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result: &store.Driver{
					DriverID:    driverID,
					MemberSince: memberSince,
				},
			},
			searchSeriesResultsCall: &searchSeriesResultsCall{
				finishRangeBegin: memberSince,
				finishRangeEnd:   rangeEnd,
				result:           nil,
				err:              iracing.ErrUpstreamUnauthorized,
			},
			pushCalls: []pushCall{
				{
					connectionID: "conn-123",
					actionType:   "ingestionFailedStaleCredentials",
					result:       true,
				},
			},
			// No UpdateDriverRacesIngestedTo call when stale credentials
		},
		{
			name: "driver not found - returns error",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			releaseIngestionLockCall: &releaseIngestionLockCall{driverID: driverID}, // Release on error
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result:   nil, // driver not found
			},
			expectedErr: "driver 12345 not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := zerolog.New(zerolog.NewTestWriter(t)).WithContext(context.Background())

			mockStore := NewMockStore(t)
			mockIRacing := NewMockIRacingClient(t)
			mockPusher := NewMockPusher(t)
			mockEventDispatcher := NewMockEventDispatcher(t)
			mockMetricsClient := NewMockMetricsClient(t)

			// Setup AcquireIngestionLock
			mockStore.EXPECT().AcquireIngestionLock(mock.Anything, tc.acquireIngestionLockCall.driverID, lockDuration).
				Return(tc.acquireIngestionLockCall.acquired, tc.acquireIngestionLockCall.err)

			// Setup ReleaseIngestionLock
			if tc.releaseIngestionLockCall != nil {
				mockStore.EXPECT().ReleaseIngestionLock(mock.Anything, tc.releaseIngestionLockCall.driverID).
					Return(tc.releaseIngestionLockCall.err)
			}

			// Setup GetDriver
			if tc.getDriverCall != nil {
				mockStore.EXPECT().GetDriver(mock.Anything, tc.getDriverCall.driverID).
					Return(tc.getDriverCall.result, tc.getDriverCall.err)
			}

			// Setup SearchSeriesResults
			if tc.searchSeriesResultsCall != nil {
				mockIRacing.EXPECT().SearchSeriesResults(
					mock.Anything,
					tc.request.IRacingAccessToken,
					tc.searchSeriesResultsCall.finishRangeBegin,
					tc.searchSeriesResultsCall.finishRangeEnd,
					mock.Anything,
				).Return(tc.searchSeriesResultsCall.result, tc.searchSeriesResultsCall.err)
			}

			// Setup GetSession calls
			for _, call := range tc.getSessionCalls {
				mockStore.EXPECT().GetSession(mock.Anything, call.subsessionID).
					Return(call.result, call.err)
			}

			// Setup GetSessionDrivers calls
			for _, call := range tc.getSessionDriversCalls {
				mockStore.EXPECT().GetSessionDrivers(mock.Anything, call.subsessionID).
					Return(call.result, call.err)
			}

			// Setup GetSessionResults calls
			for _, call := range tc.getSessionResultsCalls {
				mockIRacing.EXPECT().GetSessionResults(
					mock.Anything,
					tc.request.IRacingAccessToken,
					call.subsessionID,
					mock.Anything,
				).Return(call.result, call.err)
			}

			// Setup GetDriverSession calls
			for _, call := range tc.getDriverSessionCalls {
				mockStore.EXPECT().GetDriverSession(mock.Anything, call.driverID, call.startTime).
					Return(call.result, call.err)
			}

			// Setup GetLapData calls
			for _, call := range tc.getLapDataCalls {
				mockIRacing.EXPECT().GetLapData(
					mock.Anything,
					tc.request.IRacingAccessToken,
					call.subsessionID,
					call.simsessionNumber,
					mock.Anything,
				).Return(call.result, call.err)
			}

			// Setup PersistSessionData calls
			for _, call := range tc.persistSessionDataCalls {
				mockStore.EXPECT().PersistSessionData(mock.Anything, mock.MatchedBy(func(data store.SessionDataInsertion) bool {
					if call.validate != nil {
						call.validate(t, data)
					}
					return true
				})).Return(call.err)
			}

			// Setup Push calls
			for _, call := range tc.pushCalls {
				mockPusher.EXPECT().Push(mock.Anything, call.connectionID, call.actionType, mock.Anything).
					Return(call.result, call.err)
			}

			// Setup Broadcast calls
			for _, call := range tc.broadcastCalls {
				mockPusher.EXPECT().Broadcast(mock.Anything, call.driverID, call.actionType, call.payload).
					Return(call.err)
			}

			// Setup UpdateDriverRacesIngestedTo
			if tc.updateDriverRacesIngestedToCall != nil {
				mockStore.EXPECT().UpdateDriverRacesIngestedTo(
					mock.Anything,
					tc.updateDriverRacesIngestedToCall.driverID,
					tc.updateDriverRacesIngestedToCall.racesIngestedTo,
				).Return(tc.updateDriverRacesIngestedToCall.err)
			}

			// Setup PublishEvent
			if tc.publishEventCall != nil {
				mockEventDispatcher.EXPECT().PublishEvent(mock.Anything, tc.publishEventCall.event).
					Return(tc.publishEventCall.err)
			}

			// Setup EmitCount calls
			for _, call := range tc.emitCountCalls {
				mockMetricsClient.EXPECT().EmitCount(mock.Anything, call.name, call.count).
					Return(call.err)
			}

			processor := NewRaceProcessor(mockStore, mockIRacing, mockPusher, mockEventDispatcher, mockMetricsClient, lockDuration)
			processor.now = func() time.Time { return now }

			err := processor.IngestRaces(ctx, tc.request)

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
