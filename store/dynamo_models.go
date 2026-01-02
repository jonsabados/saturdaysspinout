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

const driverPartitionFormat = "driver#%d"
const wsConnectionSortKeyFormat = "ws#%s"
const ingestionLockSortKey = "ingestion_lock"

const websocketPartitionFormat = "websocket#%s"

const driverSessionSortKeyFormat = "session#%d" // timestamp for ordering
const journalEntrySortKeyFormat = "journal#%d"  // race_id (timestamp) for ordering

const globalCountersPartitionKey = "global"
const globalCountersSortKey = "counters"
const globalCountersAttributeDrivers = "drivers"

func globalCountersFromAttributeMap(item map[string]types.AttributeValue) (*GlobalCounters, error) {
	counters := &GlobalCounters{}

	if driversAttr, ok := item[globalCountersAttributeDrivers].(*types.AttributeValueMemberN); ok {
		drivers, err := strconv.ParseInt(driversAttr.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid 'drivers' value: %w", err)
		}
		counters.Drivers = drivers
	}

	return counters, nil
}

type driverModel struct {
	driverID        int64
	driverName      string
	memberSince     int64
	racesIngestedTo *int64
	firstLogin      int64
	lastLogin       int64
	loginCount      int64
	sessionCount    int64
	entitlements    []string
}

func (d driverModel) toAttributeMap() map[string]types.AttributeValue {
	m := map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, d.driverID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		"driver_id":      &types.AttributeValueMemberN{Value: strconv.FormatInt(d.driverID, 10)},
		"driver_name":    &types.AttributeValueMemberS{Value: d.driverName},
		"member_since":   &types.AttributeValueMemberN{Value: strconv.FormatInt(d.memberSince, 10)},
		"first_login":    &types.AttributeValueMemberN{Value: strconv.FormatInt(d.firstLogin, 10)},
		"last_login":     &types.AttributeValueMemberN{Value: strconv.FormatInt(d.lastLogin, 10)},
		"login_count":    &types.AttributeValueMemberN{Value: strconv.FormatInt(d.loginCount, 10)},
		"session_count":  &types.AttributeValueMemberN{Value: strconv.FormatInt(d.sessionCount, 10)},
	}
	if d.racesIngestedTo != nil {
		m["races_ingested_to"] = &types.AttributeValueMemberN{Value: strconv.FormatInt(*d.racesIngestedTo, 10)}
	}
	if len(d.entitlements) > 0 {
		entitlementValues := make([]types.AttributeValue, len(d.entitlements))
		for i, e := range d.entitlements {
			entitlementValues[i] = &types.AttributeValueMemberS{Value: e}
		}
		m["entitlements"] = &types.AttributeValueMemberL{Value: entitlementValues}
	}
	return m
}

func driverFromAttributeMap(item map[string]types.AttributeValue) (*Driver, error) {
	driverID, err := getInt64Attr(item, "driver_id")
	if err != nil {
		return nil, err
	}
	driverName, err := getStringAttr(item, "driver_name")
	if err != nil {
		return nil, err
	}
	firstLogin, err := getInt64Attr(item, "first_login")
	if err != nil {
		return nil, err
	}
	lastLogin, err := getInt64Attr(item, "last_login")
	if err != nil {
		return nil, err
	}
	loginCount, err := getInt64Attr(item, "login_count")
	if err != nil {
		return nil, err
	}
	memberSince, err := getInt64Attr(item, "member_since")
	if err != nil {
		return nil, err
	}

	var racesIngestedTo *time.Time
	if rit, ok := getOptionalInt64Attr(item, "races_ingested_to"); ok {
		t := time.Unix(rit, 0)
		racesIngestedTo = &t
	}

	sessionCount, _ := getOptionalInt64Attr(item, "session_count")

	entitlements, err := getOptionalStringSliceAttr(item, "entitlements")
	if err != nil {
		return nil, err
	}

	return &Driver{
		DriverID:        driverID,
		DriverName:      driverName,
		MemberSince:     time.Unix(memberSince, 0),
		RacesIngestedTo: racesIngestedTo,
		FirstLogin:      time.Unix(firstLogin, 0),
		LastLogin:       time.Unix(lastLogin, 0),
		LoginCount:      loginCount,
		SessionCount:    sessionCount,
		Entitlements:    entitlements,
	}, nil
}

type wsConnectionModel struct {
	driverID     int64
	connectionID string
	connectedAt  int64
	ttl          int64
}

