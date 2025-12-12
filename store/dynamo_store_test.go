package store_test

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
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const localDynamoEndpoint = "http://localhost:8000"

func TestInsertTrack_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	track := store.Track{
		ID:   1,
		Name: "Daytona International Speedway",
	}

	err := s.InsertTrack(ctx, track)
	require.NoError(t, err)

	// Verify by reading it back
	got, err := s.GetTrack(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, &track, got)
}

func TestInsertTrack_DuplicateReturnsError(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	track := store.Track{
		ID:   1,
		Name: "Daytona International Speedway",
	}

	err := s.InsertTrack(ctx, track)
	require.NoError(t, err)

	// Try to insert again with same ID
	err = s.InsertTrack(ctx, track)
	assert.ErrorIs(t, err, store.ErrEntityAlreadyExists)
}

func TestGetTrack_NotFound(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	got, err := s.GetTrack(ctx, 999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetGlobalCounters_Empty(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, &store.GlobalCounters{}, counters)
}

func TestGetGlobalCounters_AfterInserts(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert a couple tracks
	require.NoError(t, s.InsertTrack(ctx, store.Track{ID: 1, Name: "Track 1"}))
	require.NoError(t, s.InsertTrack(ctx, store.Track{ID: 2, Name: "Track 2"}))

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counters.Tracks)
}

func TestAddDriverNote_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	note := store.DriverNote{
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

	note := store.DriverNote{
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
	assert.ErrorIs(t, err, store.ErrEntityAlreadyExists)
}

func TestGetDriverNotes_TimeRangeFiltering(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	// Insert notes at different times
	notes := []store.DriverNote{
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
	require.Len(t, got, 2)
	assert.Equal(t, notes[1], got[0])
	assert.Equal(t, notes[2], got[1])
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

	note1 := store.DriverNote{DriverID: 1, Timestamp: time.Unix(1000, 0), SessionID: 1, LapNumber: 1, Category: "a", Notes: "driver 1 note"}
	note2 := store.DriverNote{DriverID: 2, Timestamp: time.Unix(1000, 0), SessionID: 1, LapNumber: 1, Category: "b", Notes: "driver 2 note"}

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

	require.NoError(t, s.InsertTrack(ctx, store.Track{ID: 1, Name: "Track 1"}))
	require.NoError(t, s.AddDriverNote(ctx, store.DriverNote{
		DriverID:  1,
		Timestamp: time.Unix(1000, 0),
		SessionID: 1,
		LapNumber: 1,
		Category:  "test",
		Notes:     "test note",
	}))
	require.NoError(t, s.AddDriverNote(ctx, store.DriverNote{
		DriverID:  1,
		Timestamp: time.Unix(2000, 0),
		SessionID: 1,
		LapNumber: 2,
		Category:  "test",
		Notes:     "another note",
	}))

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), counters.Tracks)
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

	driver := store.Driver{
		DriverID:   12345,
		DriverName: "Jon Sabados",
		FirstLogin: time.Unix(1000, 0),
		LastLogin:  time.Unix(1000, 0),
		LoginCount: 1,
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

	driver := store.Driver{
		DriverID:   12345,
		DriverName: "Jon Sabados",
		FirstLogin: time.Unix(1000, 0),
		LastLogin:  time.Unix(1000, 0),
		LoginCount: 1,
	}

	err := s.InsertDriver(ctx, driver)
	require.NoError(t, err)

	err = s.InsertDriver(ctx, driver)
	assert.ErrorIs(t, err, store.ErrEntityAlreadyExists)
}

func TestInsertDriver_IncrementsGlobalCounter(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	require.NoError(t, s.InsertDriver(ctx, store.Driver{
		DriverID:   1,
		DriverName: "Driver 1",
		FirstLogin: time.Unix(1000, 0),
		LastLogin:  time.Unix(1000, 0),
		LoginCount: 1,
	}))
	require.NoError(t, s.InsertDriver(ctx, store.Driver{
		DriverID:   2,
		DriverName: "Driver 2",
		FirstLogin: time.Unix(2000, 0),
		LastLogin:  time.Unix(2000, 0),
		LoginCount: 1,
	}))

	counters, err := s.GetGlobalCounters(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counters.Drivers)
}

func TestRecordLogin_Success(t *testing.T) {
	s := setupTestStore(t)
	ctx := context.Background()

	driver := store.Driver{
		DriverID:   12345,
		DriverName: "Jon Sabados",
		FirstLogin: time.Unix(1000, 0),
		LastLogin:  time.Unix(1000, 0),
		LoginCount: 1,
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

	driver := store.Driver{
		DriverID:   12345,
		DriverName: "Jon Sabados",
		FirstLogin: time.Unix(1000, 0),
		LastLogin:  time.Unix(1000, 0),
		LoginCount: 1,
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

	conn := store.WebSocketConnection{
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

	conn := store.WebSocketConnection{
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

	conn := store.WebSocketConnection{
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

	conn := store.WebSocketConnection{
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

	// Create multiple connections for same driver
	require.NoError(t, s.SaveConnection(ctx, store.WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn1",
	}))
	require.NoError(t, s.SaveConnection(ctx, store.WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn2",
	}))
	require.NoError(t, s.SaveConnection(ctx, store.WebSocketConnection{
		DriverID:     12345,
		ConnectionID: "conn3",
	}))

	connections, err := s.GetConnectionsByDriver(ctx, 12345)
	require.NoError(t, err)
	assert.Len(t, connections, 3)

	// Verify all connection IDs are present
	connIDs := make([]string, len(connections))
	for i, c := range connections {
		connIDs[i] = c.ConnectionID
	}
	assert.ElementsMatch(t, []string{"conn1", "conn2", "conn3"}, connIDs)
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
	require.NoError(t, s.SaveConnection(ctx, store.WebSocketConnection{
		DriverID:     111,
		ConnectionID: "conn-driver1",
	}))
	require.NoError(t, s.SaveConnection(ctx, store.WebSocketConnection{
		DriverID:     222,
		ConnectionID: "conn-driver2",
	}))

	// Query for driver 111 only
	connections, err := s.GetConnectionsByDriver(ctx, 111)
	require.NoError(t, err)
	require.Len(t, connections, 1)
	assert.Equal(t, "conn-driver1", connections[0].ConnectionID)
}

func setupTestStore(t *testing.T) *store.DynamoStore {
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

	return store.NewDynamoStore(client, tableName)
}
