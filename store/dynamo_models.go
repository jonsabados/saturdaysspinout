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
const wsConnectionSortKeyFormat = "ws#%s"
const ingestionLockSortKey = "ingestion_lock"

const websocketPartitionFormat = "websocket#%s"

const driverSessionSortKeyFormat = "session#%d" // timestamp for ordering

const sessionPartitionKeyFormat = "session#%d"
const sessionCarClassSortKeyFormat = "car_class#%d"
const sessionCarClassCarSortKeyFormat = "car_class#%d#car#%d"
const sessionDriverSortKeyFormat = "drivers#driver#%d"
const sessionDriverLapSortKeyFormat = "laps#driver#%d#lap#%d"

const globalCountersPartitionKey = "global"
const globalCountersSortKey = "counters"
const globalCountersAttributeDrivers = "drivers"
const globalCountersAttributeTracks = "tracks"
const globalCountersAttributeNotes = "notes"
const globalCountersAttributeSessions = "sessions"
const globalCountersAttributeLaps = "laps"

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
	id, err := getInt64Attr(item, "id")
	if err != nil {
		return nil, err
	}
	name, err := getStringAttr(item, "name")
	if err != nil {
		return nil, err
	}

	return &Track{
		ID:   id,
		Name: name,
	}, nil
}

func globalCountersFromAttributeMap(item map[string]types.AttributeValue) (*GlobalCounters, error) {
	counters := &GlobalCounters{}

	if driversAttr, ok := item[globalCountersAttributeDrivers].(*types.AttributeValueMemberN); ok {
		drivers, err := strconv.ParseInt(driversAttr.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid 'drivers' value: %w", err)
		}
		counters.Drivers = drivers
	}

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

	if sessionsAttr, ok := item[globalCountersAttributeSessions].(*types.AttributeValueMemberN); ok {
		sessions, err := strconv.ParseInt(sessionsAttr.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid 'sessions' value: %w", err)
		}
		counters.Sessions = sessions
	}

	if lapsAttr, ok := item[globalCountersAttributeLaps].(*types.AttributeValueMemberN); ok {
		laps, err := strconv.ParseInt(lapsAttr.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid 'laps' value: %w", err)
		}
		counters.Laps = laps
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
	timestamp, err := getInt64Attr(item, "timestamp")
	if err != nil {
		return nil, err
	}
	sessionID, err := getInt64Attr(item, "session_id")
	if err != nil {
		return nil, err
	}
	lapNumber, err := getInt64Attr(item, "lap_number")
	if err != nil {
		return nil, err
	}
	isMistake, err := getBoolAttr(item, "is_mistake")
	if err != nil {
		return nil, err
	}
	category, err := getStringAttr(item, "category")
	if err != nil {
		return nil, err
	}
	notes, err := getStringAttr(item, "notes")
	if err != nil {
		return nil, err
	}

	return &DriverNote{
		DriverID:  driverID,
		Timestamp: time.Unix(timestamp, 0),
		SessionID: sessionID,
		LapNumber: lapNumber,
		IsMistake: isMistake,
		Category:  category,
		Notes:     notes,
	}, nil
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
	reasonOut             string
}

func (d driverSessionModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName:           &types.AttributeValueMemberS{Value: fmt.Sprintf(driverPartitionFormat, d.driverID)},
		sortKeyName:                &types.AttributeValueMemberS{Value: fmt.Sprintf(driverSessionSortKeyFormat, d.startTime)},
		"subsession_id":            &types.AttributeValueMemberN{Value: strconv.FormatInt(d.subsessionID, 10)},
		"track_id":                 &types.AttributeValueMemberN{Value: strconv.FormatInt(d.trackID, 10)},
		"car_id":                   &types.AttributeValueMemberN{Value: strconv.FormatInt(d.carID, 10)},
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
	reasonOut, err := getStringAttr(item, "reason_out")
	if err != nil {
		return nil, err
	}

	return &DriverSession{
		DriverID:              driverID,
		SubsessionID:          subsessionID,
		TrackID:               trackID,
		CarID:                 carID,
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
		ReasonOut:             reasonOut,
	}, nil
}

// sessionModel represents race metadata (session#<id> / info)
type sessionModel struct {
	subsessionID int64
	trackID      int64
	startTime    int64
}

func (s sessionModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionPartitionKeyFormat, s.subsessionID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: defaultSortKey},
		"subsession_id":  &types.AttributeValueMemberN{Value: strconv.FormatInt(s.subsessionID, 10)},
		"track_id":       &types.AttributeValueMemberN{Value: strconv.FormatInt(s.trackID, 10)},
		"start_time":     &types.AttributeValueMemberN{Value: strconv.FormatInt(s.startTime, 10)},
	}
}