func (c wsConnectionModel) toAttributeMaps() []map[string]types.AttributeValue {
	return []map[string]types.AttributeValue{
		{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, c.driverID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(wsConnectionSortKeyFormat, c.connectionID)},
			"connection_id":  &types.AttributeValueMemberS{Value: c.connectionID},
			"connected_at":   &types.AttributeValueMemberN{Value: strconv.FormatInt(c.connectedAt, 10)},
			"ttl":            &types.AttributeValueMemberN{Value: strconv.FormatInt(c.ttl, 10)},
		},
		{
			partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(websocketPartitionFormat, c.connectionID)},
			sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
			"driver_id":      &types.AttributeValueMemberN{Value: strconv.FormatInt(c.driverID, 10)},
			"ttl":            &types.AttributeValueMemberN{Value: strconv.FormatInt(c.ttl, 10)},
		},
	}
}

func wsConnectionFromAttributeMap(item map[string]types.AttributeValue) (*WebSocketConnection, error) {
	connectionID, err := getStringAttr(item, "connection_id")
	if err != nil {
		return nil, err
	}
	connectedAt, err := getInt64Attr(item, "connected_at")
	if err != nil {
		return nil, err
	}

	// Extract driver ID from partition key (format: "driver#123")
	pk, err := getStringAttr(item, partitionKeyName)
	if err != nil {
		return nil, fmt.Errorf("missing or invalid partition key")
	}
	var driverID int64
	_, err = fmt.Sscanf(pk, driverPartitionFormat, &driverID)
	if err != nil {
		return nil, fmt.Errorf("invalid partition key format: %w", err)
	}

	return &WebSocketConnection{
		DriverID:     driverID,
		ConnectionID: connectionID,
		ConnectedAt:  time.Unix(connectedAt, 0),
	}, nil
}

// ingestionLockModel represents a lock preventing concurrent ingestion (driver#<id> / ingestion_lock)
type ingestionLockModel struct {
	driverID    int64
	lockedUntil int64
}

func (l ingestionLockModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, l.driverID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: ingestionLockSortKey},
		"locked_until":   &types.AttributeValueMemberN{Value: strconv.FormatInt(l.lockedUntil, 10)},
		"ttl":            &types.AttributeValueMemberN{Value: strconv.FormatInt(l.lockedUntil, 10)},
	}
}

// driverSessionModel represents a driver's participation in a session (driver#<id> / session#<timestamp>)
type driverSessionModel struct {
	driverID              int64
	subsessionID          int64
	trackID               int64
	carID                 int64
	seriesID              int64
	seriesName            string
	startTime             int64
	startPosition         int
	startPositionInClass  int
	finishPosition        int
	finishPositionInClass int
	incidents             int
	oldCPI                float64
	newCPI                float64
	oldIRating            int
	newIRating            int
	oldLicenseLevel       int
	newLicenseLevel       int
	oldSubLevel           int
	newSubLevel           int
	reasonOut             string
}

func (d driverSessionModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName:           &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, d.driverID)},
		sortKeyName:                &types.AttributeValueMemberS{Value: fmt.Sprintf(driverSessionSortKeyFormat, d.startTime)},
		"subsession_id":            &types.AttributeValueMemberN{Value: strconv.FormatInt(d.subsessionID, 10)},
		"track_id":                 &types.AttributeValueMemberN{Value: strconv.FormatInt(d.trackID, 10)},
		"car_id":                   &types.AttributeValueMemberN{Value: strconv.FormatInt(d.carID, 10)},
		"series_id":                &types.AttributeValueMemberN{Value: strconv.FormatInt(d.seriesID, 10)},
		"series_name":              &types.AttributeValueMemberS{Value: d.seriesName},
		"start_time":               &types.AttributeValueMemberN{Value: strconv.FormatInt(d.startTime, 10)},
		"start_position":           &types.AttributeValueMemberN{Value: strconv.Itoa(d.startPosition)},
		"start_position_in_class":  &types.AttributeValueMemberN{Value: strconv.Itoa(d.startPositionInClass)},
		"finish_position":          &types.AttributeValueMemberN{Value: strconv.Itoa(d.finishPosition)},
		"finish_position_in_class": &types.AttributeValueMemberN{Value: strconv.Itoa(d.finishPositionInClass)},
		"incidents":                &types.AttributeValueMemberN{Value: strconv.Itoa(d.incidents)},
		"old_cpi":                  &types.AttributeValueMemberN{Value: strconv.FormatFloat(d.oldCPI, 'f', -1, 64)},
		"new_cpi":                  &types.AttributeValueMemberN{Value: strconv.FormatFloat(d.newCPI, 'f', -1, 64)},
		"old_irating":              &types.AttributeValueMemberN{Value: strconv.Itoa(d.oldIRating)},
		"new_irating":              &types.AttributeValueMemberN{Value: strconv.Itoa(d.newIRating)},
		"old_license_level":        &types.AttributeValueMemberN{Value: strconv.Itoa(d.oldLicenseLevel)},
		"new_license_level":        &types.AttributeValueMemberN{Value: strconv.Itoa(d.newLicenseLevel)},
		"old_sub_level":            &types.AttributeValueMemberN{Value: strconv.Itoa(d.oldSubLevel)},
		"new_sub_level":            &types.AttributeValueMemberN{Value: strconv.Itoa(d.newSubLevel)},
		"reason_out":               &types.AttributeValueMemberS{Value: d.reasonOut},
	}
}

