package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/jonsabados/saturdaysspinout/event"
	"github.com/jonsabados/saturdaysspinout/ingestion"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/metrics"
	sqsutil "github.com/jonsabados/saturdaysspinout/sqs"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/jonsabados/saturdaysspinout/ws"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
)

type appCfg struct {
	LogLevel                     string `envconfig:"LOG_LEVEL" required:"true"`
	DynamoDBTable                string `envconfig:"DYNAMODB_TABLE" required:"true"`
	SearchWindowInDays           int    `envconfig:"SEARCH_WINDOW_IN_DAYS" default:"10"`
	WSManagementEndpoint         string `envconfig:"WS_MANAGEMENT_ENDPOINT" required:"true"`
	RaceConsumptionConcurrency   int    `envconfig:"RACE_CONSUMPTION_CONCURRENCY" required:"true"`
	LapConsumptionConcurrency    int    `envconfig:"LAP_CONSUMPTION_CONCURRENCY" required:"true"`
	IngestionQueueURL            string `envconfig:"INGESTION_QUEUE_URL" required:"true"`
	IngestionLockDurationSeconds int    `envconfig:"INGESTION_LOCK_DURATION_SECONDS" required:"true"`
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
	pusher := ws.NewPusher(apiGWClient, driverStore)

	cwClient := cloudwatch.NewFromConfig(awsCfg)
	metricsClient := metrics.NewCloudWatchEmitter(cwClient, "SaturdaysSpinout")

	sqsClient := sqs.NewFromConfig(awsCfg)
	eventDispatcher := event.NewSQSEventDispatcher(sqsClient, cfg.IngestionQueueURL)

	iracingClient := iracing.NewClient(httpClient, metricsClient)

	lockDuration := time.Duration(cfg.IngestionLockDurationSeconds) * time.Second
	processor := ingestion.NewRaceProcessor(driverStore, iracingClient, pusher, eventDispatcher, lockDuration,
		ingestion.WithSearchWindowInDays(cfg.SearchWindowInDays),
		ingestion.WithRaceConsumptionConcurrency(cfg.RaceConsumptionConcurrency),
		ingestion.WithLapConsumptionConcurrency(cfg.LapConsumptionConcurrency),
	)

	handler := NewHandler(processor)
	handler = sqsutil.WithReducedContextDeadline(handler, time.Second*5)
	handler = sqsutil.WithVisibilityResetOnError(handler, sqsClient, sqsutil.LinearVisibilityTimeoutComputer(time.Second*2))
	handler = sqsutil.WithXRayCapture(handler, "ProcessIngestion")
	handler = sqsutil.WithPanicProtection(handler)
	handler = sqsutil.WithLogger(handler, logger)

	lambda.Start(handler)
}
