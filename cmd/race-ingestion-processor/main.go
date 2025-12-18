package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/jonsabados/saturdaysspinout/ws"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
)

type appCfg struct {
	LogLevel                   string `envconfig:"LOG_LEVEL" required:"true"`
	DynamoDBTable              string `envconfig:"DYNAMODB_TABLE" required:"true"`
	SearchWindowInDays         int    `envconfig:"SEARCH_WINDOW_IN_DAYS" default:"10"`
	WSManagementEndpoint       string `envconfig:"WS_MANAGEMENT_ENDPOINT" required:"true"`
	RaceConsumptionConcurrency int    `envconfig:"RACE_CONSUMPTION_CONCURRENCY" required:"true"`
	LapConsumptionConcurrency  int    `envconfig:"LAP_CONSUMPTION_CONCURRENCY" required:"true"`
}

func main() {
	ctx := context.Background()
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

	httpClient := xray.Client(http.DefaultClient)

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithHTTPClient(httpClient))
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading AWS config")
	}
	awsv2.AWSV2Instrumentor(&awsCfg.APIOptions)

	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	driverStore := store.NewDynamoStore(dynamoClient, cfg.DynamoDBTable)

	apiGWClient := apigatewaymanagementapi.NewFromConfig(awsCfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = &cfg.WSManagementEndpoint
	})
	pusher := ws.NewPusher(apiGWClient)

	iracingClient := iracing.NewClient(httpClient)

	processor := ingestion.NewRaceProcessor(driverStore, iracingClient, pusher,
		ingestion.WithSearchWindowInDays(cfg.SearchWindowInDays),
		ingestion.WithRaceConsumptionConcurrency(cfg.RaceConsumptionConcurrency),
		ingestion.WithLapConsumptionConcurrency(cfg.LapConsumptionConcurrency),
	)

	lambda.Start(func(ctx context.Context, event events.SQSEvent) error {
		ctx = logger.WithContext(ctx)
		log := zerolog.Ctx(ctx)

		for _, record := range event.Records {
			var msg ingestion.RaceIngestionRequest
			if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
				log.Error().Err(err).Str("messageId", record.MessageId).Msg("failed to parse message")
				continue
			}

			log.Info().Int64("driverId", msg.DriverID).Str("messageId", record.MessageId).Msg("processing race ingestion")

			err := xray.Capture(ctx, "IngestRaces", func(captureCtx context.Context) error {
				_ = xray.AddAnnotation(captureCtx, "driverID", msg.DriverID)
				return processor.IngestRaces(captureCtx, msg)
			})
			if err != nil {
				log.Error().Err(err).Int64("driverId", msg.DriverID).Msg("failed to ingest races")
				return err
			}
		}

		return nil
	})
}
