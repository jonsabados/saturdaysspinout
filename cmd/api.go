package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/google/uuid"
	apiAuth "github.com/jonsabados/saturdaysspinout/api/auth"
	apiCars "github.com/jonsabados/saturdaysspinout/api/cars"
	"github.com/jonsabados/saturdaysspinout/api/developer"
	"github.com/jonsabados/saturdaysspinout/api/driver"
	"github.com/jonsabados/saturdaysspinout/api/health"
	"github.com/jonsabados/saturdaysspinout/api/ingestion"
	apiTracks "github.com/jonsabados/saturdaysspinout/api/tracks"
	"github.com/jonsabados/saturdaysspinout/cars"
	"github.com/jonsabados/saturdaysspinout/event"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/metrics"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/jonsabados/saturdaysspinout/tracks"
)

type appCfg struct {
	LogLevel                 string   `envconfig:"LOG_LEVEL" required:"true"`
	CORSAllowedOrigins       []string `envconfig:"CORS_ALLOWED_ORIGINS" required:"true"`
	IRacingCredentialsSecret string   `envconfig:"IRACING_CREDENTIALS_SECRET" required:"true"`
	JWTSigningKeySecret      string   `envconfig:"JWT_SIGNING_KEY_SECRET" required:"true"`
	JWTEncryptionKeySecret   string   `envconfig:"JWT_ENCRYPTION_KEY_SECRET" required:"true"`
	DynamoDBTable            string   `envconfig:"DYNAMODB_TABLE" required:"true"`
	RaceIngestionQueueURL    string   `envconfig:"RACE_INGESTION_QUEUE_URL" required:"true"`
	IRacingCacheBucket       string   `envconfig:"IRACING_CACHE_BUCKET" required:"true"`
}

type iRacingCredentials struct {
	OauthClientID     string `json:"oauth_client_id"`
	OauthClientSecret string `json:"oauth_client_secret"`
}

func CreateAPI() http.Handler {
	ctx := context.Background()
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	logger.Info().Msg("starting rest API")

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

	// get x-ray goodness going with the http client we will be using
	httpClient := xray.Client(http.DefaultClient)

	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithHTTPClient(httpClient))
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading default config")
	}

	// add x-ray instrumentation to all the AWS clients
	awsv2.AWSV2Instrumentor(&awsCfg.APIOptions)

	secretsClient := secretsmanager.NewFromConfig(awsCfg)
	secretResult, err := secretsClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &cfg.IRacingCredentialsSecret,
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error fetching iRacing credentials from secrets manager")
	}

	var iRacingCreds iRacingCredentials
	err = json.Unmarshal([]byte(*secretResult.SecretString), &iRacingCreds)
	if err != nil {
		logger.Fatal().Err(err).Msg("error parsing iRacing credentials")
	}

	secretHash := sha256.Sum256([]byte(iRacingCreds.OauthClientSecret))
	logger.Info().Str("oauth_client_id", iRacingCreds.OauthClientID).Str("oauth_client_secret_sha256", hex.EncodeToString(secretHash[:])).Msg("loaded iRacing OAuth credentials")

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
	driverStore := store.NewDynamoStore(dynamoClient, cfg.DynamoDBTable)

	cwClient := cloudwatch.NewFromConfig(awsCfg)
	metricsClient := metrics.NewCloudWatchEmitter(cwClient, "SaturdaysSpinout")

	iRacingOAuthClient := iracing.NewOAuthClient(httpClient, iRacingCreds.OauthClientID, iRacingCreds.OauthClientSecret)
	iRacingClient := iracing.NewClient(httpClient, metricsClient)

	s3Client := s3.NewFromConfig(awsCfg)
	cachingClient := iracing.NewGlobalInfoCachingClient(iRacingClient, s3Client, cfg.IRacingCacheBucket, 24*time.Hour)

	sqsClient := sqs.NewFromConfig(awsCfg)
	raceIngestionDispatcher := event.NewSQSEventDispatcher(sqsClient, cfg.RaceIngestionQueueURL)

	authService := auth.NewService(iRacingOAuthClient, jwtService, iRacingClient, driverStore)
	tracksService := tracks.NewService(cachingClient)
	carsService := cars.NewService(cachingClient)

	authMiddleware := api.AuthMiddleware(jwtService)
	developerMiddleware := api.EntitlementMiddleware("developer")

	routers := api.RootRouters{
		HealthRouter:    health.NewRouter(),
		AuthRouter:      apiAuth.NewRouter(authService, authMiddleware),
		DeveloperRouter: developer.NewRouter(iracing.NewDocClient(httpClient), authMiddleware, developerMiddleware),
		IngestionRouter: ingestion.NewRouter(driverStore, raceIngestionDispatcher, authMiddleware),
		DriverRouter:    driver.NewRouter(driverStore, authMiddleware, developerMiddleware),
		TracksRouter:    apiTracks.NewRouter(tracksService, authMiddleware),
		CarsRouter:      apiCars.NewRouter(carsService, authMiddleware),
	}

	apiCfg := api.RestAPIConfig{
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		DeadlineBuffer:     250 * time.Millisecond,
	}

	return api.NewRestAPI(logger, uuid.NewString, routers, apiCfg)
}
