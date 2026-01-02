package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const localDynamoEndpoint = "http://localhost:8000"

func TestGetGlobalCounters_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, &GlobalCounters{}, counters)
}

func TestGetDriver_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	got, err := s.GetDriver(ctx, 999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestInsertDriver_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}

	err := s.InsertDriver(ctx, driver)
	require.NoError(t, err)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, &driver, got)
}

func TestInsertDriver_DuplicateReturnsError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}

	err := s.InsertDriver(ctx, driver)
	require.NoError(t, err)

	err = s.InsertDriver(ctx, driver)
	assert.ErrorIs(t, err, ErrEntityAlreadyExists)
}

func TestInsertDriver_IncrementsGlobalCounter(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, s.InsertDriver(ctx, Driver{
		DriverID:    1,
		DriverName:  "Driver 1",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}))
	require.NoError(t, s.InsertDriver(ctx, Driver{
		DriverID:    2,
		DriverName:  "Driver 2",
		MemberSince: time.Unix(1500, 0),
		FirstLogin:  time.Unix(2000, 0),
		LastLogin:   time.Unix(2000, 0),
		LoginCount:  1,
	}))

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counters.Drivers)
}

func TestInsertDriver_WithEntitlements(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:     12345,
		DriverName:   "Jon Sabados",
		MemberSince:  time.Unix(500, 0),
		FirstLogin:   time.Unix(1000, 0),
		LastLogin:    time.Unix(1000, 0),
		LoginCount:   1,
		Entitlements: []string{"developer", "beta-tester"},
	}

	err := s.InsertDriver(ctx, driver)
	require.NoError(t, err)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, &driver, got)
}

func TestInsertDriver_WithoutEntitlements(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}

	err := s.InsertDriver(ctx, driver)
	require.NoError(t, err)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Nil(t, got.Entitlements)
}

func TestRecordLogin_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	err := s.RecordLogin(ctx, 12345, time.Unix(2000, 0))
	require.NoError(t, err)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, time.Unix(1000, 0), got.FirstLogin)
	assert.Equal(t, time.Unix(2000, 0), got.LastLogin)
	assert.Equal(t, int64(2), got.LoginCount)
}

func TestRecordLogin_MultipleLogins(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	require.NoError(t, s.RecordLogin(ctx, 12345, time.Unix(2000, 0)))
	require.NoError(t, s.RecordLogin(ctx, 12345, time.Unix(3000, 0)))
	require.NoError(t, s.RecordLogin(ctx, 12345, time.Unix(4000, 0)))

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, time.Unix(1000, 0), got.FirstLogin)
	assert.Equal(t, time.Unix(4000, 0), got.LastLogin)
	assert.Equal(t, int64(4), got.LoginCount)
}

func TestRecordLogin_DriverNotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	err := s.RecordLogin(ctx, 999, time.Unix(1000, 0))
	assert.Error(t, err)
}

func TestSaveConnection_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	conn := WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "abc123",
	}

	err := s.SaveConnection(ctx, conn)
	require.NoError(t, err)

	// Verify by reading it back
	got, err := s.GetConnection(ctx, 12345, "abc123")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(12345), got.DriverID)
	assert.Equal(t, "abc123", got.ConnectionID)
	assert.False(t, got.ConnectedAt.IsZero())
}

func TestSaveConnection_OverwritesExisting(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	conn := WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "abc123",
	}

	err := s.SaveConnection(ctx, conn)
	require.NoError(t, err)

	// Save again with same IDs - should overwrite without error
	err = s.SaveConnection(ctx, conn)
	require.NoError(t, err)

	// Should still only have one connection
	connections, err := s.GetConnectionsByDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Len(t, connections, 1)
}

func TestGetConnection_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	got, err := s.GetConnection(ctx, 999, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetConnection_WrongDriver(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	conn := WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "abc123",
	}
	require.NoError(t, s.SaveConnection(ctx, conn))

	// Try to get with wrong driver ID
	got, err := s.GetConnection(ctx, 99999, "abc123")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeleteConnection_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	conn := WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "abc123",
	}
	require.NoError(t, s.SaveConnection(ctx, conn))

	err := s.DeleteConnection(ctx, 12345, "abc123")
	require.NoError(t, err)

	// Verify it's gone
	got, err := s.GetConnection(ctx, 12345, "abc123")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeleteConnection_NotFoundNoError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Deleting non-existent connection should not error
	err := s.DeleteConnection(ctx, 999, "nonexistent")
	require.NoError(t, err)
}

