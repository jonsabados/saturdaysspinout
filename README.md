# Saturday's Spinout

A full-stack web application for race logging, built with Go, Vue 3, and deployed to AWS.

## Disclaimer

This is a toy project, meant to scratch an itch while also giving me time to play with the core elements of my craft 
without being distracted by things like mentoring, orchestrating and politicking. There are no delivery timelines, worries
about will this make sense & be maintainable to junior engineers, and so on. So there is lots of over-engineering, which feels
appropriate given the itch this is scratching is focused around an activity where we are driving simulated cars as fast
as we can just for shits and giggles.

Budget however is a real issue, and even though this architecture at its core would scale as far as your wallet allows
it should be incredibly inexpensive well beyond the point where the user base gets big enough to be problematic on its own.
So, the technologies in use are serverless and billed based on usage. Expectation is the primary cost center will be Route53 
fees, maybe a couple bucks a month in dynamo storage, and pocket change for compute (via Lambdas, all of which have very low 
reserved concurrency limits set as a safety).

## Architecture Overview

```mermaid
flowchart TB
    subgraph CloudFront["CloudFront CDN"]
        CF1["app.* (Frontend)"]
        CF2["api.* (API)"]
        CF3["www/root (Website)"]
        CF4["ws.* (WebSocket)"]
    end

    CF1 --> S3_SPA["S3 Bucket<br/>(Vue SPA)"]
    CF2 --> APIGW["API Gateway<br/>(REST)"]
    CF3 --> S3_Static["S3 Bucket<br/>(Static)"]
    CF4 --> WS_APIGW

    APIGW --> Lambda["API Lambda<br/>(Go)"]
    Lambda --> DynamoDB["DynamoDB"]
    Lambda --> KMS["KMS"]
    Lambda --> SQS["SQS<br/>(Race Ingestion)"]

    SQS --> IngestionLambda["Ingestion Lambda<br/>(Go)"]
    IngestionLambda --> DynamoDB
    IngestionLambda --> iRacingAPI["iRacing Data API"]

    WS_APIGW["API Gateway<br/>(WebSocket)"] --> WS_Lambda["WebSocket Lambda<br/>(Go)"]
    WS_Lambda --> DynamoDB
    WS_Lambda --> KMS["KMS"]
```

## Project Structure

```
.
├── .github/workflows/      # CI/CD pipeline (GitHub Actions)
├── api/                    # API endpoint handlers and HTTP setup
├── auth/                   # JWT creation and KMS-based signing/encryption
├── cmd/                    # Application entry points
│   ├── lambda-based-api/   # AWS Lambda handler (REST API)
│   ├── race-ingestion-lambda/ # SQS consumer for race data ingestion
│   ├── standalone-api/     # Local development server
│   └── websocket-lambda/   # WebSocket Lambda handler
├── correlation/            # Request correlation ID middleware
├── ingestion/              # Race data ingestion processing
├── iracing/                # iRacing API client and OAuth integration
├── store/                  # Data persistence layer (DynamoDB)
├── ws/                     # WebSocket handler package
├── frontend/               # Vue 3 SPA
├── terraform/              # Infrastructure as Code
├── website/                # Static marketing site
├── scripts/                # Build scripts
└── Makefile                # Build orchestration
```

## Backend (Go)

