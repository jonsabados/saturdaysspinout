package journal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jonsabados/saturdaysspinout/metrics"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateTags(t *testing.T) {
	testCases := []struct {
		name           string
		tags           []string
		expectedErrors int
		checkError     func(t *testing.T, errors []FieldValidation) // optional detailed check
	}{
		{
			name:           "empty tags",
			tags:           []string{},
			expectedErrors: 0,
		},
		{
			name:           "nil tags",
			tags:           nil,
			expectedErrors: 0,
		},
		{
			name:           "free-form tags without prefix",
			tags:           []string{"podium", "clean-race", "personal-best"},
			expectedErrors: 0,
		},
		{
			name:           "valid sentiment:good",
			tags:           []string{"sentiment:good"},
			expectedErrors: 0,
		},
		{
			name:           "valid sentiment:neutral",
			tags:           []string{"sentiment:neutral"},
			expectedErrors: 0,
		},
		{
			name:           "valid sentiment:bad",
			tags:           []string{"sentiment:bad"},
			expectedErrors: 0,
		},
		{
			name:           "invalid sentiment tag",
			tags:           []string{"sentiment:horrible"},
			expectedErrors: 1,
			checkError: func(t *testing.T, errors []FieldValidation) {
				assert.Equal(t, "tags", errors[0].Field)
				assert.Equal(t, "invalid_tag_value", errors[0].Code)
				assert.Equal(t, "sentiment", errors[0].Params["prefix"])
				assert.Equal(t, "horrible", errors[0].Params["value"])
				assert.Equal(t, "good,neutral,bad", errors[0].Params["allowed"])
			},
		},
		{
			name:           "mixed valid and invalid",
			tags:           []string{"podium", "sentiment:good", "sentiment:invalid", "clean-race"},
			expectedErrors: 1,
			checkError: func(t *testing.T, errors []FieldValidation) {
				assert.Equal(t, "invalid", errors[0].Params["value"])
			},
		},
		{
			name:           "unknown prefix allowed",
			tags:           []string{"custom:anything", "track:laguna-seca"},
			expectedErrors: 0,
		},
		{
			name:           "multiple invalid tags",
			tags:           []string{"sentiment:awful", "sentiment:terrible"},
			expectedErrors: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateTags(tc.tags)
			assert.Len(t, errs, tc.expectedErrors)
			if tc.checkError != nil && len(errs) > 0 {
				tc.checkError(t, errs)
			}
		})
	}
}

func TestService_ValidateRaceExists(t *testing.T) {
	ctx := context.Background()
	driverID := int64(12345)
	raceID := int64(1700000000)
	startTime := store.TimeFromDriverRaceID(raceID)

	testCases := []struct {
		name          string
		setupMock     func(*MockStore)
		expectedExist bool
		expectedErr   bool
	}{
		{
			name: "race exists",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(&store.DriverSession{DriverID: driverID, StartTime: startTime}, nil)
			},
			expectedExist: true,
			expectedErr:   false,
		},
		{
			name: "race does not exist",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(nil, nil)
			},
			expectedExist: false,
			expectedErr:   false,
		},
		{
			name: "store error",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(nil, errors.New("database error"))
			},
			expectedExist: false,
			expectedErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockMetrics := NewMockMetricsEmitter(t)
			tc.setupMock(mockStore)

			svc := NewService(mockStore, mockMetrics)
			exists, err := svc.ValidateRaceExists(ctx, driverID, raceID)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedExist, exists)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	ctx := context.Background()
	driverID := int64(12345)
	raceID := int64(1700000000)
	startTime := store.TimeFromDriverRaceID(raceID)
	createdAt := time.Unix(1000, 0)
	updatedAt := time.Unix(2000, 0)

	testCases := []struct {
		name        string
		setupMock   func(*MockStore)
		expected    *Entry
		expectedErr bool
	}{
		{
			name: "entry and race exist",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(&store.RaceJournalEntry{
						DriverID:  driverID,
						RaceID:    raceID,
						Notes:     "Great race!",
						Tags:      []string{"sentiment:good"},
						CreatedAt: createdAt,
						UpdatedAt: updatedAt,
					}, nil)
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(&store.DriverSession{
						DriverID:       driverID,
						StartTime:      startTime,
						FinishPosition: 2,
					}, nil)
			},
			expected: &Entry{
				RaceID:    raceID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Notes:     "Great race!",
				Tags:      []string{"sentiment:good"},
				Race: &store.DriverSession{
					DriverID:       driverID,
					StartTime:      startTime,
					FinishPosition: 2,
				},
			},
			expectedErr: false,
		},
		{
			name: "entry exists but race missing",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(&store.RaceJournalEntry{
						DriverID:  driverID,
						RaceID:    raceID,
						Notes:     "Orphaned entry",
						CreatedAt: createdAt,
						UpdatedAt: updatedAt,
					}, nil)
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(nil, nil)
			},
			expected: &Entry{
				RaceID:    raceID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Notes:     "Orphaned entry",
				Tags:      []string{},
				Race:      nil,
			},
			expectedErr: false,
		},
		{
			name: "entry does not exist",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(nil, nil)
			},
			expected:    nil,
			expectedErr: false,
		},
		{
			name: "GetJournalEntry error",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(nil, errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "GetDriverSession error",
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(&store.RaceJournalEntry{
						DriverID:  driverID,
						RaceID:    raceID,
						CreatedAt: createdAt,
						UpdatedAt: updatedAt,
					}, nil)
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(nil, errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockMetrics := NewMockMetricsEmitter(t)
			tc.setupMock(mockStore)

			svc := NewService(mockStore, mockMetrics)
			entry, err := svc.Get(ctx, driverID, raceID)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, entry)
			}
		})
	}
}