func TestGetConnectionsByDriver_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	// Create multiple connections for same driver
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn1",
	}))
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn2",
	}))
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn3",
	}))

	connections, err := s.GetConnectionsByDriver(ctx, 12345)
	require.NoError(t, err)
	assert.ElementsMatch(t, []WebSocketConnection{
		{DriverID: 12345, ConnectionID: "conn1", ConnectedAt: fixedTime},
		{DriverID: 12345, ConnectionID: "conn2", ConnectedAt: fixedTime},
		{DriverID: 12345, ConnectionID: "conn3", ConnectedAt: fixedTime},
	}, connections)
}

func TestGetConnectionsByDriver_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	connections, err := s.GetConnectionsByDriver(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, connections)
}

func TestGetConnectionsByDriver_IsolatedByDriver(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Create connections for different drivers
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     111,
		ConnectionID: "conn-driver1",
	}))
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     222,
		ConnectionID: "conn-driver2",
	}))

	// Query for driver 111 only
	connections, err := s.GetConnectionsByDriver(ctx, 111)
	require.NoError(t, err)
	require.Len(t, connections, 1)
	assert.Equal(t, "conn-driver1", connections[0].ConnectionID)
}

func TestGetDriverIDByConnection_RecordExists(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Create connections for different drivers
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     111,
		ConnectionID: "conn-driver1",
	}))
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     222,
		ConnectionID: "conn-driver2",
	}))

	driver, err := s.GetDriverIDByConnection(ctx, "conn-driver1")
	assert.NoError(t, err)
	assert.Equal(t, aws.Int64(int64(111)), driver)
}

func TestGetDriverIDByConnection_RecordNotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Create connections for different drivers
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     111,
		ConnectionID: "conn-driver1",
	}))
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     222,
		ConnectionID: "conn-driver2",
	}))

	driver, err := s.GetDriverIDByConnection(ctx, "conn-driverBLAH")
	assert.NoError(t, err)
	assert.Nil(t, driver)
}

func TestUpdateDriverRacesIngestedTo_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	ingestedTo := time.Unix(5000, 0)
	err := s.UpdateDriverRacesIngestedTo(ctx, 12345, ingestedTo)
	require.NoError(t, err)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got.RacesIngestedTo)
	assert.Equal(t, ingestedTo, *got.RacesIngestedTo)
	// Verify other fields unchanged
	assert.Equal(t, driver.DriverName, got.DriverName)
	assert.Equal(t, driver.FirstLogin, got.FirstLogin)
	assert.Equal(t, driver.LoginCount, got.LoginCount)
}

func TestUpdateDriverRacesIngestedTo_DriverNotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	err := s.UpdateDriverRacesIngestedTo(ctx, 999, time.Unix(1000, 0))
	assert.Error(t, err)
}

func TestUpdateDriverRacesIngestedTo_UpdatesExistingValue(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	ingestedTo := time.Unix(3000, 0)
	driver := Driver{
		DriverID:        12345,
		DriverName:      "Jon Sabados",
		MemberSince:     time.Unix(500, 0),
		FirstLogin:      time.Unix(1000, 0),
		LastLogin:       time.Unix(1000, 0),
		LoginCount:      1,
		RacesIngestedTo: &ingestedTo,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	newIngestedTo := time.Unix(6000, 0)
	err := s.UpdateDriverRacesIngestedTo(ctx, 12345, newIngestedTo)
	require.NoError(t, err)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got.RacesIngestedTo)
	assert.Equal(t, newIngestedTo, *got.RacesIngestedTo)
}

func TestAcquireIngestionLock_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	acquired, err := s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Verify lock is visible via GetDriver (need a driver record first)
	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got.IngestionBlockedUntil)
	assert.Equal(t, fixedTime.Add(15*time.Minute), *got.IngestionBlockedUntil)
}

func TestAcquireIngestionLock_AlreadyHeld(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	// First acquisition should succeed
	acquired, err := s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Second acquisition should fail (lock still held)
	acquired, err = s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.False(t, acquired)
}

