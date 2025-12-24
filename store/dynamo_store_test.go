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

func TestAddDriverNote_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	note := DriverNote{
		DriverID:  1,
		Timestamp: time.Unix(1000, 0),
		SessionID: 100,
		LapNumber: 5,
		IsMistake: true,
		Category:  "braking",
		Notes:     "Braked too late into turn 1",
	}

	err := s.AddDriverNote(ctx, note)
	require.NoError(t, err)

	// Verify by reading it back
	notes, err := s.GetDriverNotes(ctx, 1, time.Unix(0, 0), time.Unix(2000, 0))
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, note, notes[0])
}

func TestAddDriverNote_DuplicateReturnsError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	note := DriverNote{
		DriverID:  1,
		Timestamp: time.Unix(1000, 0),
		SessionID: 100,
		LapNumber: 5,
		IsMistake: false,
		Category:  "racing line",
		Notes:     "Good apex",
	}

	err := s.AddDriverNote(ctx, note)
	require.NoError(t, err)

	// Try to insert again with same driver + timestamp
	err = s.AddDriverNote(ctx, note)
	assert.ErrorIs(t, err, ErrEntityAlreadyExists)
}

func TestGetDriverNotes_TimeRangeFiltering(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert notes at different times
	notes := []DriverNote{
		{DriverID: 1, Timestamp: time.Unix(1000, 0), SessionID: 1, LapNumber: 1, Category: "a", Notes: "note 1"},
		{DriverID: 1, Timestamp: time.Unix(2000, 0), SessionID: 1, LapNumber: 2, Category: "b", Notes: "note 2"},
		{DriverID: 1, Timestamp: time.Unix(3000, 0), SessionID: 1, LapNumber: 3, Category: "c", Notes: "note 3"},
		{DriverID: 1, Timestamp: time.Unix(4000, 0), SessionID: 1, LapNumber: 4, Category: "d", Notes: "note 4"},
	}
	for _, n := range notes {
		require.NoError(t, s.AddDriverNote(ctx, n))
	}

	// Query with inclusive start, exclusive end
	got, err := s.GetDriverNotes(ctx, 1, time.Unix(2000, 0), time.Unix(4000, 0))
	require.NoError(t, err)
	assert.Equal(t, []DriverNote{notes[1], notes[2]}, got)
}

func TestGetDriverNotes_EmptyResult(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	notes, err := s.GetDriverNotes(ctx, 999, time.Unix(0, 0), time.Unix(1000, 0))
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestGetDriverNotes_DifferentDriversIsolated(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	note1 := DriverNote{DriverID: 1, Timestamp: time.Unix(1000, 0), SessionID: 1, LapNumber: 1, Category: "a", Notes: "driver 1 note"}
	note2 := DriverNote{DriverID: 2, Timestamp: time.Unix(1000, 0), SessionID: 1, LapNumber: 1, Category: "b", Notes: "driver 2 note"}

	require.NoError(t, s.AddDriverNote(ctx, note1))
	require.NoError(t, s.AddDriverNote(ctx, note2))

	// Query for driver 1 only
	got, err := s.GetDriverNotes(ctx, 1, time.Unix(0, 0), time.Unix(2000, 0))
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, note1, got[0])
}

func TestGetGlobalCounters_IncludesNotes(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, s.AddDriverNote(ctx, DriverNote{
		DriverID:  1,
		Timestamp: time.Unix(1000, 0),
		SessionID: 1,
		LapNumber: 1,
		Category:  "test",
		Notes:     "test note",
	}))
	require.NoError(t, s.AddDriverNote(ctx, DriverNote{
		DriverID:  1,
		Timestamp: time.Unix(2000, 0),
		SessionID: 1,
		LapNumber: 2,
		Category:  "test",
		Notes:     "another note",
	}))

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counters.Notes)
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

