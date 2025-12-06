package store

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const partitionKeyName = "partition_key"
const sortKeyName = "sort_key"

const defaultSortKey = "info"

const trackPartitionKeyFormat = "track#%d"

const globalCountersPartitionKey = "global"
const globalCountersSortKey = "counters"
const globalCountersAttributeTracks = "tracks"

type trackModel struct {
	id   int64
	name string
}

func (t trackModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(trackPartitionKeyFormat, t.id)},
		sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		"id":             &types.AttributeValueMemberN{Value: strconv.FormatInt(t.id, 10)},
		"name":           &types.AttributeValueMemberS{Value: t.name},
	}
}

func trackFromAttributeMap(item map[string]types.AttributeValue) (*Track, error) {
	idAttr, ok := item["id"].(*types.AttributeValueMemberN)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'id' attribute")
	}
	id, err := strconv.ParseInt(idAttr.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid 'id' value: %w", err)
	}

	nameAttr, ok := item["name"].(*types.AttributeValueMemberS)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'name' attribute")
	}

	return &Track{
		ID:   id,
		Name: nameAttr.Value,
	}, nil
}

func globalCountersFromAttributeMap(item map[string]types.AttributeValue) (*GlobalCounters, error) {
	counters := &GlobalCounters{}

	if tracksAttr, ok := item[globalCountersAttributeTracks].(*types.AttributeValueMemberN); ok {
		tracks, err := strconv.ParseInt(tracksAttr.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid 'tracks' value: %w", err)
		}
		counters.Tracks = tracks
	}

	return counters, nil
}
