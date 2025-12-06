package store

import (
	"context"
	"errors"
	"fmt"

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
			{
				Update: &types.Update{
					TableName: aws.String(s.table),
					Key: map[string]types.AttributeValue{
						partitionKeyName: &types.AttributeValueMemberS{Value: globalCountersPartitionKey},
						sortKeyName:      &types.AttributeValueMemberS{Value: globalCountersSortKey},
					},
					UpdateExpression: aws.String("ADD #tracks :inc"),
					ExpressionAttributeNames: map[string]string{
						"#tracks": globalCountersAttributeTracks,
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":inc": &types.AttributeValueMemberN{Value: "1"},
					},
				},
			},
		},
	})
	if err != nil {
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
	return nil
}