func driverSessionFromAttributeMap(driverID int64, item map[string]types.AttributeValue) (*DriverSession, error) {
	subsessionID, err := getInt64Attr(item, "subsession_id")
	if err != nil {
		return nil, err
	}
	trackID, err := getInt64Attr(item, "track_id")
	if err != nil {
		return nil, err
	}
	carID, err := getInt64Attr(item, "car_id")
	if err != nil {
		return nil, err
	}
	seriesID, err := getInt64Attr(item, "series_id")
	if err != nil {
		return nil, err
	}
	seriesName, err := getStringAttr(item, "series_name")
	if err != nil {
		return nil, err
	}
	startTime, err := getInt64Attr(item, "start_time")
	if err != nil {
		return nil, err
	}
	startPosition, err := getIntAttr(item, "start_position")
	if err != nil {
		return nil, err
	}
	startPositionInClass, err := getIntAttr(item, "start_position_in_class")
	if err != nil {
		return nil, err
	}
	finishPosition, err := getIntAttr(item, "finish_position")
	if err != nil {
		return nil, err
	}
	finishPositionInClass, err := getIntAttr(item, "finish_position_in_class")
	if err != nil {
		return nil, err
	}
	incidents, err := getIntAttr(item, "incidents")
	if err != nil {
		return nil, err
	}
	oldCPI, err := getFloatAttr(item, "old_cpi")
	if err != nil {
		return nil, err
	}
	newCPI, err := getFloatAttr(item, "new_cpi")
	if err != nil {
		return nil, err
	}
	oldIRating, err := getIntAttr(item, "old_irating")
	if err != nil {
		return nil, err
	}
	newIRating, err := getIntAttr(item, "new_irating")
	if err != nil {
		return nil, err
	}
	oldLicenseLevel, err := getIntAttr(item, "old_license_level")
	if err != nil {
		return nil, err
	}
	newLicenseLevel, err := getIntAttr(item, "new_license_level")
	if err != nil {
		return nil, err
	}
	oldSubLevel, err := getIntAttr(item, "old_sub_level")
	if err != nil {
		return nil, err
	}
	newSubLevel, err := getIntAttr(item, "new_sub_level")
	if err != nil {
		return nil, err
	}
	reasonOut, err := getStringAttr(item, "reason_out")
	if err != nil {
		return nil, err
	}

	return &DriverSession{
		DriverID:              driverID,
		SubsessionID:          subsessionID,
		TrackID:               trackID,
		CarID:                 carID,
		SeriesID:              seriesID,
		SeriesName:            seriesName,
		StartTime:             time.Unix(startTime, 0),
		StartPosition:         startPosition,
		StartPositionInClass:  startPositionInClass,
		FinishPosition:        finishPosition,
		FinishPositionInClass: finishPositionInClass,
		Incidents:             incidents,
		OldCPI:                oldCPI,
		NewCPI:                newCPI,
		OldIRating:            oldIRating,
		NewIRating:            newIRating,
		OldLicenseLevel:       oldLicenseLevel,
		NewLicenseLevel:       newLicenseLevel,
		OldSubLevel:           oldSubLevel,
		NewSubLevel:           newSubLevel,
		ReasonOut:             reasonOut,
	}, nil
}

// journalEntryModel represents a journal entry for a race (driver#<id> / journal#<race_id>)
type journalEntryModel struct {
	driverID  int64
	raceID    int64
	createdAt int64
	updatedAt int64
	notes     string
	tags      []string
}