func TestAcquireIngestionLock_ExpiredLockCanBeAcquired(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	currentTime := time.Unix(1000, 0)
	s.now = func() time.Time { return currentTime }

	// First acquisition
	acquired, err := s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Advance time past lock expiration
	currentTime = time.Unix(1000+16*60, 0) // 16 minutes later

	// Should be able to acquire again
	acquired, err = s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestAcquireIngestionLock_DifferentDriversIndependent(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	// Acquire lock for driver 1
	acquired, err := s.AcquireIngestionLock(ctx, 111, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Should be able to acquire lock for driver 2
	acquired, err = s.AcquireIngestionLock(ctx, 222, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestReleaseIngestionLock_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	// Acquire then release
	acquired, err := s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	err = s.ReleaseIngestionLock(ctx, 12345)
	require.NoError(t, err)

	// Should be able to acquire again immediately
	acquired, err = s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestReleaseIngestionLock_NotFoundNoError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Releasing non-existent lock should not error
	err := s.ReleaseIngestionLock(ctx, 99999)
	require.NoError(t, err)
}

func TestGetDriver_IngestionBlockedUntilFromLock(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	// Initially no lock
	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Nil(t, got.IngestionBlockedUntil)

	// Acquire lock
	_, err = s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)

	// Now should see the lock
	got, err = s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got.IngestionBlockedUntil)
	assert.Equal(t, fixedTime.Add(15*time.Minute), *got.IngestionBlockedUntil)

	// Other fields unchanged
	assert.Equal(t, driver.DriverName, got.DriverName)
	assert.Equal(t, driver.FirstLogin, got.FirstLogin)
	assert.Equal(t, driver.LoginCount, got.LoginCount)
}

func TestGetDriver_ExpiredLockNotReturned(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	currentTime := time.Unix(1000, 0)
	s.now = func() time.Time { return currentTime }

	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	// Acquire lock
	_, err := s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)

	// Advance time past lock expiration
	currentTime = time.Unix(1000+16*60, 0) // 16 minutes later

	// Expired lock should not be returned
	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Nil(t, got.IngestionBlockedUntil)
}

func TestSaveDriverSessions_HappyPath(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	sessionStartTime := time.Unix(1700000000, 0)

	sessions := []DriverSession{
		{
			DriverID:              1001,
			SubsessionID:          12345,
			TrackID:               100,
			CarID:                 101,
			SeriesID:              42,
			SeriesName:            "Advanced Mazda MX-5 Cup Series",
			StartTime:             sessionStartTime,
			StartPosition:         1,
			StartPositionInClass:  1,
			FinishPosition:        2,
			FinishPositionInClass: 2,
			Incidents:             3,
			OldCPI:                0.5,
			NewCPI:                0.6,
			OldIRating:            2000,
			NewIRating:            2050,
			OldLicenseLevel:       17,
			NewLicenseLevel:       18,
			OldSubLevel:           381,
			NewSubLevel:           399,
			ReasonOut:             "Running",
		},
		{
			DriverID:              1002,
			SubsessionID:          12345,
			TrackID:               100,
			CarID:                 102,
			SeriesID:              42,
			SeriesName:            "Advanced Mazda MX-5 Cup Series",
			StartTime:             sessionStartTime,
			StartPosition:         2,
			StartPositionInClass:  2,
			FinishPosition:        1,
			FinishPositionInClass: 1,
			Incidents:             0,
			OldCPI:                0.3,
			NewCPI:                0.25,
			OldIRating:            2100,
			NewIRating:            2150,
			OldLicenseLevel:       18,
			NewLicenseLevel:       18,
			OldSubLevel:           425,
			NewSubLevel:           450,
			ReasonOut:             "Running",
		},
	}

	err := s.SaveDriverSessions(ctx, sessions)
	require.NoError(t, err)

	// Verify GetDriverSessionsByTimeRange
	driverSessions, err := s.GetDriverSessionsByTimeRange(ctx, 1001, sessionStartTime.Add(-time.Hour), sessionStartTime.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []DriverSession{sessions[0]}, driverSessions)

	driverSessions, err = s.GetDriverSessionsByTimeRange(ctx, 1002, sessionStartTime.Add(-time.Hour), sessionStartTime.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []DriverSession{sessions[1]}, driverSessions)
}

func TestSaveDriverSessions_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Saving empty slice should succeed
	err := s.SaveDriverSessions(ctx, []DriverSession{})
	require.NoError(t, err)
}

