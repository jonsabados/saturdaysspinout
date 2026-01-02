package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const wsConnectionTTLDuration = 24 * time.Hour
const maxTransactWriteItems = 100
const maxBatchWriteItems = 25

type DynamoStore struct {
	client *dynamodb.Client
	table  string
	now    func() time.Time
}

func NewDynamoStore(client *dynamodb.Client, table string) *DynamoStore {
	return &DynamoStore{
		client: client,
		table:  table,
		now:    time.Now,
	}
}

func (s *DynamoStore) GetGlobalCounters(ctx context.Context) (*GlobalCounters, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: globalCountersPartitionKey},
			sortKeyName:      &types.AttributeValueMemberS{Value: globalCountersSortKey},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return &GlobalCounters{}, nil
	}
	return globalCountersFromAttributeMap(result.Item)
}

func mapTransactionError(err error) error {
	if err == nil {
		return nil
	}
	var txErr *types.TransactionCanceledException
	if errors.As(err, &txErr) {
		for _, reason := range txErr.CancellationReasons {
			if reason.Code != nil && *reason.Code == "ConditionalCheckFailed" {
				return ErrEntityAlreadyExists
			}
		}
	}
	return err
}

func (s *DynamoStore) GetDriver(ctx context.Context, driverID int64) (*Driver, error) {
	pk := fmt.Sprintf(driverPartitionFormat, driverID)

	result, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			s.table: {
				Keys: []map[string]types.AttributeValue{
					{
						partitionKeyName: &types.AttributeValueMemberS{Value: pk},
						sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
					},
					{
						partitionKeyName: &types.AttributeValueMemberS{Value: pk},
						sortKeyName:      &types.AttributeValueMemberS{Value: ingestionLockSortKey},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	var driver *Driver
	var lockedUntil *time.Time

	for _, item := range result.Responses[s.table] {
		sk, err := getStringAttr(item, sortKeyName)
		if err != nil {
			return nil, fmt.Errorf("reading sort key from driver item: %w", err)
		}
		switch sk {
		case defaultSortKey:
			driver, err = driverFromAttributeMap(item)
			if err != nil {
				return nil, err
			}
		case ingestionLockSortKey:
			if lu, ok := getOptionalInt64Attr(item, "locked_until"); ok {
				t := time.Unix(lu, 0)
				if t.After(s.now()) {
					lockedUntil = &t
				}
			}
		}
	}

	if driver == nil {
		return nil, nil
	}

	driver.IngestionBlockedUntil = lockedUntil
	return driver, nil
}

func (s *DynamoStore) InsertDriver(ctx context.Context, driver Driver) error {
	model := driverModel{
		driverID:     driver.DriverID,
		driverName:   driver.DriverName,
		memberSince:  toUnixSeconds(driver.MemberSince),
		firstLogin:   toUnixSeconds(driver.FirstLogin),
		lastLogin:    toUnixSeconds(driver.LastLogin),
		loginCount:   driver.LoginCount,
		entitlements: driver.Entitlements,
	}
	if driver.RacesIngestedTo != nil {
		rit := toUnixSeconds(*driver.RacesIngestedTo)
		model.racesIngestedTo = &rit
	}

	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					Item:                model.toAttributeMap(),
					TableName:           aws.String(s.table),
					ConditionExpression: aws.String("attribute_not_exists(#pk)"),
					ExpressionAttributeNames: map[string]string{
						"#pk": partitionKeyName,
					},
				},
			},
			s.incrementCounter(globalCountersAttributeDrivers),
		},
	})
	return mapTransactionError(err)
}

func (s *DynamoStore) RecordLogin(ctx context.Context, driverID int64, loginTime time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		},
		UpdateExpression: aws.String("SET #last_login = :login_time ADD #login_count :inc"),
		ExpressionAttributeNames: map[string]string{
			"#pk":          partitionKeyName,
			"#last_login":  "last_login",
			"#login_count": "login_count",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":login_time": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", toUnixSeconds(loginTime))},
			":inc":        &types.AttributeValueMemberN{Value: "1"},
		},
		ConditionExpression: aws.String("attribute_exists(#pk)"),
	})
	return err
}

