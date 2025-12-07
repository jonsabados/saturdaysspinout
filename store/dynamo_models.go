package store

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const partitionKeyName = "partition_key"
const sortKeyName = "sort_key"

const defaultSortKey = "info"

const trackPartitionKeyFormat = "track#%d"

const driverPartitionFormat = "driver#%d"
const driverNoteSortKeyFormat = "note#%d" // note, the number should be a unix timestamp allowing us to find notes within a specified time period

const globalCountersPartitionKey = "global"
const globalCountersSortKey = "counters"
const globalCountersAttributeTracks = "tracks"
const globalCountersAttributeNotes = "notes"

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

	if notesAttr, ok := item[globalCountersAttributeNotes].(*types.AttributeValueMemberN); ok {
		notes, err := strconv.ParseInt(notesAttr.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid 'notes' value: %w", err)
		}
		counters.Notes = notes
	}

	return counters, nil
}

type driverNoteModel struct {
	driverID  int64
	timestamp int64
	sessionID int64
	lapNumber int64
	isMistake bool
	category  string
	notes     string
}

func (n driverNoteModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, n.driverID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(driverNoteSortKeyFormat, n.timestamp)},
		"timestamp":      &types.AttributeValueMemberN{Value: strconv.FormatInt(n.timestamp, 10)},
		"session_id":     &types.AttributeValueMemberN{Value: strconv.FormatInt(n.sessionID, 10)},
		"lap_number":     &types.AttributeValueMemberN{Value: strconv.FormatInt(n.lapNumber, 10)},
		"is_mistake":     &types.AttributeValueMemberBOOL{Value: n.isMistake},
		"category":       &types.AttributeValueMemberS{Value: n.category},
		"notes":          &types.AttributeValueMemberS{Value: n.notes},
	}
}

func driverNoteFromAttributeMap(driverID int64, item map[string]types.AttributeValue) (*DriverNote, error) {
	timestampAttr, ok := item["timestamp"].(*types.AttributeValueMemberN)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'timestamp' attribute")
	}
	timestamp, err := strconv.ParseInt(timestampAttr.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid 'timestamp' value: %w", err)
	}

	sessionIDAttr, ok := item["session_id"].(*types.AttributeValueMemberN)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'session_id' attribute")
	}
	sessionID, err := strconv.ParseInt(sessionIDAttr.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid 'session_id' value: %w", err)
	}

	lapNumberAttr, ok := item["lap_number"].(*types.AttributeValueMemberN)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'lap_number' attribute")
	}
	lapNumber, err := strconv.ParseInt(lapNumberAttr.Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid 'lap_number' value: %w", err)
	}

	isMistakeAttr, ok := item["is_mistake"].(*types.AttributeValueMemberBOOL)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'is_mistake' attribute")
	}

	categoryAttr, ok := item["category"].(*types.AttributeValueMemberS)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'category' attribute")
	}

	notesAttr, ok := item["notes"].(*types.AttributeValueMemberS)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'notes' attribute")
	}

	return &DriverNote{
		DriverID:  driverID,
		Timestamp: time.Unix(timestamp, 0),
		SessionID: sessionID,
		LapNumber: lapNumber,
		IsMistake: isMistakeAttr.Value,
		Category:  categoryAttr.Value,
		Notes:     notesAttr.Value,
	}, nil
}
