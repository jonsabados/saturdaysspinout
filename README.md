# Saturday's Spinout

A full-stack web application for race logging, built with Go, Vue 3, and deployed to AWS.

## Architecture Overview

```mermaid
flowchart TB
    subgraph CloudFront["CloudFront CDN"]
        CF1["app.* (Frontend)"]
        CF2["api.* (API)"]
        CF3["www/root (Website)"]
    end

    CF1 --> S3_SPA["S3 Bucket<br/>(Vue SPA)"]
    CF2 --> APIGW["API Gateway"]
    CF3 --> S3_Static["S3 Bucket<br/>(Static)"]

    APIGW --> Lambda["Lambda<br/>(Go)"]
    Lambda --> DynamoDB["DynamoDB"]
```

## Project Structure

```
.
├── .github/workflows/      # CI/CD pipeline (GitHub Actions)
├── api/                    # API endpoint handlers and HTTP setup
├── auth/                   # JWT creation and KMS-based signing/encryption
├── cmd/                    # Application entry points
│   ├── lambda-based-api/   # AWS Lambda handler
│   └── standalone-api/     # Local development server
├── correlation/            # Request correlation ID middleware
├── iracing/                # iRacing API client and OAuth integration
├── store/                  # Data persistence layer (DynamoDB)
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
| Lambda | [`cmd/lambda-based-api/main.go`](cmd/lambda-based-api/main.go) | AWS Lambda handler using `aws-lambda-go-api-proxy` |
| Standalone | [`cmd/standalone-api/main.go`](cmd/standalone-api/main.go) | Standard `net/http` server for local development |

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
| [`terraform/api.tf`](terraform/api.tf) | Lambda, API Gateway, certificates, environment variables |
| [`terraform/front-end.tf`](terraform/front-end.tf) | S3 bucket, CloudFront distribution for SPA |
| [`terraform/website.tf`](terraform/website.tf) | S3 bucket, CloudFront for static site |
| [`terraform/store.tf`](terraform/store.tf) | DynamoDB table |
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
| `VITE_IRACING_CLIENT_ID` | Yes | iRacing OAuth client ID (see `frontend/.env.local.example`) |