func sessionFromAttributeMap(item map[string]types.AttributeValue) (*Session, error) {
	subsessionID, err := getInt64Attr(item, "subsession_id")
	if err != nil {
		return nil, err
	}
	trackID, err := getInt64Attr(item, "track_id")
	if err != nil {
		return nil, err
	}
	startTime, err := getInt64Attr(item, "start_time")
	if err != nil {
		return nil, err
	}

	return &Session{
		SubsessionID: subsessionID,
		TrackID:      trackID,
		StartTime:    time.Unix(startTime, 0),
	}, nil
}

// sessionCarClassModel represents a car class in a session (session#<id> / car_class#<id>)
type sessionCarClassModel struct {
	subsessionID    int64
	carClassID      int64
	strengthOfField int
	numberOfEntries int
}

func (c sessionCarClassModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName:    &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionPartitionKeyFormat, c.subsessionID)},
		sortKeyName:         &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionCarClassSortKeyFormat, c.carClassID)},
		"subsession_id":     &types.AttributeValueMemberN{Value: strconv.FormatInt(c.subsessionID, 10)},
		"car_class_id":      &types.AttributeValueMemberN{Value: strconv.FormatInt(c.carClassID, 10)},
		"strength_of_field": &types.AttributeValueMemberN{Value: strconv.Itoa(c.strengthOfField)},
		"num_entries":       &types.AttributeValueMemberN{Value: strconv.Itoa(c.numberOfEntries)},
	}
}

// sessionCarClassCarModel represents a car within a class in a session (session#<id> / car_class#<id>#car#<id>)
type sessionCarClassCarModel struct {
	subsessionID int64
	carClassID   int64
	carID        int64
}

func (c sessionCarClassCarModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionPartitionKeyFormat, c.subsessionID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionCarClassCarSortKeyFormat, c.carClassID, c.carID)},
		"subsession_id":  &types.AttributeValueMemberN{Value: strconv.FormatInt(c.subsessionID, 10)},
		"car_class_id":   &types.AttributeValueMemberN{Value: strconv.FormatInt(c.carClassID, 10)},
		"car_id":         &types.AttributeValueMemberN{Value: strconv.FormatInt(c.carID, 10)},
	}
}

// sessionDriverModel represents a driver's result in a session (session#<id> / drivers#driver#<id>)
type sessionDriverModel struct {
	subsessionID          int64
	driverID              int64
	carID                 int64
	startPosition         int
	startPositionInClass  int
	finishPosition        int
	finishPositionInClass int
	incidents             int
	oldCPI                float64
	newCPI                float64
	oldIRating            int
	newIRating            int
	reasonOut             string
	ai                    bool
}

func (d sessionDriverModel) toAttributeMap() map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		partitionKeyName:           &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionPartitionKeyFormat, d.subsessionID)},
		sortKeyName:                &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionDriverSortKeyFormat, d.driverID)},
		"subsession_id":            &types.AttributeValueMemberN{Value: strconv.FormatInt(d.subsessionID, 10)},
		"driver_id":                &types.AttributeValueMemberN{Value: strconv.FormatInt(d.driverID, 10)},
		"car_id":                   &types.AttributeValueMemberN{Value: strconv.FormatInt(d.carID, 10)},
		"start_position":           &types.AttributeValueMemberN{Value: strconv.Itoa(d.startPosition)},
		"start_position_in_class":  &types.AttributeValueMemberN{Value: strconv.Itoa(d.startPositionInClass)},
		"finish_position":          &types.AttributeValueMemberN{Value: strconv.Itoa(d.finishPosition)},
		"finish_position_in_class": &types.AttributeValueMemberN{Value: strconv.Itoa(d.finishPositionInClass)},
		"incidents":                &types.AttributeValueMemberN{Value: strconv.Itoa(d.incidents)},
		"old_cpi":                  &types.AttributeValueMemberN{Value: strconv.FormatFloat(d.oldCPI, 'f', -1, 64)},
		"new_cpi":                  &types.AttributeValueMemberN{Value: strconv.FormatFloat(d.newCPI, 'f', -1, 64)},
		"old_irating":              &types.AttributeValueMemberN{Value: strconv.Itoa(d.oldIRating)},
		"new_irating":              &types.AttributeValueMemberN{Value: strconv.Itoa(d.newIRating)},
		"reason_out":               &types.AttributeValueMemberS{Value: d.reasonOut},
		"ai":                       &types.AttributeValueMemberBOOL{Value: d.ai},
	}
}