func TestPersistSessionData_HappyPath(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	sessionStartTime := time.Unix(1700000000, 0)

	data := SessionDataInsertion{
		SessionEntries: []Session{
			{
				SubsessionID: 12345,
				TrackID:      100,
				StartTime:    sessionStartTime,
				CarClasses: []SessionCarClass{
					{
						SubsessionID:    12345,
						CarClassID:      1,
						StrengthOfField: 2500,
						NumberOfEntries: 20,
						Cars: []SessionCarClassCar{
							{SubsessionID: 12345, CarClassID: 1, CarID: 101},
							{SubsessionID: 12345, CarClassID: 1, CarID: 102},
						},
					},
					{
						SubsessionID:    12345,
						CarClassID:      2,
						StrengthOfField: 1800,
						NumberOfEntries: 10,
						Cars: []SessionCarClassCar{
							{SubsessionID: 12345, CarClassID: 2, CarID: 201},
						},
					},
				},
			},
		},
		SessionDriverEntries: []SessionDriver{
			{
				SubsessionID:          12345,
				DriverID:              1001,
				CarID:                 101,
				StartPosition:         1,
				StartPositionInClass:  1,
				FinishPosition:        2,
				FinishPositionInClass: 2,
				Incidents:             3,
				OldCPI:                0.5,
				NewCPI:                0.6,
				OldIRating:            2000,
				NewIRating:            2050,
				ReasonOut:             "Running",
				AI:                    false,
			},
			{
				SubsessionID:          12345,
				DriverID:              1002,
				CarID:                 102,
				StartPosition:         2,
				StartPositionInClass:  2,
				FinishPosition:        1,
				FinishPositionInClass: 1,
				Incidents:             0,
				OldCPI:                0.3,
				NewCPI:                0.25,
				OldIRating:            2100,
				NewIRating:            2150,
				ReasonOut:             "Running",
				AI:                    false,
			},
		},
		SessionDriverLapEntries: []SessionDriverLap{
			{SubsessionID: 12345, DriverID: 1001, LapNumber: 1, LapTime: 90 * time.Second, Flags: 0, Incident: false, LapEvents: nil},
			{SubsessionID: 12345, DriverID: 1001, LapNumber: 2, LapTime: 88 * time.Second, Flags: 0, Incident: true, LapEvents: []string{"off track"}},
			{SubsessionID: 12345, DriverID: 1001, LapNumber: 3, LapTime: 87 * time.Second, Flags: 0, Incident: false, LapEvents: nil},
			{SubsessionID: 12345, DriverID: 1002, LapNumber: 1, LapTime: 89 * time.Second, Flags: 0, Incident: false, LapEvents: nil},
			{SubsessionID: 12345, DriverID: 1002, LapNumber: 2, LapTime: 86 * time.Second, Flags: 0, Incident: false, LapEvents: nil},
		},
		DriverSessionEntries: []DriverSession{
			{
				DriverID:              1001,
				SubsessionID:          12345,
				TrackID:               100,
				CarID:                 101,
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
				ReasonOut:             "Running",
			},
			{
				DriverID:              1002,
				SubsessionID:          12345,
				TrackID:               100,
				CarID:                 102,
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
				ReasonOut:             "Running",
			},
		},
	}

	err := s.PersistSessionData(ctx, data)
	require.NoError(t, err)

	// Verify GetSession returns session with car classes
	session, err := s.GetSession(ctx, 12345)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, int64(12345), session.SubsessionID)
	assert.Equal(t, int64(100), session.TrackID)
	assert.Equal(t, sessionStartTime, session.StartTime)
	assert.Len(t, session.CarClasses, 2)

	// Find car classes by ID for deterministic assertions
	carClassByID := make(map[int64]SessionCarClass)
	for _, cc := range session.CarClasses {
		carClassByID[cc.CarClassID] = cc
	}
	assert.Equal(t, 2500, carClassByID[1].StrengthOfField)
	assert.Equal(t, 20, carClassByID[1].NumberOfEntries)
	assert.Len(t, carClassByID[1].Cars, 2)
	assert.Equal(t, 1800, carClassByID[2].StrengthOfField)
	assert.Len(t, carClassByID[2].Cars, 1)

	// Verify GetSessionDrivers
	drivers, err := s.GetSessionDrivers(ctx, 12345)
	require.NoError(t, err)
	assert.Len(t, drivers, 2)

	driverByID := make(map[int64]SessionDriver)
	for _, d := range drivers {
		driverByID[d.DriverID] = d
	}
	assert.Equal(t, 2, driverByID[1001].FinishPosition)
	assert.Equal(t, 2, driverByID[1001].FinishPositionInClass)
	assert.Equal(t, 3, driverByID[1001].Incidents)
	assert.Equal(t, 1, driverByID[1002].FinishPosition)
	assert.Equal(t, 1, driverByID[1002].FinishPositionInClass)

	// Verify GetSessionDriverLaps for driver 1001
	laps1001, err := s.GetSessionDriverLaps(ctx, 12345, 1001)
	require.NoError(t, err)
	assert.Len(t, laps1001, 3)

	lapByNumber := make(map[int]SessionDriverLap)
	for _, l := range laps1001 {
		lapByNumber[l.LapNumber] = l
	}
	assert.Equal(t, 90*time.Second, lapByNumber[1].LapTime)
	assert.False(t, lapByNumber[1].Incident)
	assert.Equal(t, 88*time.Second, lapByNumber[2].LapTime)
	assert.True(t, lapByNumber[2].Incident)
	assert.Equal(t, []string{"off track"}, lapByNumber[2].LapEvents)

	// Verify GetSessionDriverLaps for driver 1002
	laps1002, err := s.GetSessionDriverLaps(ctx, 12345, 1002)
	require.NoError(t, err)
	assert.Len(t, laps1002, 2)

	// Verify GetDriverSessions
	driverSessions, err := s.GetDriverSessions(ctx, 1001, sessionStartTime.Add(-time.Hour), sessionStartTime.Add(time.Hour))
	require.NoError(t, err)
	assert.Equal(t, []DriverSession{
		{
			DriverID:              1001,
			SubsessionID:          12345,
			TrackID:               100,
			CarID:                 101,
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
			ReasonOut:             "Running",
		},
	}, driverSessions)

	// Verify global counters
	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counters.Sessions)
	assert.Equal(t, int64(5), counters.Laps)
}

