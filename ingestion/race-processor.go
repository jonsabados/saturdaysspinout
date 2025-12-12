package ingestion

import "context"

type RaceProcessor struct {
}

func NewRaceProcessor() *RaceProcessor {
	return &RaceProcessor{}
}

func (r *RaceProcessor) IngestRaces(ctx context.Context, driverID int64, iRacingAccessToken string) error {
	return nil
}
