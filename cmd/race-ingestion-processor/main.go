package main

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
)

type appCfg struct {
	LogLevel string `envconfig:"LOG_LEVEL" required:"true"`
}

type raceIngestionMessage struct {
	DriverID           int64  `json:"driverId"`
	IRacingAccessToken string `json:"IRacingAccessToken"`
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	logger.Info().Msg("starting race ingestion processor")

	var cfg appCfg
	err := envconfig.Process("", &cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading config")
	}

	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Fatal().Str("input", cfg.LogLevel).Err(err).Msg("error parsing log level")
	}
	logger = logger.Level(logLevel)

	err = xray.Configure(xray.Config{
		LogLevel: "warn",
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error configuring x-ray")
	}

	processor := ingestion.NewRaceProcessor()

	lambda.Start(func(ctx context.Context, event events.SQSEvent) error {
		ctx = logger.WithContext(ctx)
		log := zerolog.Ctx(ctx)

		for _, record := range event.Records {
			var msg raceIngestionMessage
			if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
				log.Error().Err(err).Str("messageId", record.MessageId).Msg("failed to parse message")
				continue
			}

			log.Info().Int64("driverId", msg.DriverID).Str("messageId", record.MessageId).Msg("processing race ingestion")

			if err := processor.IngestRaces(ctx, msg.DriverID, msg.IRacingAccessToken); err != nil {
				log.Error().Err(err).Int64("driverId", msg.DriverID).Msg("failed to ingest races")
				return err
			}
		}

		return nil
	})
}
