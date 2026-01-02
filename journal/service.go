package journal

import (
	"context"
	"strings"
	"time"

	"github.com/jonsabados/saturdaysspinout/metrics"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

// MetricsEmitter defines the metrics interface needed by the journal service.
type MetricsEmitter interface {
	EmitCount(ctx context.Context, name string, count int) error
}

// Known tag prefixes with constrained values
var knownTagPrefixes = map[string][]string{
	"sentiment": {"good", "neutral", "bad"},
}

// FieldValidation represents a validation failure for a specific field.
type FieldValidation struct {
	Field  string
	Code   string
	Params map[string]string
}

// ValidateTags checks that tags with known prefixes have valid values.
// Returns validation errors for any invalid tags.
func ValidateTags(tags []string) []FieldValidation {
	var errors []FieldValidation
	for _, tag := range tags {
		if idx := strings.Index(tag, ":"); idx != -1 {
			prefix := tag[:idx]
			value := tag[idx+1:]

			if allowedValues, ok := knownTagPrefixes[prefix]; ok {
				valid := false
				for _, allowed := range allowedValues {
					if value == allowed {
						valid = true
						break
					}
				}
				if !valid {
					errors = append(errors, FieldValidation{
						Field: "tags",
						Code:  "invalid_tag_value",
						Params: map[string]string{
							"prefix":  prefix,
							"value":   value,
							"allowed": strings.Join(allowedValues, ","),
						},
					})
				}
			}
		}
	}
	return errors
}

// Entry represents a journal entry with joined race context.
type Entry struct {
	RaceID    int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Notes     string
	Tags      []string
	Race      *store.DriverSession
}

// Store defines the data access methods needed by the journal service.
type Store interface {
	GetDriverSession(ctx context.Context, driverID int64, startTime time.Time) (*store.DriverSession, error)
	GetDriverSessions(ctx context.Context, driverID int64, startTimes []time.Time) ([]store.DriverSession, error)
	SaveJournalEntry(ctx context.Context, entry store.RaceJournalEntry) error
	GetJournalEntry(ctx context.Context, driverID, raceID int64) (*store.RaceJournalEntry, error)
	GetJournalEntries(ctx context.Context, driverID int64, from, to time.Time) ([]store.RaceJournalEntry, error)
	DeleteJournalEntry(ctx context.Context, driverID, raceID int64) error
}

// Service provides business logic for race journal operations.
type Service struct {
	store   Store
	metrics MetricsEmitter
}

// NewService creates a new journal service.
func NewService(store Store, metrics MetricsEmitter) *Service {
	return &Service{store: store, metrics: metrics}
}

// ValidateRaceExists checks if a race exists for the given driver.
// Returns true if the race exists, false if not. Error is only for infrastructure failures.
func (s *Service) ValidateRaceExists(ctx context.Context, driverID, raceID int64) (bool, error) {
	session, err := s.store.GetDriverSession(ctx, driverID, store.TimeFromDriverRaceID(raceID))
	if err != nil {
		return false, err
	}
	return session != nil, nil
}

// SaveInput contains the data needed to save a journal entry.
type SaveInput struct {
	DriverID int64
	RaceID   int64
	Notes    string
	Tags     []string
}

// Save creates or updates a journal entry. Returns the saved entry with race context.
// Callers should validate input with ValidateTags and ValidateRaceExists before calling Save.
func (s *Service) Save(ctx context.Context, input SaveInput) (*Entry, error) {
	// Save the entry
	entry := store.RaceJournalEntry{
		DriverID: input.DriverID,
		RaceID:   input.RaceID,
		Notes:    input.Notes,
		Tags:     input.Tags,
	}

	if err := s.store.SaveJournalEntry(ctx, entry); err != nil {
		return nil, err
	}

	// Emit metric (includes both creates and updates for simplicity)
	if err := s.metrics.EmitCount(ctx, metrics.JournalEntriesCreated, 1); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to emit journal entry metric")
	}

	// Fetch the saved entry to get timestamps and race context
	return s.Get(ctx, input.DriverID, input.RaceID)
}

// Get retrieves a single journal entry with its race context.
// Returns nil if the entry doesn't exist. Error is only for infrastructure failures.
func (s *Service) Get(ctx context.Context, driverID, raceID int64) (*Entry, error) {
	entry, err := s.store.GetJournalEntry(ctx, driverID, raceID)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	// Fetch the associated race session
	session, err := s.store.GetDriverSession(ctx, driverID, store.TimeFromDriverRaceID(raceID))
	if err != nil {
		return nil, err
	}

	return &Entry{
		RaceID:    entry.RaceID,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
		Notes:     entry.Notes,
		Tags:      normalizeTags(entry.Tags),
		Race:      session,
	}, nil
}

// ListInput contains parameters for listing journal entries.
type ListInput struct {
	DriverID int64
	From     time.Time
	To       time.Time
}

// List retrieves journal entries within a time range, joined with race context.
func (s *Service) List(ctx context.Context, input ListInput) ([]Entry, error) {
	entries, err := s.store.GetJournalEntries(ctx, input.DriverID, input.From, input.To)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return []Entry{}, nil
	}

	// Collect race start times for batch fetch
	startTimes := make([]time.Time, len(entries))
	for i, entry := range entries {
		startTimes[i] = store.TimeFromDriverRaceID(entry.RaceID)
	}

	// Batch fetch all sessions
	sessions, err := s.store.GetDriverSessions(ctx, input.DriverID, startTimes)
	if err != nil {
		return nil, err
	}

	// Build a lookup map by start time
	sessionMap := make(map[int64]*store.DriverSession, len(sessions))
	for i := range sessions {
		sessionMap[sessions[i].StartTime.Unix()] = &sessions[i]
	}

	// Join entries with sessions
	results := make([]Entry, len(entries))
	for i, entry := range entries {
		results[i] = Entry{
			RaceID:    entry.RaceID,
			CreatedAt: entry.CreatedAt,
			UpdatedAt: entry.UpdatedAt,
			Notes:     entry.Notes,
			Tags:      normalizeTags(entry.Tags),
			Race:      sessionMap[entry.RaceID],
		}
	}

	return results, nil
}

// Delete removes a journal entry. Idempotent - succeeds even if entry doesn't exist.
func (s *Service) Delete(ctx context.Context, driverID, raceID int64) error {
	return s.store.DeleteJournalEntry(ctx, driverID, raceID)
}

// normalizeTags ensures tags is never nil (returns empty slice instead).
func normalizeTags(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}