func (s *DynamoStore) UpdateDriverRacesIngestedTo(ctx context.Context, driverID int64, racesIngestedTo time.Time) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		},
		UpdateExpression: aws.String("SET #races_ingested_to = :val"),
		ExpressionAttributeNames: map[string]string{
			"#pk":                partitionKeyName,
			"#races_ingested_to": "races_ingested_to",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":val": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", toUnixSeconds(racesIngestedTo))},
		},
		ConditionExpression: aws.String("attribute_exists(#pk)"),
	})
	return err
}

// AcquireIngestionLock attempts to acquire an ingestion lock for a driver.
// Returns (true, nil) if lock acquired, (false, nil) if lock already held, (false, err) on error.
func (s *DynamoStore) AcquireIngestionLock(ctx context.Context, driverID int64, lockDuration time.Duration) (bool, error) {
	now := s.now()
	lockedUntil := now.Add(lockDuration)

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item: ingestionLockModel{
			driverID:    driverID,
			lockedUntil: lockedUntil.Unix(),
		}.toAttributeMap(),
		ConditionExpression: aws.String("attribute_not_exists(#pk) OR #locked_until < :now"),
		ExpressionAttributeNames: map[string]string{
			"#pk":           partitionKeyName,
			"#locked_until": "locked_until",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now.Unix())},
		},
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ReleaseIngestionLock removes the ingestion lock for a driver.
func (s *DynamoStore) ReleaseIngestionLock(ctx context.Context, driverID int64) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: ingestionLockSortKey},
		},
	})
	return err
}

func (s *DynamoStore) incrementCounter(name string) types.TransactWriteItem {
	return types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(s.table),
			Key: map[string]types.AttributeValue{
				partitionKeyName: &types.AttributeValueMemberS{Value: globalCountersPartitionKey},
				sortKeyName:      &types.AttributeValueMemberS{Value: globalCountersSortKey},
			},
			UpdateExpression: aws.String("ADD #counter :inc"),
			ExpressionAttributeNames: map[string]string{
				"#counter": name,
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":inc": &types.AttributeValueMemberN{Value: "1"},
			},
		},
	}
}

func (s *DynamoStore) SaveConnection(ctx context.Context, conn WebSocketConnection) error {
	now := s.now()

	rowsToWrite := wsConnectionModel{
		driverID:     conn.DriverID,
		connectionID: conn.ConnectionID,
		connectedAt:  toUnixSeconds(now),
		ttl:          toUnixSeconds(now.Add(wsConnectionTTLDuration)),
	}.toAttributeMaps()

	toWrite := make([]types.TransactWriteItem, len(rowsToWrite))

	for i, row := range rowsToWrite {
		toWrite[i] = types.TransactWriteItem{
			Put: &types.Put{
				TableName: aws.String(s.table),
				Item:      row,
			},
		}
	}

	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: toWrite,
	})
	return err
}

func (s *DynamoStore) DeleteConnection(ctx context.Context, driverID int64, connectionID string) error {
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Delete: &types.Delete{
					TableName: aws.String(s.table),
					Key: map[string]types.AttributeValue{
						partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
						sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(wsConnectionSortKeyFormat, connectionID)},
					},
				},
			},
			{
				Delete: &types.Delete{
					TableName: aws.String(s.table),
					Key: map[string]types.AttributeValue{
						partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(websocketPartitionFormat, connectionID)},
						sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
					},
				},
			},
		},
	})
	return err
}

func (s *DynamoStore) GetConnectionsByDriver(ctx context.Context, driverID int64) ([]WebSocketConnection, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.table),
		KeyConditionExpression: aws.String("#pk = :pk AND begins_with(#sk, :sk_prefix)"),
		ExpressionAttributeNames: map[string]string{
			"#pk": partitionKeyName,
			"#sk": sortKeyName,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			":sk_prefix": &types.AttributeValueMemberS{Value: "ws#"},
		},
	})
	if err != nil {
		return nil, err
	}

	connections := make([]WebSocketConnection, 0, len(result.Items))
	for _, item := range result.Items {
		conn, err := wsConnectionFromAttributeMap(item)
		if err != nil {
			return nil, err
		}
		connections = append(connections, *conn)
	}
	return connections, nil
}