func TestPersistSessionData_DuplicateSessionReturnsError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	data := SessionDataInsertion{
		SessionEntries: []Session{
			{
				SubsessionID: 12345,
				TrackID:      100,
				StartTime:    time.Unix(1700000000, 0),
			},
		},
	}

	err := s.PersistSessionData(ctx, data)
	require.NoError(t, err)

	// Try to insert again - should fail on session record
	err = s.PersistSessionData(ctx, data)
	assert.ErrorIs(t, err, ErrEntityAlreadyExists)
}

func TestGetSession_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	session, err := s.GetSession(ctx, 99999)
	require.NoError(t, err)
	assert.Nil(t, session)
}

func TestGetSessionDrivers_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	drivers, err := s.GetSessionDrivers(ctx, 99999)
	require.NoError(t, err)
	assert.Empty(t, drivers)
}

func TestGetSessionDriverLaps_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	laps, err := s.GetSessionDriverLaps(ctx, 99999, 1)
	require.NoError(t, err)
	assert.Empty(t, laps)
}

func TestGetDriverSession_Found(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	targetStartTime := time.Unix(1700000000, 0)

	// Insert multiple sessions for multiple drivers as noise
	data := SessionDataInsertion{
		SessionEntries: []Session{
			{SubsessionID: 11111, TrackID: 100, StartTime: time.Unix(1699999000, 0)},
			{SubsessionID: 12345, TrackID: 100, StartTime: targetStartTime},
			{SubsessionID: 22222, TrackID: 100, StartTime: time.Unix(1700001000, 0)},
		},
		DriverSessionEntries: []DriverSession{
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
		},
	}
	require.NoError(t, s.PersistSessionData(ctx, data))

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
	data := SessionDataInsertion{
		SessionEntries: []Session{
			{SubsessionID: 12345, TrackID: 100, StartTime: time.Unix(1700000000, 0)},
		},
		DriverSessionEntries: []DriverSession{
			{DriverID: 1001, SubsessionID: 12345, TrackID: 100, CarID: 101, StartTime: time.Unix(1700000000, 0), ReasonOut: "Running"},
		},
	}
	require.NoError(t, s.PersistSessionData(ctx, data))

	// Query for non-existent driver
	got, err := s.GetDriverSession(ctx, 99999, time.Unix(1700000000, 0))
	require.NoError(t, err)
	assert.Nil(t, got)

	// Query for existing driver but wrong time
	got, err = s.GetDriverSession(ctx, 1001, time.Unix(1600000000, 0))
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetDriverSessions_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	sessions, err := s.GetDriverSessions(ctx, 99999, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestGetDriverSessions_DateRangeFiltering(t *testing.T) {
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
		data := SessionDataInsertion{
			SessionEntries: []Session{
				{
					SubsessionID: int64(i + 1),
					TrackID:      100,
					StartTime:    startTime,
				},
			},
			DriverSessionEntries: []DriverSession{
				{
					DriverID:     1001,
					SubsessionID: int64(i + 1),
					TrackID:      100,
					CarID:        101,
					StartTime:    startTime,
					ReasonOut:    "Running",
				},
			},
		}
		require.NoError(t, s.PersistSessionData(ctx, data))
	}

	// Query middle range - expect newest first
	sessions, err := s.GetDriverSessions(ctx, 1001, time.Unix(2000, 0), time.Unix(3000, 0))
	require.NoError(t, err)
	assert.Equal(t, []DriverSession{
		{DriverID: 1001, SubsessionID: 3, TrackID: 100, CarID: 101, StartTime: time.Unix(3000, 0), ReasonOut: "Running"},
		{DriverID: 1001, SubsessionID: 2, TrackID: 100, CarID: 101, StartTime: time.Unix(2000, 0), ReasonOut: "Running"},
	}, sessions)
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
	data := SessionDataInsertion{
		SessionEntries: []Session{
			{SubsessionID: 12345, TrackID: 100, StartTime: sessionStartTime},
		},
		DriverSessionEntries: []DriverSession{
			{DriverID: 12345, SubsessionID: 12345, TrackID: 100, CarID: 101, StartTime: sessionStartTime, ReasonOut: "Running"},
		},
	}
	require.NoError(t, s.PersistSessionData(ctx, data))

	// Verify driver has sessions
	sessions, err := s.GetDriverSessions(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
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
	sessions, err = s.GetDriverSessions(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
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
		data := SessionDataInsertion{
			SessionEntries: []Session{
				{SubsessionID: int64(i + 1), TrackID: 100, StartTime: startTime},
			},
			DriverSessionEntries: []DriverSession{
				{DriverID: 12345, SubsessionID: int64(i + 1), TrackID: 100, CarID: 101, StartTime: startTime, ReasonOut: "Running"},
			},
		}
		require.NoError(t, s.PersistSessionData(ctx, data))
	}

	// Verify driver has sessions
	sessions, err := s.GetDriverSessions(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Len(t, sessions, 30)

	// Delete driver races
	err = s.DeleteDriverRaces(ctx, 12345)
	require.NoError(t, err)

	// Verify all sessions are deleted
	sessions, err = s.GetDriverSessions(ctx, 12345, time.Unix(0, 0), time.Unix(9999999999, 0))
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
	data := SessionDataInsertion{
		SessionEntries: []Session{
			{SubsessionID: 1, TrackID: 100, StartTime: time.Unix(1700000000, 0)},
		},
		DriverSessionEntries: []DriverSession{
			{DriverID: 111, SubsessionID: 1, TrackID: 100, CarID: 101, StartTime: time.Unix(1700000000, 0), ReasonOut: "Running"},
			{DriverID: 222, SubsessionID: 1, TrackID: 100, CarID: 102, StartTime: time.Unix(1700000000, 0), ReasonOut: "Running"},
		},
	}
	require.NoError(t, s.PersistSessionData(ctx, data))

	// Delete only driver 111's races
	err := s.DeleteDriverRaces(ctx, 111)
	require.NoError(t, err)

	// Driver 111 should have no sessions
	sessions, err := s.GetDriverSessions(ctx, 111, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Empty(t, sessions)

	// Driver 222 should still have their session
	sessions, err = s.GetDriverSessions(ctx, 222, time.Unix(0, 0), time.Unix(9999999999, 0))
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
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
