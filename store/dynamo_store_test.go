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