func (s *DynamoStore) GetDriverIDByConnection(ctx context.Context, connectionID string) (*int64, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(websocketPartitionFormat, connectionID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		},
	})

	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	ret, err := getInt64Attr(result.Item, "driver_id")
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func (s *DynamoStore) GetConnection(ctx context.Context, driverID int64, connectionID string) (*WebSocketConnection, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(wsConnectionSortKeyFormat, connectionID)},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	return wsConnectionFromAttributeMap(result.Item)
}

func (s *DynamoStore) GetDriverSession(ctx context.Context, driverID int64, startTime time.Time) (*DriverSession, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(driverSessionSortKeyFormat, toUnixSeconds(startTime))},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	return driverSessionFromAttributeMap(driverID, result.Item)
}

// GetDriverSessions retrieves specific driver sessions by their exact start times.
// Uses BatchGetItem for efficient fetching. Returns sessions in the order they were found.
func (s *DynamoStore) GetDriverSessions(ctx context.Context, driverID int64, startTimes []time.Time) ([]DriverSession, error) {
	if len(startTimes) == 0 {
		return []DriverSession{}, nil
	}

	pk := fmt.Sprintf(driverPartitionFormat, driverID)
	keys := make([]map[string]types.AttributeValue, len(startTimes))
	for i, startTime := range startTimes {
		keys[i] = map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: pk},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(driverSessionSortKeyFormat, toUnixSeconds(startTime))},
		}
	}

	result, err := s.client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			s.table: {Keys: keys},
		},
	})
	if err != nil {
		return nil, err
	}

	sessions := make([]DriverSession, 0, len(result.Responses[s.table]))
	for _, item := range result.Responses[s.table] {
		session, err := driverSessionFromAttributeMap(driverID, item)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

func (s *DynamoStore) GetDriverSessionsByTimeRange(ctx context.Context, driverID int64, from, to time.Time) ([]DriverSession, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.table),
		KeyConditionExpression: aws.String("#pk = :pk AND #sk BETWEEN :from AND :to"),
		ExpressionAttributeNames: map[string]string{
			"#pk": partitionKeyName,
			"#sk": sortKeyName,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":   &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			":from": &types.AttributeValueMemberS{Value: fmt.Sprintf(driverSessionSortKeyFormat, toUnixSeconds(from))},
			":to":   &types.AttributeValueMemberS{Value: fmt.Sprintf(driverSessionSortKeyFormat, toUnixSeconds(to))},
		},
		ScanIndexForward: aws.Bool(false),
	})
	if err != nil {
		return nil, err
	}

	sessions := make([]DriverSession, 0, len(result.Items))
	for _, item := range result.Items {
		session, err := driverSessionFromAttributeMap(driverID, item)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// SaveDriverSessions saves driver session records and increments session counts atomically.
// Uses transactions to ensure duplicate prevention via key checks.
func (s *DynamoStore) SaveDriverSessions(ctx context.Context, sessions []DriverSession) error {
	if len(sessions) == 0 {
		return nil
	}

	var items []types.TransactWriteItem

	// Track session counts per driver
	driverSessionCounts := make(map[int64]int)

	for _, ds := range sessions {
		driverSessionCounts[ds.DriverID]++
		items = append(items, s.putWithKeyCheck(driverSessionModel{
			driverID:              ds.DriverID,
			subsessionID:          ds.SubsessionID,
			trackID:               ds.TrackID,
			carID:                 ds.CarID,
			seriesID:              ds.SeriesID,
			seriesName:            ds.SeriesName,
			startTime:             toUnixSeconds(ds.StartTime),
			startPosition:         ds.StartPosition,
			startPositionInClass:  ds.StartPositionInClass,
			finishPosition:        ds.FinishPosition,
			finishPositionInClass: ds.FinishPositionInClass,
			incidents:             ds.Incidents,
			oldCPI:                ds.OldCPI,
			newCPI:                ds.NewCPI,
			oldIRating:            ds.OldIRating,
			newIRating:            ds.NewIRating,
			oldLicenseLevel:       ds.OldLicenseLevel,
			newLicenseLevel:       ds.NewLicenseLevel,
			oldSubLevel:           ds.OldSubLevel,
			newSubLevel:           ds.NewSubLevel,
			reasonOut:             ds.ReasonOut,
		}.toAttributeMap()))
	}

	// Increment session count for each driver
	for driverID, count := range driverSessionCounts {
		items = append(items, s.incrementDriverSessionCount(driverID, count))
	}

	return s.executeBatchedTransact(ctx, items)
}

func (s *DynamoStore) executeBatchedTransact(ctx context.Context, items []types.TransactWriteItem) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); i += maxTransactWriteItems {
		end := i + maxTransactWriteItems
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]

		_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: batch,
		})
		if err != nil {
			batchNum := (i / maxTransactWriteItems) + 1
			totalBatches := (len(items) + maxTransactWriteItems - 1) / maxTransactWriteItems
			return fmt.Errorf("batch %d/%d failed: %w", batchNum, totalBatches, mapTransactionError(err))
		}
	}

	return nil
}

