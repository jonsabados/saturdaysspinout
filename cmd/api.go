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
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/google/uuid"
	apiAuth "github.com/jonsabados/saturdaysspinout/api/auth"
	"github.com/jonsabados/saturdaysspinout/api/doc"
	"github.com/jonsabados/saturdaysspinout/api/health"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/iracing"
	"github.com/jonsabados/saturdaysspinout/store"
)

type appCfg struct {
	LogLevel                 string   `envconfig:"LOG_LEVEL" required:"true"`
	CORSAllowedOrigins       []string `envconfig:"CORS_ALLOWED_ORIGINS" required:"true"`
	IRacingCredentialsSecret string   `envconfig:"IRACING_CREDENTIALS_SECRET" required:"true"`
	JWTSigningKeyARN         string   `envconfig:"JWT_SIGNING_KEY_ARN" required:"true"`
	JWTEncryptionKeyARN      string   `envconfig:"JWT_ENCRYPTION_KEY_ARN" required:"true"`
	DynamoDBTable            string   `envconfig:"DYNAMODB_TABLE" required:"true"`
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

	kmsClient := kms.NewFromConfig(awsCfg)
	awsKMSClient := auth.NewAWSKMSClient(kmsClient)
	jwtSigner := auth.NewKMSSignerAdapter(awsKMSClient, cfg.JWTSigningKeyARN)
	jwtEncryptor := auth.NewKMSEncryptorAdapter(awsKMSClient, cfg.JWTEncryptionKeyARN)

	jwtService := auth.NewJWTService(jwtSigner, jwtEncryptor, uuid.NewString, "saturdaysspinout", 24*time.Hour)

	dynamoClient := dynamodb.NewFromConfig(awsCfg)
	driverStore := store.NewDynamoStore(dynamoClient, cfg.DynamoDBTable)

	iRacingOAuthClient := iracing.NewOAuthClient(httpClient, iRacingCreds.OauthClientID, iRacingCreds.OauthClientSecret)
	iRacingClient := iracing.NewClient(httpClient)

	authService := auth.NewService(iRacingOAuthClient, jwtService, iRacingClient, driverStore)

	authMiddleware := api.AuthMiddleware(jwtService)

	routers := api.RootRouters{
		HealthRouter: health.NewRouter(),
		AuthRouter:   apiAuth.NewRouter(authService, authMiddleware),
		DocRouter:    doc.NewRouter(iracing.NewDocClient(httpClient), authMiddleware),
	}

	return api.NewRestAPI(logger, uuid.NewString, cfg.CORSAllowedOrigins, routers)
}
