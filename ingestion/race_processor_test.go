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

type saveDriverSessionsCall struct {
	validate func(t *testing.T, sessions []store.DriverSession)
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

	// For continuation scenario (driver with RacesIngestedTo set)
	racesIngestedTo := time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)
	continuationRangeBegin := racesIngestedTo.Add(-time.Hour * 4) // 4-hour buffer
	continuationRangeEnd := now                                   // capped by now since rangeBegin + 10 days > now

	testCases := []struct {
		name string

		request RaceIngestionRequest

		acquireIngestionLockCall        acquireIngestionLockCall
		releaseIngestionLockCall        *releaseIngestionLockCall
		getDriverCall                   *getDriverCall
		searchSeriesResultsCall         *searchSeriesResultsCall
		getSessionResultsCalls          []getSessionResultsCall
		getDriverSessionCalls           []getDriverSessionCall
		saveDriverSessionsCalls         []saveDriverSessionsCall
		emitCountCalls                  []emitCountCall
		pushCalls                       []pushCall
		broadcastCalls                  []broadcastCall
		updateDriverRacesIngestedToCall *updateDriverRacesIngestedToCall
		publishEventCall                *publishEventCall

		expectedErr string
	}{
		{
			name: "happy path - new driver session",
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
						SessionResults: []iracing.SimSessionResult{
							{
								SimsessionNumber: 0, // main event
								Results: []iracing.DriverResult{
									{
										CustID:                  driverID,
										DisplayName:             "Test Driver",
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
										ReasonOut:               "Running",
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
			saveDriverSessionsCalls: []saveDriverSessionsCall{
				{
					validate: func(t *testing.T, sessions []store.DriverSession) {
						require.Len(t, sessions, 1)
						ds := sessions[0]
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
						assert.Equal(t, "Running", ds.ReasonOut)
					},
				},
			},
			emitCountCalls: []emitCountCall{
				{name: metrics.DriverSessionsIngested, count: 1},
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
			name: "driver session already exists - skips save and notify",
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
			getSessionResultsCalls: []getSessionResultsCall{
				{
					subsessionID: subsessionID,
					result: &iracing.SessionResult{
						SubsessionID: subsessionID,
						StartTime:    sessionStartTime,
						SessionResults: []iracing.SimSessionResult{
							{SimsessionNumber: 0, Results: []iracing.DriverResult{{CustID: driverID}}},
						},
					},
				},
			},
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
			// No save calls - already exists
			saveDriverSessionsCalls: []saveDriverSessionsCall{},
			// No raceIngested broadcast - already exists
			broadcastCalls: []broadcastCall{
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
		},
		{
			name: "driver not found - returns error",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			releaseIngestionLockCall: &releaseIngestionLockCall{driverID: driverID},
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result:   nil,
			},
			expectedErr: "driver 12345 not found",
		},
		{
			name: "team event - skipped without processing",
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
					{SubsessionID: subsessionID, DriverChanges: true}, // team event
				},
			},
			// No API calls or saves - team event is skipped
			getSessionResultsCalls:  []getSessionResultsCall{},
			saveDriverSessionsCalls: []saveDriverSessionsCall{},
			broadcastCalls: []broadcastCall{
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
			name: "ingestion lock not acquired - skips processing",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: false},
			// No other calls - lock not acquired means skip
		},
		{
			name: "driver not found in session results - logs warning and continues",
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
			getSessionResultsCalls: []getSessionResultsCall{
				{
					subsessionID: subsessionID,
					result: &iracing.SessionResult{
						SubsessionID: subsessionID,
						StartTime:    sessionStartTime,
						SessionResults: []iracing.SimSessionResult{
							{
								SimsessionNumber: 0,
								Results: []iracing.DriverResult{
									{CustID: 99999}, // different driver
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
					result:    nil,
				},
			},
			// No save - driver not in results
			saveDriverSessionsCalls: []saveDriverSessionsCall{},
			broadcastCalls: []broadcastCall{
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
			name: "continuation ingestion - applies 4-hour buffer to search range",
			request: RaceIngestionRequest{
				DriverID:           driverID,
				IRacingAccessToken: "test-token",
				NotifyConnectionID: "conn-123",
			},
			acquireIngestionLockCall: acquireIngestionLockCall{driverID: driverID, acquired: true},
			// No releaseIngestionLockCall - willBeUpToDate=true so lock expires naturally
			getDriverCall: &getDriverCall{
				driverID: driverID,
				result: &store.Driver{
					DriverID:        driverID,
					MemberSince:     memberSince,
					RacesIngestedTo: &racesIngestedTo, // continuing from previous ingestion
				},
			},
			searchSeriesResultsCall: &searchSeriesResultsCall{
				finishRangeBegin: continuationRangeBegin, // RacesIngestedTo - 4 hours
				finishRangeEnd:   continuationRangeEnd,   // capped by now
				result:           []iracing.SeriesResult{},
			},
			// No races found, so no session calls
			getSessionResultsCalls:  []getSessionResultsCall{},
			saveDriverSessionsCalls: []saveDriverSessionsCall{},
			broadcastCalls: []broadcastCall{
				{
					driverID:   driverID,
					actionType: "ingestionChunkComplete",
					payload:    ChunkCompleteMsg{IngestedTo: continuationRangeEnd},
				},
			},
			updateDriverRacesIngestedToCall: &updateDriverRacesIngestedToCall{
				driverID:        driverID,
				racesIngestedTo: continuationRangeEnd,
			},
			// No publishEventCall - willBeUpToDate=true since rangeEnd was capped by now
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

			// Setup SaveDriverSessions calls
			for _, call := range tc.saveDriverSessionsCalls {
				mockStore.EXPECT().SaveDriverSessions(mock.Anything, mock.MatchedBy(func(sessions []store.DriverSession) bool {
					if call.validate != nil {
						call.validate(t, sessions)
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