The API is built with [chi](https://github.com/go-chi/chi) and can run either as an AWS Lambda function or as a standalone HTTP server.

### Entry Points

| Mode | File | Description |
|------|------|-------------|
| REST Lambda | [`cmd/lambda-based-api/main.go`](cmd/lambda-based-api/main.go) | AWS Lambda handler using `aws-lambda-go-api-proxy` |
| Standalone | [`cmd/standalone-api/main.go`](cmd/standalone-api/main.go) | Standard `net/http` server for local development |
| WebSocket Lambda | [`cmd/websocket-lambda/main.go`](cmd/websocket-lambda/main.go) | WebSocket API Gateway handler for real-time connections |
| Race Ingestion Lambda | [`cmd/race-ingestion-lambda/main.go`](cmd/race-ingestion-lambda/main.go) | SQS consumer for async race data ingestion |

Both entry points share the same API setup via [`cmd/api.go`](cmd/api.go), which configures:
- Structured logging with [zerolog](https://github.com/rs/zerolog)
- AWS X-Ray tracing
- Environment-based configuration

### API Layer

| File | Purpose |
|------|---------|
| [`api/rest-api.go`](api/rest-api.go) | Router setup, middleware stack (CORS, logging, correlation IDs) |
| [`api/auth-middleware.go`](api/auth-middleware.go) | JWT authentication middleware |
| [`api/common-responses.go`](api/common-responses.go) | Shared response utilities |
| [`api/health/`](api/health/) | Health check endpoints (`GET /health/ping`) |
| [`api/auth/`](api/auth/) | Auth endpoints (`POST /auth/ir/callback`) |
| [`api/doc/`](api/doc/) | iRacing API doc proxy (`GET /doc/iracing-api/*`) |

### Authentication

The `auth/` package handles JWT creation with AWS KMS for signing and encryption. JWTs contain encrypted iRacing tokens, allowing the backend to make iRacing API calls on behalf of authenticated users.

| File | Purpose |
|------|---------|
| [`auth/service.go`](auth/service.go) | Auth service orchestrating OAuth callback flow |
| [`auth/jwt.go`](auth/jwt.go) | JWT creation with KMS signing and payload encryption |
| [`auth/kms.go`](auth/kms.go) | AWS KMS client wrappers for signing and encryption |

### iRacing Integration

The `iracing/` package provides OAuth and API client functionality for iRacing.

| File | Purpose |
|------|---------|
| [`iracing/oauth.go`](iracing/oauth.go) | OAuth token exchange with PKCE support |
| [`iracing/client.go`](iracing/client.go) | iRacing API client for user info and data retrieval |
| [`iracing/doc_client.go`](iracing/doc_client.go) | Proxy client for iRacing API documentation endpoints |

### Middleware

| File | Purpose |
|------|---------|
| [`correlation/correlation-id-middleware.go`](correlation/correlation-id-middleware.go) | Generates/propagates correlation IDs for request tracing |

### Data Store

The persistence layer uses DynamoDB with a single-table design. See the [DynamoDB Schema](https://docs.google.com/spreadsheets/d/180olt3Va13ixvT3XxMSK6MDvFg8pyDLWFzAqu48gUJM/edit?gid=0#gid=0) for the full table structure.

| File | Purpose |
|------|---------|
| [`store/dynamo_store.go`](store/dynamo_store.go) | DynamoDB client and CRUD operations |
| [`store/dynamo_models.go`](store/dynamo_models.go) | Attribute mapping between entities and DynamoDB items |
| [`store/entities.go`](store/entities.go) | Domain entity definitions |

### WebSocket

The `ws/` package handles real-time WebSocket connections via API Gateway WebSocket APIs.

| File | Purpose |
|------|---------|
| [`ws/handler.go`](ws/handler.go) | Main router - dispatches to route-specific handlers |
| [`ws/push.go`](ws/push.go) | `Pusher` abstraction for sending messages and managing connections |
| [`ws/auth/handler.go`](ws/auth/handler.go) | Authentication handler - validates JWT, stores connection |
| [`ws/ping/handler.go`](ws/ping/handler.go) | Heartbeat handler - verifies connection, responds with pong |

**Connection Flow:**
1. Client connects to `wss://ws.{domain}`
2. Client sends `{"action": "auth", "token": "<JWT>"}` to authenticate
3. Server validates JWT, stores connection mapping in DynamoDB
4. Client sends periodic `{"action": "pingRequest", "driverId": <id>}` for heartbeat
5. Connections have 24h TTL in DynamoDB for automatic cleanup

### Race Ingestion

The `ingestion/` package handles asynchronous ingestion of race history from the iRacing Data API. Processing is decoupled from the REST API via SQS.

| File | Purpose |
|------|---------|
| [`ingestion/race-processor.go`](ingestion/race-processor.go) | Fetches race results from iRacing and stores them |

**Ingestion Flow:**
1. REST API receives request at `POST /ingestion/race` with authenticated user
2. API enqueues message to SQS with driver ID and iRacing access token
3. Race Ingestion Lambda consumes message, queries iRacing `/data/results/search_series`
4. Results are filtered to races only (event_type=5) and stored in DynamoDB
5. Driver's `races_ingested_to` timestamp is updated for incremental sync

The iRacing search API returns chunked responses (results split across multiple S3 URLs). The client fetches all chunks and combines them. Search window is capped at 90 days per request.

## Frontend (Vue 3 + TypeScript)

A single-page application built with Vue 3, TypeScript, and Vite.

| File | Purpose |
|------|---------|
| [`frontend/src/main.ts`](frontend/src/main.ts) | Application bootstrap |
| [`frontend/src/App.vue`](frontend/src/App.vue) | Root component |
| [`frontend/src/router/index.ts`](frontend/src/router/index.ts) | Vue Router configuration |
| [`frontend/src/views/HomeView.vue`](frontend/src/views/HomeView.vue) | Home page |

## Infrastructure (Terraform)

All AWS infrastructure is defined in Terraform with workspace support for multiple environments.

| File | Purpose |
|------|---------|
| [`terraform/api.tf`](terraform/api.tf) | REST API Lambda, API Gateway, certificates, environment variables |
| [`terraform/race-ingestion.tf`](terraform/race-ingestion.tf) | SQS queue, Race Ingestion Lambda, event source mapping |
| [`terraform/websockets.tf`](terraform/websockets.tf) | WebSocket API Gateway, custom domain, routes |
| [`terraform/websockets-lambda.tf`](terraform/websockets-lambda.tf) | WebSocket Lambda function and IAM permissions |
| [`terraform/front-end.tf`](terraform/front-end.tf) | S3 bucket, CloudFront distribution for SPA |
| [`terraform/website.tf`](terraform/website.tf) | S3 bucket, CloudFront for static site |
| [`terraform/store.tf`](terraform/store.tf) | DynamoDB table (with TTL for WebSocket connections) |
| [`terraform/kms.tf`](terraform/kms.tf) | KMS keys for JWT signing and encryption |
| [`terraform/secrets.tf`](terraform/secrets.tf) | Secrets Manager references (iRacing credentials) |
| [`terraform/backend.tf`](terraform/backend.tf) | S3 backend for Terraform state |

## CI/CD

GitHub Actions runs on push and PR to `main`. The workflow ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)):

1. **Backend tests** - Runs Go tests with race detection and coverage against a DynamoDB Local service container
2. **Frontend tests** - Runs Vitest with coverage
3. **Build** - Builds the Lambda deployment package (only after tests pass)

Test results are published as GitHub check annotations and coverage is uploaded to Codecov.

## Development

### Prerequisites

- Make (included on Linux/macOS; Windows users can use [GnuWin32](http://gnuwin32.sourceforge.net/packages/make.htm) or WSL)
- Go 1.21+
- Node.js 18+
- Terraform 1.0+
- AWS CLI (configured)
- Docker (for local DynamoDB)

Run `make` or `make help` to see all available targets.

### Local Development

Run the backend API:
```bash
make run-rest-api
```

Run the frontend (in a separate terminal):
```bash
make run-frontend
```

The frontend dev server runs on `http://localhost:5173` and the API on `http://localhost:8080`.

#### Environment Variables from Terraform

The `make run-rest-api` target automatically sources environment variables from Terraform, ensuring local development uses the same configuration as the deployed Lambda. This is accomplished via the `app_env_vars` output:

```hcl
# In terraform/api.tf
locals {
  app_env_vars = {
    LOG_LEVEL                  = "info"
    CORS_ALLOWED_ORIGINS       = "..."
    IRACING_CREDENTIALS_SECRET = data.aws_secretsmanager_secret.iracing_credentials.arn
    JWT_SIGNING_KEY_ARN        = aws_kms_key.jwt.arn
    JWT_ENCRYPTION_KEY_ARN     = aws_kms_key.jwt_encryption.arn
  }
}

# Lambda uses the same map
resource "aws_lambda_function" "api_lambda" {
  environment {
    variables = local.app_env_vars
  }
}

# Output for local dev (formatted as KEY=VALUE pairs)
output "app_env_vars" {
  value = join(" ", [for k, v in local.app_env_vars : "${k}=${v}"])
}
```

The Makefile then uses this output:
```makefile
run-rest-api:
	env $(terraform -chdir=terraform output -raw app_env_vars) LOG_LEVEL=trace go run ...
```

This pattern ensures that any new environment variables added to the Lambda are automatically available during local development without manual synchronization.

### Testing

Tests use a local DynamoDB container. Manage it with:

```bash
make dynamo-start   # Start local DynamoDB (creates container if needed)
make dynamo-stop    # Stop the container (preserves data)
make dynamo-rm      # Stop and remove the container
make dynamo-status  # Check container status
```

Run tests:
```bash
go test ./...
```

### Building

```bash
# Build Lambda deployment package
make build

# Build frontend for deployment
make build-frontend
```

### Deploying

The frontend build sources the API URL from Terraform output (`terraform output -raw api_url`), so the build is workspace-specific. When switching Terraform workspaces, always rebuild the frontend before deploying:

```bash
# Switch workspace
cd terraform && terraform workspace select <workspace> && cd ..

# Clean and rebuild frontend for the new workspace
make clean
make build-frontend

# Deploy
make deploy-frontend
```

Other deploy commands:

```bash
# Deploy static website
make deploy-website
```

## Environment Variables

### Backend

These are managed in `terraform/api.tf` as `local.app_env_vars` and automatically provided to both Lambda and local development.

| Variable | Description |
|----------|-------------|
| `LOG_LEVEL` | Logging level (trace, debug, info, warn, error) |
| `CORS_ALLOWED_ORIGINS` | Comma-separated list of allowed origins |
| `IRACING_CREDENTIALS_SECRET` | ARN of Secrets Manager secret containing iRacing OAuth credentials |
| `JWT_SIGNING_KEY_ARN` | ARN of KMS key used to sign JWTs |
| `JWT_ENCRYPTION_KEY_ARN` | ARN of KMS key used to encrypt JWT payloads |

### Frontend

| Variable | Required | Description |
|----------|----------|-------------|
| `VITE_API_BASE_URL` | No | API base URL (defaults to `http://localhost:8080`) |
| `VITE_WS_BASE_URL` | No | WebSocket base URL (defaults to `ws://localhost:8081`) |
| `VITE_IRACING_CLIENT_ID` | Yes | iRacing OAuth client ID (see `frontend/.env.local.example`) |