func sessionDriverFromAttributeMap(item map[string]types.AttributeValue) (*SessionDriver, error) {
	subsessionID, err := getInt64Attr(item, "subsession_id")
	if err != nil {
		return nil, err
	}
	driverID, err := getInt64Attr(item, "driver_id")
	if err != nil {
		return nil, err
	}
	carID, err := getInt64Attr(item, "car_id")
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
	reasonOut, err := getStringAttr(item, "reason_out")
	if err != nil {
		return nil, err
	}
	ai, err := getBoolAttr(item, "ai")
	if err != nil {
		return nil, err
	}

	return &SessionDriver{
		SubsessionID:          subsessionID,
		DriverID:              driverID,
		CarID:                 carID,
		StartPosition:         startPosition,
		StartPositionInClass:  startPositionInClass,
		FinishPosition:        finishPosition,
		FinishPositionInClass: finishPositionInClass,
		Incidents:             incidents,
		OldCPI:                oldCPI,
		NewCPI:                newCPI,
		OldIRating:            oldIRating,
		NewIRating:            newIRating,
		ReasonOut:             reasonOut,
		AI:                    ai,
	}, nil
}

// sessionDriverLapModel represents a lap for a driver in a session (session#<id> / laps#driver#<id>#lap#<lap>)
type sessionDriverLapModel struct {
	subsessionID int64
	driverID     int64
	lapNumber    int
	lapTime      int64 // stored as nanoseconds
	flags        int
	incident     bool
	lapEvents    []string
}

func (l sessionDriverLapModel) toAttributeMap() map[string]types.AttributeValue {
	m := map[string]types.AttributeValue{
		partitionKeyName: &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionPartitionKeyFormat, l.subsessionID)},
		sortKeyName:      &types.AttributeValueMemberS{Value: fmt.Sprintf(sessionDriverLapSortKeyFormat, l.driverID, l.lapNumber)},
		"subsession_id":  &types.AttributeValueMemberN{Value: strconv.FormatInt(l.subsessionID, 10)},
		"driver_id":      &types.AttributeValueMemberN{Value: strconv.FormatInt(l.driverID, 10)},
		"lap_number":     &types.AttributeValueMemberN{Value: strconv.Itoa(l.lapNumber)},
		"lap_time":       &types.AttributeValueMemberN{Value: strconv.FormatInt(l.lapTime, 10)},
		"flags":          &types.AttributeValueMemberN{Value: strconv.Itoa(l.flags)},
		"incident":       &types.AttributeValueMemberBOOL{Value: l.incident},
	}
	if len(l.lapEvents) > 0 {
		events := make([]types.AttributeValue, len(l.lapEvents))
		for i, e := range l.lapEvents {
			events[i] = &types.AttributeValueMemberS{Value: e}
		}
		m["lap_events"] = &types.AttributeValueMemberL{Value: events}
	}
	return m
}

func sessionDriverLapFromAttributeMap(item map[string]types.AttributeValue) (*SessionDriverLap, error) {
	subsessionID, err := getInt64Attr(item, "subsession_id")
	if err != nil {
		return nil, err
	}
	driverID, err := getInt64Attr(item, "driver_id")
	if err != nil {
		return nil, err
	}
	lapNumber, err := getIntAttr(item, "lap_number")
	if err != nil {
		return nil, err
	}
	lapTime, err := getInt64Attr(item, "lap_time")
	if err != nil {
		return nil, err
	}
	flags, err := getIntAttr(item, "flags")
	if err != nil {
		return nil, err
	}
	incident, err := getBoolAttr(item, "incident")
	if err != nil {
		return nil, err
	}

	var lapEvents []string
	if eventsAttr, ok := item["lap_events"].(*types.AttributeValueMemberL); ok {
		for _, e := range eventsAttr.Value {
			if s, ok := e.(*types.AttributeValueMemberS); ok {
				lapEvents = append(lapEvents, s.Value)
			}
		}
	}

	return &SessionDriverLap{
		SubsessionID: subsessionID,
		DriverID:     driverID,
		LapNumber:    lapNumber,
		LapTime:      time.Duration(lapTime),
		Flags:        flags,
		Incident:     incident,
		LapEvents:    lapEvents,
	}, nil
}

func sessionCarClassFromAttributeMap(subsessionID int64, item map[string]types.AttributeValue) (*SessionCarClass, error) {
	carClassID, err := getInt64Attr(item, "car_class_id")
	if err != nil {
		return nil, err
	}
	strengthOfField, err := getIntAttr(item, "strength_of_field")
	if err != nil {
		return nil, err
	}
	numberOfEntries, err := getIntAttr(item, "num_entries")
	if err != nil {
		return nil, err
	}

	return &SessionCarClass{
		SubsessionID:    subsessionID,
		CarClassID:      carClassID,
		StrengthOfField: strengthOfField,
		NumberOfEntries: numberOfEntries,
	}, nil
}

func sessionCarClassCarFromAttributeMap(subsessionID int64, item map[string]types.AttributeValue) (*SessionCarClassCar, error) {
	carClassID, err := getInt64Attr(item, "car_class_id")
	if err != nil {
		return nil, err
	}
	carID, err := getInt64Attr(item, "car_id")
	if err != nil {
		return nil, err
	}

	return &SessionCarClassCar{
		SubsessionID: subsessionID,
		CarClassID:   carClassID,
		CarID:        carID,
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
