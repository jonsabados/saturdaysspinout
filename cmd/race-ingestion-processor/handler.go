package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/jonsabados/saturdaysspinout/sqs"
	"github.com/rs/zerolog"
)

type Processor interface {
	IngestRaces(ctx context.Context, request ingestion.RaceIngestionRequest) error
}

func NewHandler(processor Processor) sqs.HandlerFunc {
	return func(ctx context.Context, event events.SQSEvent) error {
		log := zerolog.Ctx(ctx)

		for _, record := range event.Records {
			var msg ingestion.RaceIngestionRequest
			if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
				log.Error().Err(err).Str("messageId", record.MessageId).Msg("failed to parse message")
				continue
			}

			log.Info().Int64("driverId", msg.DriverID).Str("messageId", record.MessageId).Msg("processing race ingestion")

			if err := processor.IngestRaces(ctx, msg); err != nil {
				log.Error().Err(err).Int64("driverId", msg.DriverID).Msg("failed to ingest races")
				return err
			}
		}

		return nil
	}
}