func TestGetDriverSession_Found(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	targetStartTime := time.Unix(1700000000, 0)

	// Insert multiple sessions for multiple drivers as noise
	sessions := []DriverSession{
		// Other driver, different session
		{DriverID: 9999, SubsessionID: 11111, TrackID: 100, CarID: 101, StartTime: time.Unix(1699999000, 0), ReasonOut: "Running", FinishPosition: 5},
		// Target driver, earlier session
		{DriverID: 1001, SubsessionID: 11111, TrackID: 100, CarID: 101, StartTime: time.Unix(1699999000, 0), ReasonOut: "Running", FinishPosition: 10},
		// Target driver, target session - this is the one we want
		{DriverID: 1001, SubsessionID: 12345, TrackID: 100, CarID: 101, StartTime: targetStartTime, FinishPosition: 2, Incidents: 3, ReasonOut: "Running"},
		// Target driver, later session
		{DriverID: 1001, SubsessionID: 22222, TrackID: 100, CarID: 101, StartTime: time.Unix(1700001000, 0), ReasonOut: "Running", FinishPosition: 1},
		// Other driver, same time as target
		{DriverID: 8888, SubsessionID: 12345, TrackID: 100, CarID: 102, StartTime: targetStartTime, ReasonOut: "Running", FinishPosition: 7},
	}
	require.NoError(t, s.SaveDriverSessions(ctx, sessions))

	got, err := s.GetDriverSession(ctx, 1001, targetStartTime)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, int64(1001), got.DriverID)
	assert.Equal(t, int64(12345), got.SubsessionID)
	assert.Equal(t, 2, got.FinishPosition)
	assert.Equal(t, 3, got.Incidents)
}

func TestGetDriverSession_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Add some data as noise
	sessions := []DriverSession{
		{DriverID: 1001, SubsessionID: 12345, TrackID: 100, CarID: 101, StartTime: time.Unix(1700000000, 0), ReasonOut: "Running"},
	}
	require.NoError(t, s.SaveDriverSessions(ctx, sessions))

	// Query for non-existent driver
	got, err := s.GetDriverSession(ctx, 99999, time.Unix(1700000000, 0))
	require.NoError(t, err)
	assert.Nil(t, got)

	// Query for existing driver but wrong time
	got, err = s.GetDriverSession(ctx, 1001, time.Unix(1600000000, 0))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetDriverSessionsByTimeRange_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	sessions, err := s.GetDriverSessionsByTimeRange(ctx, 99999, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestGetDriverSessionsByTimeRange_DateRangeFiltering(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert sessions at different times
	times := []time.Time{
		time.Unix(1000, 0),
		time.Unix(2000, 0),
		time.Unix(3000, 0),
		time.Unix(4000, 0),
	}

	for i, startTime := range times {
		sessions := []DriverSession{
			{
				DriverID:     1001,
				SubsessionID: int64(i + 1),
				TrackID:      100,
				CarID:        101,
				StartTime:    startTime,
				ReasonOut:    "Running",
			},
		}
		require.NoError(t, s.SaveDriverSessions(ctx, sessions))
	}

	// Query middle range - expect newest first
	sessions, err := s.GetDriverSessionsByTimeRange(ctx, 1001, time.Unix(2000, 0), time.Unix(3000, 0))
	require.NoError(t, err)
	assert.Equal(t, []DriverSession{
		{DriverID: 1001, SubsessionID: 3, TrackID: 100, CarID: 101, StartTime: time.Unix(3000, 0), ReasonOut: "Running"},
		{DriverID: 1001, SubsessionID: 2, TrackID: 100, CarID: 101, StartTime: time.Unix(2000, 0), ReasonOut: "Running"},
	}, sessions)
}

func TestGetDriverSessions_EmptyStartTimes(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	sessions, err := s.GetDriverSessions(ctx, 12345, []time.Time{})
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestGetDriverSessions_SingleMatch(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	startTime := time.Unix(1000, 0)
	require.NoError(t, s.SaveDriverSessions(ctx, []DriverSession{
		{
			DriverID:     1001,
			SubsessionID: 100,
			TrackID:      10,
			CarID:        20,
			StartTime:    startTime,
			ReasonOut:    "Running",
		},
	}))

	sessions, err := s.GetDriverSessions(ctx, 1001, []time.Time{startTime})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, int64(100), sessions[0].SubsessionID)
}

func TestGetDriverSessions_MultipleMatches(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	times := []time.Time{
		time.Unix(1000, 0),
		time.Unix(2000, 0),
		time.Unix(3000, 0),
	}

	for i, startTime := range times {
		require.NoError(t, s.SaveDriverSessions(ctx, []DriverSession{
			{
				DriverID:     1001,
				SubsessionID: int64(100 + i),
				TrackID:      10,
				CarID:        20,
				StartTime:    startTime,
				ReasonOut:    "Running",
			},
		}))
	}

	sessions, err := s.GetDriverSessions(ctx, 1001, times)
	require.NoError(t, err)
	require.Len(t, sessions, 3)

	// BatchGetItem doesn't guarantee order, so check by subsession ID
	subsessionIDs := make(map[int64]bool)
	for _, s := range sessions {
		subsessionIDs[s.SubsessionID] = true
	}
	assert.True(t, subsessionIDs[100])
	assert.True(t, subsessionIDs[101])
	assert.True(t, subsessionIDs[102])
}

func TestGetDriverSessions_PartialMatches(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	existingTime := time.Unix(1000, 0)
	require.NoError(t, s.SaveDriverSessions(ctx, []DriverSession{
		{
			DriverID:     1001,
			SubsessionID: 100,
			TrackID:      10,
			CarID:        20,
			StartTime:    existingTime,
			ReasonOut:    "Running",
		},
	}))

	// Query with one existing and one non-existing time
	sessions, err := s.GetDriverSessions(ctx, 1001, []time.Time{existingTime, time.Unix(9999, 0)})
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, int64(100), sessions[0].SubsessionID)
}

