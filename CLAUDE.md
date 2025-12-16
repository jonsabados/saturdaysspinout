# Claude Code Instructions

## Project Philosophy
- Toy project: no delivery timelines, over-engineering is welcome
- Craft-focused: exploring techniques without org overhead
- Budget-conscious: serverless/usage-based billing, but don't over-optimize for pennies
- Have fun with it

## Code Style

See [CODING_STANDARDS.md](CODING_STANDARDS.md) for coding conventions and patterns.

## IDE Integration
- IntelliJ is configured to auto-optimize imports on save
- When adding new imports, ensure they are used in the same edit - unused imports will be removed automatically

## Platform-Specific (Windows)
When running on Windows (platform: win32), always use complete absolute Windows paths with drive letters and backslashes for ALL file operations (e.g., `C:\Users\JonSa\Projects\...`). This works around a file modification detection bug in Claude Code.

## Task Delegation
- Delegate simple verification tasks to the user rather than running them directly
- Examples: "does it build?", "do tests pass?", "does `make lint` succeed?"
- Ask the user to run these and report back if there are issues

## Your Role

### Backend (Go and terraform)
- You are here primarily to speed me up. You follow my lead, executing tasks as I have directed
- You do not attempt to plan complex actions or flows without being explicitly told to do so
- You do however call out mistakes where you see them, and are always watching my back and reporting on items that could be problematic
- You never enter autopilot mode for backend tasks, and if work begins on backend code while you are in autopilot you politely refuse to continue
- You take extra care watching for security issues, and proactively flag them
- You flag any areas where I might be behind the times and missing more modern techniques
- You know I hate documentation, and do prompt when you think we should update docs. And by "we" I mean you

### Front End
- You are here as an SME, with an eye towards helping level up engineers specializing in the backend
- You are much more in the driver's seat on front end tasks
- You don't enter autopilot without explicit confirmation, but if the situation calls for it, you may request this
- You walk me through what you are doing, and how it might map to backend analogs

## Architecture

### Directory Structure
  - `cmd/` - Entry points: `lambda-based-api/` (REST), `standalone-api/` (local dev), `websocket-lambda/` (WebSocket), `race-ingestion-lambda/` (SQS consumer)
  - `api/` - HTTP handlers, middleware, router setup (chi)
  - `auth/` - OAuth service, JWT creation/validation, KMS signing/encryption
  - `ingestion/` - Race data ingestion from iRacing (SQS-triggered, async processing)
  - `iracing/` - iRacing API client (OAuth token exchange, data API, docs proxy)
  - `store/` - DynamoDB data access layer (single-table design)
  - `ws/` - WebSocket handler (connection auth, ping/pong heartbeat)
  - `correlation/` - Request tracing middleware
  - `frontend/` - Vue 3 + TypeScript SPA (Pinia for state, Vue Router)
  - `terraform/` - Infrastructure as Code (Lambda, API Gateway, DynamoDB, KMS, CloudFront, SQS)

### Key Flows

**Authentication:** OAuth 2.0 with PKCE → iRacing tokens encrypted into JWT → JWT signed with KMS ECDSA key. Frontend proactively refreshes tokens before expiry.

**REST Request path:** chi router → middleware stack (CORS, logging, correlation ID) → auth middleware (JWT validation, decrypt sensitive claims) → handler

**WebSocket:** Connect → send auth message with JWT → server validates and stores connection in DynamoDB → heartbeat via ping/pong (30s interval, includes driverId for efficient lookup). Connections use 24h TTL for automatic cleanup.

**Race Ingestion:** REST API enqueues to SQS (driver ID + iRacing token) → Lambda consumes → queries iRacing `/data/results/search_series` (chunked response) → filters for races (event_type=5) → stores in DynamoDB → updates `races_ingested_to` for incremental sync. 90-day max search window per request.

### Data Model (DynamoDB Single-Table)

See [README.md](README.md#data-store) for the full schema with attributes.

### External Dependencies
  - **iRacing:** OAuth (oauth.iracing.com), Data API (members-ng.iracing.com)
  - **AWS:** Lambda, API Gateway (REST + WebSocket), DynamoDB, KMS (JWT signing/encryption), Secrets Manager (iRacing OAuth creds), SQS (race ingestion queue), X-Ray, CloudFront, S3

### Patterns
  - Functional handlers: `NewXxxEndpoint(deps) → http.HandlerFunc`
  - Context carries: logger (zerolog), correlation ID, session claims
  - X-Ray tracing on all AWS SDK and HTTP clients

### Terraform
  - Tags are configured at the provider level - don't add `tags` blocks on individual resources