func TestService_Save(t *testing.T) {
	ctx := context.Background()
	driverID := int64(12345)
	raceID := int64(1700000000)
	startTime := store.TimeFromDriverRaceID(raceID)
	createdAt := time.Unix(1000, 0)
	updatedAt := time.Unix(2000, 0)

	testCases := []struct {
		name            string
		input           SaveInput
		setupMock       func(*MockStore, *MockMetricsEmitter)
		expected        *Entry
		expectedErr     bool
		expectMetricErr bool
	}{
		{
			name: "success",
			input: SaveInput{
				DriverID: driverID,
				RaceID:   raceID,
				Notes:    "Great race!",
				Tags:     []string{"sentiment:good"},
			},
			setupMock: func(m *MockStore, me *MockMetricsEmitter) {
				m.EXPECT().SaveJournalEntry(mock.Anything, store.RaceJournalEntry{
					DriverID: driverID,
					RaceID:   raceID,
					Notes:    "Great race!",
					Tags:     []string{"sentiment:good"},
				}).Return(nil)
				me.EXPECT().EmitCount(mock.Anything, metrics.JournalEntriesCreated, 1).Return(nil)
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(&store.RaceJournalEntry{
						DriverID:  driverID,
						RaceID:    raceID,
						Notes:     "Great race!",
						Tags:      []string{"sentiment:good"},
						CreatedAt: createdAt,
						UpdatedAt: updatedAt,
					}, nil)
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(&store.DriverSession{
						DriverID:       driverID,
						StartTime:      startTime,
						FinishPosition: 2,
					}, nil)
			},
			expected: &Entry{
				RaceID:    raceID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Notes:     "Great race!",
				Tags:      []string{"sentiment:good"},
				Race: &store.DriverSession{
					DriverID:       driverID,
					StartTime:      startTime,
					FinishPosition: 2,
				},
			},
			expectedErr: false,
		},
		{
			name: "success with metric error (logs but continues)",
			input: SaveInput{
				DriverID: driverID,
				RaceID:   raceID,
				Notes:    "Great race!",
				Tags:     []string{"sentiment:good"},
			},
			setupMock: func(m *MockStore, me *MockMetricsEmitter) {
				m.EXPECT().SaveJournalEntry(mock.Anything, store.RaceJournalEntry{
					DriverID: driverID,
					RaceID:   raceID,
					Notes:    "Great race!",
					Tags:     []string{"sentiment:good"},
				}).Return(nil)
				me.EXPECT().EmitCount(mock.Anything, metrics.JournalEntriesCreated, 1).Return(errors.New("cloudwatch error"))
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(&store.RaceJournalEntry{
						DriverID:  driverID,
						RaceID:    raceID,
						Notes:     "Great race!",
						Tags:      []string{"sentiment:good"},
						CreatedAt: createdAt,
						UpdatedAt: updatedAt,
					}, nil)
				m.EXPECT().GetDriverSession(mock.Anything, driverID, startTime).
					Return(&store.DriverSession{
						DriverID:       driverID,
						StartTime:      startTime,
						FinishPosition: 2,
					}, nil)
			},
			expected: &Entry{
				RaceID:    raceID,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Notes:     "Great race!",
				Tags:      []string{"sentiment:good"},
				Race: &store.DriverSession{
					DriverID:       driverID,
					StartTime:      startTime,
					FinishPosition: 2,
				},
			},
			expectedErr: false,
		},
		{
			name: "save error",
			input: SaveInput{
				DriverID: driverID,
				RaceID:   raceID,
				Notes:    "Great race!",
			},
			setupMock: func(m *MockStore, me *MockMetricsEmitter) {
				m.EXPECT().SaveJournalEntry(mock.Anything, mock.Anything).
					Return(errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "get after save error",
			input: SaveInput{
				DriverID: driverID,
				RaceID:   raceID,
				Notes:    "Great race!",
			},
			setupMock: func(m *MockStore, me *MockMetricsEmitter) {
				m.EXPECT().SaveJournalEntry(mock.Anything, mock.Anything).Return(nil)
				me.EXPECT().EmitCount(mock.Anything, metrics.JournalEntriesCreated, 1).Return(nil)
				m.EXPECT().GetJournalEntry(mock.Anything, driverID, raceID).
					Return(nil, errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockMetrics := NewMockMetricsEmitter(t)
			tc.setupMock(mockStore, mockMetrics)

			svc := NewService(mockStore, mockMetrics)
			entry, err := svc.Save(ctx, tc.input)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, entry)
			}
		})
	}
}

func TestService_List(t *testing.T) {
	ctx := context.Background()
	driverID := int64(12345)
	from := time.Unix(1000, 0)
	to := time.Unix(5000, 0)

	raceID1 := int64(2000)
	raceID2 := int64(3000)
	startTime1 := store.TimeFromDriverRaceID(raceID1)
	startTime2 := store.TimeFromDriverRaceID(raceID2)

	testCases := []struct {
		name        string
		input       ListInput
		setupMock   func(*MockStore)
		expected    []Entry
		expectedErr bool
	}{
		{
			name:  "empty results",
			input: ListInput{DriverID: driverID, From: from, To: to},
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntries(mock.Anything, driverID, from, to).
					Return([]store.RaceJournalEntry{}, nil)
			},
			expected:    []Entry{},
			expectedErr: false,
		},
		{
			name:  "entries with sessions",
			input: ListInput{DriverID: driverID, From: from, To: to},
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntries(mock.Anything, driverID, from, to).
					Return([]store.RaceJournalEntry{
						{DriverID: driverID, RaceID: raceID1, Notes: "Race 1", CreatedAt: from, UpdatedAt: from},
						{DriverID: driverID, RaceID: raceID2, Notes: "Race 2", CreatedAt: to, UpdatedAt: to},
					}, nil)
				m.EXPECT().GetDriverSessions(mock.Anything, driverID, []time.Time{startTime1, startTime2}).
					Return([]store.DriverSession{
						{DriverID: driverID, StartTime: startTime1, FinishPosition: 1},
						{DriverID: driverID, StartTime: startTime2, FinishPosition: 2},
					}, nil)
			},
			expected: []Entry{
				{RaceID: raceID1, Notes: "Race 1", Tags: []string{}, CreatedAt: from, UpdatedAt: from, Race: &store.DriverSession{DriverID: driverID, StartTime: startTime1, FinishPosition: 1}},
				{RaceID: raceID2, Notes: "Race 2", Tags: []string{}, CreatedAt: to, UpdatedAt: to, Race: &store.DriverSession{DriverID: driverID, StartTime: startTime2, FinishPosition: 2}},
			},
			expectedErr: false,
		},
		{
			name:  "entries with some missing sessions",
			input: ListInput{DriverID: driverID, From: from, To: to},
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntries(mock.Anything, driverID, from, to).
					Return([]store.RaceJournalEntry{
						{DriverID: driverID, RaceID: raceID1, Notes: "Race 1", CreatedAt: from, UpdatedAt: from},
						{DriverID: driverID, RaceID: raceID2, Notes: "Race 2", CreatedAt: to, UpdatedAt: to},
					}, nil)
				m.EXPECT().GetDriverSessions(mock.Anything, driverID, []time.Time{startTime1, startTime2}).
					Return([]store.DriverSession{
						{DriverID: driverID, StartTime: startTime1, FinishPosition: 1},
					}, nil) // Only one session returned
			},
			expected: []Entry{
				{RaceID: raceID1, Notes: "Race 1", Tags: []string{}, CreatedAt: from, UpdatedAt: from, Race: &store.DriverSession{DriverID: driverID, StartTime: startTime1, FinishPosition: 1}},
				{RaceID: raceID2, Notes: "Race 2", Tags: []string{}, CreatedAt: to, UpdatedAt: to, Race: nil},
			},
			expectedErr: false,
		},
		{
			name:  "GetJournalEntries error",
			input: ListInput{DriverID: driverID, From: from, To: to},
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntries(mock.Anything, driverID, from, to).
					Return(nil, errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name:  "GetDriverSessions error",
			input: ListInput{DriverID: driverID, From: from, To: to},
			setupMock: func(m *MockStore) {
				m.EXPECT().GetJournalEntries(mock.Anything, driverID, from, to).
					Return([]store.RaceJournalEntry{
						{DriverID: driverID, RaceID: raceID1, Notes: "Race 1"},
					}, nil)
				m.EXPECT().GetDriverSessions(mock.Anything, driverID, []time.Time{startTime1}).
					Return(nil, errors.New("database error"))
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockMetrics := NewMockMetricsEmitter(t)
			tc.setupMock(mockStore)

			svc := NewService(mockStore, mockMetrics)
			entries, err := svc.List(ctx, tc.input)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, entries)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	ctx := context.Background()
	driverID := int64(12345)
	raceID := int64(1700000000)

	testCases := []struct {
		name        string
		setupMock   func(*MockStore)
		expectedErr bool
	}{
		{
			name: "success",
			setupMock: func(m *MockStore) {
				m.EXPECT().DeleteJournalEntry(mock.Anything, driverID, raceID).Return(nil)
			},
			expectedErr: false,
		},
		{
			name: "store error",
			setupMock: func(m *MockStore) {
				m.EXPECT().DeleteJournalEntry(mock.Anything, driverID, raceID).
					Return(errors.New("database error"))
			},
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := NewMockStore(t)
			mockMetrics := NewMockMetricsEmitter(t)
			tc.setupMock(mockStore)

			svc := NewService(mockStore, mockMetrics)
			err := svc.Delete(ctx, driverID, raceID)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