func (s *DynamoStore) putWithKeyCheck(item map[string]types.AttributeValue) types.TransactWriteItem {
	return types.TransactWriteItem{
		Put: &types.Put{
			TableName:           aws.String(s.table),
			Item:                item,
			ConditionExpression: aws.String("attribute_not_exists(#pk)"),
			ExpressionAttributeNames: map[string]string{
				"#pk": partitionKeyName,
			},
		},
	}
}

func (s *DynamoStore) incrementDriverSessionCount(driverID int64, count int) types.TransactWriteItem {
	return types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(s.table),
			Key: map[string]types.AttributeValue{
				partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
				sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
			},
			UpdateExpression: aws.String("ADD #session_count :count"),
			ExpressionAttributeNames: map[string]string{
				"#session_count": "session_count",
			},
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":count": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", count)},
			},
		},
	}
}

// DeleteDriverRaces removes all records under a driver's partition except their info record,
// and resets their sync state to appear as if they've never synced (useful for testing initial sync flows).
func (s *DynamoStore) DeleteDriverRaces(ctx context.Context, driverID int64) error {
	pk := fmt.Sprintf(driverPartitionFormat, driverID)

	// Query all items under driver partition, fetching only keys for efficiency
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.table),
		KeyConditionExpression: aws.String("#pk = :pk"),
		ProjectionExpression:   aws.String("#pk, #sk"),
		ExpressionAttributeNames: map[string]string{
			"#pk": partitionKeyName,
			"#sk": sortKeyName,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
	})
	if err != nil {
		return fmt.Errorf("querying driver partition: %w", err)
	}

	// Collect keys to delete (everything except info)
	var keysToDelete []map[string]types.AttributeValue
	for _, item := range result.Items {
		sk, err := getStringAttr(item, sortKeyName)
		if err != nil {
			return fmt.Errorf("reading sort key from driver item: %w", err)
		}
		if sk != defaultSortKey {
			keysToDelete = append(keysToDelete, map[string]types.AttributeValue{
				partitionKeyName: item[partitionKeyName],
				sortKeyName:      item[sortKeyName],
			})
		}
	}

	// Batch delete in chunks of 25
	for i := 0; i < len(keysToDelete); i += maxBatchWriteItems {
		end := i + maxBatchWriteItems
		if end > len(keysToDelete) {
			end = len(keysToDelete)
		}
		batch := keysToDelete[i:end]

		writeRequests := make([]types.WriteRequest, len(batch))
		for j, key := range batch {
			writeRequests[j] = types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{Key: key},
			}
		}

		_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				s.table: writeRequests,
			},
		})
		if err != nil {
			return fmt.Errorf("batch delete failed: %w", err)
		}
	}

	// Reset races_ingested_to to nil and session_count to 0
	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: pk},
			sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		},
		UpdateExpression: aws.String("REMOVE #races_ingested_to SET #session_count = :zero"),
		ExpressionAttributeNames: map[string]string{
			"#races_ingested_to": "races_ingested_to",
			"#session_count":     "session_count",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":zero": &types.AttributeValueMemberN{Value: "0"},
		},
	})
	if err != nil {
		return fmt.Errorf("resetting driver info: %w", err)
	}

	return nil
}

