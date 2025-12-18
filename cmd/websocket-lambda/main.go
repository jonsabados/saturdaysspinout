package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/google/uuid"
	wsauth "github.com/jonsabados/saturdaysspinout/ws/auth"
	"github.com/jonsabados/saturdaysspinout/ws/disconnect"
	"github.com/jonsabados/saturdaysspinout/ws/ping"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/jonsabados/saturdaysspinout/ws"
)

type appCfg struct {
	LogLevel               string `envconfig:"LOG_LEVEL" required:"true"`
	JWTSigningKeySecret    string `envconfig:"JWT_SIGNING_KEY_SECRET" required:"true"`
	JWTEncryptionKeySecret string `envconfig:"JWT_ENCRYPTION_KEY_SECRET" required:"true"`
	DynamoDBTable          string `envconfig:"DYNAMODB_TABLE" required:"true"`
	WSManagementEndpoint   string `envconfig:"WS_MANAGEMENT_ENDPOINT" required:"true"`
}

func main() {
	ctx := context.Background()
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	logger.Info().Msg("starting websocket handler")

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
		logger.Fatal().Err(err).Msg("error loading default config")
	}

	awsv2.AWSV2Instrumentor(&awsCfg.APIOptions)

	secretsClient := secretsmanager.NewFromConfig(awsCfg)

	signingKeyResult, err := secretsClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &cfg.JWTSigningKeySecret,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error fetching JWT signing key from secrets manager")
	}

	signingKey, err := auth.ParseSigningKeyPEM([]byte(*signingKeyResult.SecretString))
	if err != nil {
		logger.Fatal().Err(err).Msg("error parsing JWT signing key")
	}
	logger.Info().Msg("loaded JWT signing key")

	encryptionKeyResult, err := secretsClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &cfg.JWTEncryptionKeySecret,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error fetching JWT encryption key from secrets manager")
	}

	encryptionKey, err := auth.ParseEncryptionKeyBase64(*encryptionKeyResult.SecretString)
	if err != nil {
		logger.Fatal().Err(err).Msg("error parsing JWT encryption key")
	}
	logger.Info().Msg("loaded JWT encryption key")

	jwtService, err := auth.NewJWTService(signingKey, encryptionKey, uuid.NewString, "saturdaysspinout", 24*time.Hour)
	if err != nil {
		logger.Fatal().Err(err).Msg("error creating JWT service")
	}

	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	connStore := store.NewDynamoStore(dynamoClient, cfg.DynamoDBTable)

	apiClient := apigatewaymanagementapi.NewFromConfig(awsCfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = &cfg.WSManagementEndpoint
	})

	pusher := ws.NewPusher(apiClient, connStore)
	disconnectHandler := disconnect.NewHandler(connStore)
	authHandler := wsauth.NewHandler(jwtService, pusher, connStore)
	pingHandler := ping.NewHandler(pusher, connStore)

	handler := ws.NewHandler(disconnectHandler, authHandler, pingHandler)

	lambda.Start(func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		ctx = logger.WithContext(ctx)

		return handler.Handle(ctx, request)
	})
}
