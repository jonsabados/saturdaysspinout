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

type DynamoStore struct {
	client *dynamodb.Client
	table  string
}

func NewDynamoStore(client *dynamodb.Client, table string) *DynamoStore {
	return &DynamoStore{
		client: client,
		table:  table,
	}
}

func (s *DynamoStore) GetTrack(ctx context.Context, id int64) (*Track, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(trackPartitionKeyFormat, id)},
			sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		},
	})
	if err != nil {
		return nil, err
	}
	if result.Item == nil {
		return nil, nil
	}
	return trackFromAttributeMap(result.Item)
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

func (s *DynamoStore) InsertTrack(ctx context.Context, value Track) error {
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					Item: trackModel{
						id:   value.ID,
						name: value.Name,
					}.toAttributeMap(),
					TableName:           aws.String(s.table),
					ConditionExpression: aws.String("attribute_not_exists(#pk)"),
					ExpressionAttributeNames: map[string]string{
						"#pk": partitionKeyName,
					},
				},
			},
			s.incrementCounter(globalCountersAttributeTracks),
		},
	})
	return mapTransactionError(err)
}

func (s *DynamoStore) GetDriverNotes(ctx context.Context, driverID int64, fromInclusive, toExclusive time.Time) ([]DriverNote, error) {
	result, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.table),
		KeyConditionExpression: aws.String("#pk = :pk AND #sk BETWEEN :from AND :to"),
		ExpressionAttributeNames: map[string]string{
			"#pk": partitionKeyName,
			"#sk": sortKeyName,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":   &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, driverID)},
			":from": &types.AttributeValueMemberS{Value: fmt.Sprintf(driverNoteSortKeyFormat, fromInclusive.Unix())},
			":to":   &types.AttributeValueMemberS{Value: fmt.Sprintf(driverNoteSortKeyFormat, toExclusive.Unix()-1)},
		},
	})
	if err != nil {
		return nil, err
	}

	notes := make([]DriverNote, 0, len(result.Items))
	for _, item := range result.Items {
		note, err := driverNoteFromAttributeMap(driverID, item)
		if err != nil {
			return nil, err
		}
		notes = append(notes, *note)
	}
	return notes, nil
}

func (s *DynamoStore) AddDriverNote(ctx context.Context, note DriverNote) error {
	_, err := s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(s.table),
					Item: driverNoteModel{
						driverID:  note.DriverID,
						timestamp: note.Timestamp.Unix(),
						sessionID: note.SessionID,
						lapNumber: note.LapNumber,
						isMistake: note.IsMistake,
						category:  note.Category,
						notes:     note.Notes,
					}.toAttributeMap(),
					ConditionExpression: aws.String("attribute_not_exists(#pk)"),
					ExpressionAttributeNames: map[string]string{
						"#pk": partitionKeyName,
					},
				},
			},
			s.incrementCounter(globalCountersAttributeNotes),
		},
	})
	return mapTransactionError(err)
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