// SaveJournalEntry creates or updates a journal entry for a race (upsert semantics).
// CreatedAt is set on first save; UpdatedAt is always updated.
func (s *DynamoStore) SaveJournalEntry(ctx context.Context, entry RaceJournalEntry) error {
	now := s.now()

	// For upsert: set created_at only if it doesn't exist, always update updated_at
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, entry.DriverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(journalEntrySortKeyFormat, entry.RaceID)},
		},
		UpdateExpression: aws.String("SET #driver_id = :driver_id, #race_id = :race_id, #notes = :notes, #updated_at = :updated_at, #created_at = if_not_exists(#created_at, :created_at), #tags = :tags"),
		ExpressionAttributeNames: map[string]string{
			"#driver_id":  "driver_id",
			"#race_id":    "race_id",
			"#notes":      "notes",
			"#updated_at": "updated_at",
			"#created_at": "created_at",
			"#tags":       "tags",
		},
		ExpressionAttributeValues: s.journalEntryUpdateValues(entry, now),
	})
	return err
}

func (s *DynamoStore) journalEntryUpdateValues(entry RaceJournalEntry, now time.Time) map[string]types.AttributeValue {
	nowUnix := toUnixSeconds(now)
	values := map[string]types.AttributeValue{
		":driver_id":  &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", entry.DriverID)},
		":race_id":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", entry.RaceID)},
		":notes":      &types.AttributeValueMemberS{Value: entry.Notes},
		":updated_at": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nowUnix)},
		":created_at": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", nowUnix)},
	}
	if len(entry.Tags) > 0 {
		tagValues := make([]types.AttributeValue, len(entry.Tags))
		for i, t := range entry.Tags {
			tagValues[i] = &types.AttributeValueMemberS{Value: t}
		}
		values[":tags"] = &types.AttributeValueMemberL{Value: tagValues}
	} else {
		values[":tags"] = &types.AttributeValueMemberL{Value: []types.AttributeValue{}}
	}
	return values
}

// GetJournalEntry retrieves a single journal entry for a specific race.
// Returns nil if no entry exists.
func (s *DynamoStore) GetJournalEntry(ctx context.Context, driverID, raceID int64) (*RaceJournalEntry, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(journalEntrySortKeyFormat, raceID)},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	return journalEntryFromAttributeMap(result.Item)
}

// GetJournalEntries retrieves journal entries for a driver within a time range.
// Returns entries in reverse chronological order (newest first).
func (s *DynamoStore) GetJournalEntries(ctx context.Context, driverID int64, from, to time.Time) ([]RaceJournalEntry, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.table),
		KeyConditionExpression: aws.String("#pk = :pk AND #sk BETWEEN :from AND :to"),
		ExpressionAttributeNames: map[string]string{
			"#pk": partitionKeyName,
			"#sk": sortKeyName,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":   &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			":from": &types.AttributeValueMemberS{Value: fmt.Sprintf(journalEntrySortKeyFormat, toUnixSeconds(from))},
			":to":   &types.AttributeValueMemberS{Value: fmt.Sprintf(journalEntrySortKeyFormat, toUnixSeconds(to))},
		},
		ScanIndexForward: aws.Bool(false), // newest first
	})
	if err != nil {
		return nil, err
	}

	entries := make([]RaceJournalEntry, 0, len(result.Items))
	for _, item := range result.Items {
		entry, err := journalEntryFromAttributeMap(item)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, nil
}

// DeleteJournalEntry removes a journal entry for a specific race.
// Returns nil even if the entry doesn't exist (idempotent delete).
func (s *DynamoStore) DeleteJournalEntry(ctx context.Context, driverID, raceID int64) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(journalEntrySortKeyFormat, raceID)},
		},
	})
	return err
}