func TestGetDriverSessions_NoMatches(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Query for times that don't exist
	sessions, err := s.GetDriverSessions(ctx, 1001, []time.Time{time.Unix(9999, 0), time.Unix(8888, 0)})
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestGetDriverSessions_DifferentDriver(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	startTime := time.Unix(1000, 0)
	require.NoError(t, s.SaveDriverSessions(ctx, []DriverSession{
		{
			DriverID:     1001,
			SubsessionID: 100,
			TrackID:      10,
			CarID:        20,
			StartTime:    startTime,
			ReasonOut:    "Running",
		},
	}))

	// Query with correct time but wrong driver
	sessions, err := s.GetDriverSessions(ctx, 9999, []time.Time{startTime})
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestGetOptionalStringSliceAttr_NotPresent(t *testing.T) {
	item := map[string]types.AttributeValue{}

	result, err := getOptionalStringSliceAttr(item, "entitlements")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetOptionalStringSliceAttr_ValidList(t *testing.T) {
	item := map[string]types.AttributeValue{
		"entitlements": &types.AttributeValueMemberL{
			Value: []types.AttributeValue{
				&types.AttributeValueMemberS{Value: "developer"},
				&types.AttributeValueMemberS{Value: "beta-tester"},
			},
		},
	}

	result, err := getOptionalStringSliceAttr(item, "entitlements")
	require.NoError(t, err)
	assert.Equal(t, []string{"developer", "beta-tester"}, result)
}

func TestGetOptionalStringSliceAttr_EmptyList(t *testing.T) {
	item := map[string]types.AttributeValue{
		"entitlements": &types.AttributeValueMemberL{
			Value: []types.AttributeValue{},
		},
	}

	result, err := getOptionalStringSliceAttr(item, "entitlements")
	require.NoError(t, err)
	assert.Equal(t, []string{}, result)
}

func TestGetOptionalStringSliceAttr_WrongType(t *testing.T) {
	item := map[string]types.AttributeValue{
		"entitlements": &types.AttributeValueMemberS{Value: "not-a-list"},
	}

	result, err := getOptionalStringSliceAttr(item, "entitlements")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a list")
	assert.Nil(t, result)
}

func TestGetOptionalStringSliceAttr_ListContainsNonString(t *testing.T) {
	item := map[string]types.AttributeValue{
		"entitlements": &types.AttributeValueMemberL{
			Value: []types.AttributeValue{
				&types.AttributeValueMemberS{Value: "developer"},
				&types.AttributeValueMemberN{Value: "123"},
			},
		},
	}

	result, err := getOptionalStringSliceAttr(item, "entitlements")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element at index 1 is not a string")
	assert.Nil(t, result)
}

func TestDeleteDriverRaces_DeletesAllExceptInfo(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert a driver with RacesIngestedTo set
	ingestedTo := time.Unix(5000, 0)
	driver := Driver{
		DriverID:        12345,
		DriverName:      "Jon Sabados",
		MemberSince:     time.Unix(500, 0),
		FirstLogin:      time.Unix(1000, 0),
		LastLogin:       time.Unix(1000, 0),
		LoginCount:      1,
		RacesIngestedTo: &ingestedTo,
		Entitlements:    []string{"developer"},
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	// Add session data for this driver
	sessionStartTime := time.Unix(1700000000, 0)
	driverSessions := []DriverSession{
		{DriverID: 12345, SubsessionID: 12345, TrackID: 100, CarID: 101, StartTime: sessionStartTime, ReasonOut: "Running"},
	}
	require.NoError(t, s.SaveDriverSessions(ctx, driverSessions))

	// Verify driver has sessions
	sessions, err := s.GetDriverSessionsByTimeRange(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	// Delete driver races
	err = s.DeleteDriverRaces(ctx, 12345)
	require.NoError(t, err)

	// Verify driver info still exists with entitlements preserved
	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Jon Sabados", got.DriverName)
	assert.Equal(t, []string{"developer"}, got.Entitlements)

	// Verify RacesIngestedTo is nil'd
	assert.Nil(t, got.RacesIngestedTo)

	// Verify SessionCount is reset to 0
	assert.Equal(t, int64(0), got.SessionCount)

	// Verify sessions are deleted
	sessions, err = s.GetDriverSessionsByTimeRange(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestDeleteDriverRaces_DeletesConnectionsAndLocks(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	// Insert a driver
	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	// Add WebSocket connection
	require.NoError(t, s.SaveConnection(ctx, WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn123",
	}))

	// Acquire ingestion lock
	acquired, err := s.AcquireIngestionLock(ctx, 12345, 15*time.Minute)
	require.NoError(t, err)
	assert.True(t, acquired)

	// Verify they exist
	conn, err := s.GetConnection(ctx, 12345, "conn123")
	require.NoError(t, err)
	require.NotNil(t, conn)

	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got.IngestionBlockedUntil)

	// Delete driver races
	err = s.DeleteDriverRaces(ctx, 12345)
	require.NoError(t, err)

	// Verify connection is gone
	conn, err = s.GetConnection(ctx, 12345, "conn123")
	require.NoError(t, err)
	assert.Nil(t, conn)

	// Verify ingestion lock is gone
	got, err = s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Nil(t, got.IngestionBlockedUntil)
}

func TestDeleteDriverRaces_NoRecordsToDelete(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert a driver with no other records
	driver := Driver{
		DriverID:    12345,
		DriverName:  "Jon Sabados",
		MemberSince: time.Unix(500, 0),
		FirstLogin:  time.Unix(1000, 0),
		LastLogin:   time.Unix(1000, 0),
		LoginCount:  1,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	// Delete should succeed even with nothing to delete
	err := s.DeleteDriverRaces(ctx, 12345)
	require.NoError(t, err)

	// Driver info should still exist
	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Jon Sabados", got.DriverName)
}

func TestDeleteDriverRaces_MultipleSessions(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert a driver
	ingestedTo := time.Unix(5000, 0)
	driver := Driver{
		DriverID:        12345,
		DriverName:      "Jon Sabados",
		MemberSince:     time.Unix(500, 0),
		FirstLogin:      time.Unix(1000, 0),
		LastLogin:       time.Unix(1000, 0),
		LoginCount:      1,
		RacesIngestedTo: &ingestedTo,
	}
	require.NoError(t, s.InsertDriver(ctx, driver))

	// Add multiple sessions
	for i := 0; i < 30; i++ { // More than maxBatchWriteItems (25) to test batching
		startTime := time.Unix(int64(1700000000+i*1000), 0)
		driverSessions := []DriverSession{
			{DriverID: 12345, SubsessionID: int64(i + 1), TrackID: 100, CarID: 101, StartTime: startTime, ReasonOut: "Running"},
		}
		require.NoError(t, s.SaveDriverSessions(ctx, driverSessions))
	}

	// Verify driver has sessions
	sessions, err := s.GetDriverSessionsByTimeRange(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Len(t, sessions, 30)

	// Delete driver races
	err = s.DeleteDriverRaces(ctx, 12345)
	require.NoError(t, err)

	// Verify all sessions are deleted
	sessions, err = s.GetDriverSessionsByTimeRange(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Verify driver info still exists
	got, err := s.GetDriver(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Nil(t, got.RacesIngestedTo)
	assert.Equal(t, int64(0), got.SessionCount)
}

func TestDeleteDriverRaces_IsolatedByDriver(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert two drivers
	driver1 := Driver{DriverID: 111, DriverName: "Driver 1", MemberSince: time.Unix(500, 0), FirstLogin: time.Unix(1000, 0), LastLogin: time.Unix(1000, 0), LoginCount: 1}
	driver2 := Driver{DriverID: 222, DriverName: "Driver 2", MemberSince: time.Unix(500, 0), FirstLogin: time.Unix(1000, 0), LastLogin: time.Unix(1000, 0), LoginCount: 1}
	require.NoError(t, s.InsertDriver(ctx, driver1))
	require.NoError(t, s.InsertDriver(ctx, driver2))

	// Add sessions for both drivers
	driverSessions := []DriverSession{
		{DriverID: 111, SubsessionID: 1, TrackID: 100, CarID: 101, StartTime: time.Unix(1700000000, 0), ReasonOut: "Running"},
		{DriverID: 222, SubsessionID: 1, TrackID: 100, CarID: 102, StartTime: time.Unix(1700000000, 0), ReasonOut: "Running"},
	}
	require.NoError(t, s.SaveDriverSessions(ctx, driverSessions))

	// Delete only driver 111's races
	err := s.DeleteDriverRaces(ctx, 111)
	require.NoError(t, err)

	// Driver 111 should have no sessions
	sessions, err := s.GetDriverSessionsByTimeRange(ctx, 111, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Driver 222 should still have their session
	sessions, err = s.GetDriverSessionsByTimeRange(ctx, 222, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
}

func TestSaveJournalEntry_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	fixedTime := time.Unix(1000, 0)
	s.now = func() time.Time { return fixedTime }

	entry := RaceJournalEntry{
		DriverID: 12345,
		RaceID:   1700000000,
		Notes:    "Great race! Finally nailed turn 3.",
		Tags:     []string{"sentiment:good", "podium", "clean-race"},
	}

	err := s.SaveJournalEntry(ctx, entry)
	require.NoError(t, err)

	// Verify by reading it back
	got, err := s.GetJournalEntry(ctx, 12345, 1700000000)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, entry.DriverID, got.DriverID)
	assert.Equal(t, entry.RaceID, got.RaceID)
	assert.Equal(t, entry.Notes, got.Notes)
	assert.Equal(t, entry.Tags, got.Tags)
	assert.Equal(t, fixedTime, got.CreatedAt)
	assert.Equal(t, fixedTime, got.UpdatedAt)
}

func TestSaveJournalEntry_UpsertPreservesCreatedAt(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	createTime := time.Unix(1000, 0)
	s.now = func() time.Time { return createTime }

	entry := RaceJournalEntry{
		DriverID: 12345,
		RaceID:   1700000000,
		Notes:    "Initial notes",
		Tags:     []string{"sentiment:neutral"},
	}

	err := s.SaveJournalEntry(ctx, entry)
	require.NoError(t, err)

	// Update at a later time
	updateTime := time.Unix(2000, 0)
	s.now = func() time.Time { return updateTime }

	entry.Notes = "Updated notes after reflection"
	entry.Tags = []string{"sentiment:good", "lesson-learned"}

	err = s.SaveJournalEntry(ctx, entry)
	require.NoError(t, err)

	// Verify created_at is preserved, updated_at is changed
	got, err := s.GetJournalEntry(ctx, 12345, 1700000000)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Updated notes after reflection", got.Notes)
	assert.Equal(t, []string{"sentiment:good", "lesson-learned"}, got.Tags)
	assert.Equal(t, createTime, got.CreatedAt, "CreatedAt should be preserved on upsert")
	assert.Equal(t, updateTime, got.UpdatedAt, "UpdatedAt should be changed on upsert")
}

func TestSaveJournalEntry_EmptyTags(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	entry := RaceJournalEntry{
		DriverID: 12345,
		RaceID:   1700000000,
		Notes:    "Just some notes, no tags",
		Tags:     nil,
	}

	err := s.SaveJournalEntry(ctx, entry)
	require.NoError(t, err)

	got, err := s.GetJournalEntry(ctx, 12345, 1700000000)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Just some notes, no tags", got.Notes)
	assert.Empty(t, got.Tags)
}

func TestGetJournalEntry_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	got, err := s.GetJournalEntry(ctx, 99999, 1700000000)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetJournalEntries_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	entries, err := s.GetJournalEntries(ctx, 99999, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestGetJournalEntries_TimeRangeFiltering(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert entries at different times (race_id is unix timestamp)
	entries := []RaceJournalEntry{
		{DriverID: 1, RaceID: 1000, Notes: "race 1"},
		{DriverID: 1, RaceID: 2000, Notes: "race 2"},
		{DriverID: 1, RaceID: 3000, Notes: "race 3"},
		{DriverID: 1, RaceID: 4000, Notes: "race 4"},
	}
	for _, e := range entries {
		require.NoError(t, s.SaveJournalEntry(ctx, e))
	}

	// Query middle range (inclusive)
	got, err := s.GetJournalEntries(ctx, 1, time.Unix(2000, 0), time.Unix(3000, 0))
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Results should be newest first
	assert.Equal(t, int64(3000), got[0].RaceID)
	assert.Equal(t, int64(2000), got[1].RaceID)
}

func TestGetJournalEntries_ReturnsNewestFirst(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert in random order
	entries := []RaceJournalEntry{
		{DriverID: 1, RaceID: 3000, Notes: "middle"},
		{DriverID: 1, RaceID: 1000, Notes: "oldest"},
		{DriverID: 1, RaceID: 5000, Notes: "newest"},
	}
	for _, e := range entries {
		require.NoError(t, s.SaveJournalEntry(ctx, e))
	}

	got, err := s.GetJournalEntries(ctx, 1, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, int64(5000), got[0].RaceID, "newest should be first")
	assert.Equal(t, int64(3000), got[1].RaceID)
	assert.Equal(t, int64(1000), got[2].RaceID, "oldest should be last")
}

func TestGetJournalEntries_IsolatedByDriver(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert entries for different drivers at same race time
	require.NoError(t, s.SaveJournalEntry(ctx, RaceJournalEntry{DriverID: 111, RaceID: 1000, Notes: "driver 1 notes"}))
	require.NoError(t, s.SaveJournalEntry(ctx, RaceJournalEntry{DriverID: 222, RaceID: 1000, Notes: "driver 2 notes"}))

	// Query for driver 111 only
	got, err := s.GetJournalEntries(ctx, 111, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "driver 1 notes", got[0].Notes)
}

func TestDeleteJournalEntry_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	entry := RaceJournalEntry{
		DriverID: 12345,
		RaceID:   1700000000,
		Notes:    "Entry to delete",
	}
	require.NoError(t, s.SaveJournalEntry(ctx, entry))

	// Verify it exists
	got, err := s.GetJournalEntry(ctx, 12345, 1700000000)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Delete it
	err = s.DeleteJournalEntry(ctx, 12345, 1700000000)
	require.NoError(t, err)

	// Verify it's gone
	got, err = s.GetJournalEntry(ctx, 12345, 1700000000)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestDeleteJournalEntry_NotFoundNoError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Deleting non-existent entry should not error (idempotent)
	err := s.DeleteJournalEntry(ctx, 99999, 1700000000)
	require.NoError(t, err)
}

func setupTestStore(t *testing.T) *DynamoStore {
	t.Helper()
	t.Parallel()

	tableName := fmt.Sprintf("test-%s-%d", t.Name(), time.Now().UnixNano())

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
	)
	require.NoError(t, err)

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(localDynamoEndpoint)
	})

	_, err = client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("partition_key"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("sort_key"), KeyType: types.KeyTypeRange},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("partition_key"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("sort_key"), AttributeType: types.ScalarAttributeTypeS},
		},
		BillingMode: types.BillingModePayPerRequest,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
			TableName: aws.String(tableName),
		})
	})

	return NewDynamoStore(client, tableName)
}