func (j journalEntryModel) toAttributeMap() map[string]types.AttributeValue {
	m := map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, j.driverID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(journalEntrySortKeyFormat, j.raceID)},
		"driver_id":      &types.AttributeValueMemberN{Value: strconv.FormatInt(j.driverID, 10)},
		"race_id":        &types.AttributeValueMemberN{Value: strconv.FormatInt(j.raceID, 10)},
		"created_at":     &types.AttributeValueMemberN{Value: strconv.FormatInt(j.createdAt, 10)},
		"updated_at":     &types.AttributeValueMemberN{Value: strconv.FormatInt(j.updatedAt, 10)},
		"notes":          &types.AttributeValueMemberS{Value: j.notes},
	}
	if len(j.tags) > 0 {
		tagValues := make([]types.AttributeValue, len(j.tags))
		for i, t := range j.tags {
			tagValues[i] = &types.AttributeValueMemberS{Value: t}
		}
		m["tags"] = &types.AttributeValueMemberL{Value: tagValues}
	}
	return m
}

func journalEntryFromAttributeMap(item map[string]types.AttributeValue) (*RaceJournalEntry, error) {
	driverID, err := getInt64Attr(item, "driver_id")
	if err != nil {
		return nil, err
	}
	raceID, err := getInt64Attr(item, "race_id")
	if err != nil {
		return nil, err
	}
	createdAt, err := getInt64Attr(item, "created_at")
	if err != nil {
		return nil, err
	}
	updatedAt, err := getInt64Attr(item, "updated_at")
	if err != nil {
		return nil, err
	}
	notes, err := getStringAttr(item, "notes")
	if err != nil {
		return nil, err
	}
	tags, err := getOptionalStringSliceAttr(item, "tags")
	if err != nil {
		return nil, err
	}

	return &RaceJournalEntry{
		DriverID:  driverID,
		RaceID:    raceID,
		CreatedAt: time.Unix(createdAt, 0),
		UpdatedAt: time.Unix(updatedAt, 0),
		Notes:     notes,
		Tags:      tags,
	}, nil
}

func getInt64Attr(item map[string]types.AttributeValue, name string) (int64, error) {
	attr, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0, fmt.Errorf("missing or invalid '%s' attribute", name)
	}
	val, err := strconv.ParseInt(attr.Value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid '%s' value: %w", name, err)
	}
	return val, nil
}

func getOptionalInt64Attr(item map[string]types.AttributeValue, name string) (int64, bool) {
	attr, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0, false
	}
	val, err := strconv.ParseInt(attr.Value, 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func getIntAttr(item map[string]types.AttributeValue, name string) (int, error) {
	attr, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0, fmt.Errorf("missing or invalid '%s' attribute", name)
	}
	val, err := strconv.Atoi(attr.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid '%s' value: %w", name, err)
	}
	return val, nil
}

func getFloatAttr(item map[string]types.AttributeValue, name string) (float64, error) {
	attr, ok := item[name].(*types.AttributeValueMemberN)
	if !ok {
		return 0, fmt.Errorf("missing or invalid '%s' attribute", name)
	}
	val, err := strconv.ParseFloat(attr.Value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid '%s' value: %w", name, err)
	}
	return val, nil
}

func getStringAttr(item map[string]types.AttributeValue, name string) (string, error) {
	attr, ok := item[name].(*types.AttributeValueMemberS)
	if !ok {
		return "", fmt.Errorf("missing or invalid '%s' attribute", name)
	}
	return attr.Value, nil
}

func getBoolAttr(item map[string]types.AttributeValue, name string) (bool, error) {
	attr, ok := item[name].(*types.AttributeValueMemberBOOL)
	if !ok {
		return false, fmt.Errorf("missing or invalid '%s' attribute", name)
	}
	return attr.Value, nil
}

func getOptionalStringSliceAttr(item map[string]types.AttributeValue, name string) ([]string, error) {
	attr, ok := item[name]
	if !ok || attr == nil {
		return nil, nil
	}
	listAttr, ok := attr.(*types.AttributeValueMemberL)
	if !ok {
		return nil, fmt.Errorf("'%s' attribute is not a list", name)
	}
	result := make([]string, 0, len(listAttr.Value))
	for i, elem := range listAttr.Value {
		strElem, ok := elem.(*types.AttributeValueMemberS)
		if !ok {
			return nil, fmt.Errorf("'%s' element at index %d is not a string", name, i)
		}
		result = append(result, strElem.Value)
	}
	return result, nil
}

// toUnixSeconds truncates a time to second precision and returns the Unix timestamp.
// This ensures consistent key generation regardless of sub-second precision in the input.
func toUnixSeconds(t time.Time) int64 {
	return t.Truncate(time.Second).Unix()
}
