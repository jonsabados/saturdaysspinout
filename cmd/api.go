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
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"

	"github.com/jonsabados/saturdaysspinout/api"
)

type appCfg struct {
	LogLevel                 string   `envconfig:"LOG_LEVEL" required:"true"`
	CORSAllowedOrigins       []string `envconfig:"CORS_ALLOWED_ORIGINS" required:"true"`
	IRacingCredentialsSecret string   `envconfig:"IRACING_CREDENTIALS_SECRET" required:"true"`
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

	// load our config from environmental variables
	var cfg appCfg
	err := envconfig.Process("", &cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading config")
	}

	// set the log level per our config
	logLevel, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.Fatal().Str("input", cfg.LogLevel).Err(err).Msg("error parsing log level")
	}
	logger = logger.Level(logLevel)

	// initialize x-ray
	err = xray.Configure(xray.Config{
		LogLevel: "warn",
	})
	if err != nil {
		logger.Fatal().Err(err).Msg("error configuring x-ray")
	}

	// get x-ray goodness going with the http client we will be using
	httpClient := xray.Client(http.DefaultClient)

	// get our AWS environment setup
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithHTTPClient(httpClient))
	if err != nil {
		logger.Fatal().Err(err).Msg("error loading default config")
	}

	// add x-ray instrumentation to all the AWS clients
	awsv2.AWSV2Instrumentor(&awsCfg.APIOptions)

	// fetch iRacing OAuth credentials from secrets manager
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

	pingEndpoint := api.NewPingEndpoint()
	return api.NewRestAPI(logger, uuid.NewString, cfg.CORSAllowedOrigins, pingEndpoint)
}
