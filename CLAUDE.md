# Claude Code Instructions

## Code Style

### Comments
- Only add comments that provide meaningful context not obvious from the code itself
- Avoid redundant comments that just restate the name (e.g., `// KMSSigner handles cryptographic operations via KMS`)
- Good comments explain *why*, not *what*

### Go - 
- Prefer functional handler pattern (higher-order functions returning `http.HandlerFunc`) over struct-based handlers
- Interfaces belong with the consumer, not the implementation - accept interfaces, return concrete types
- Table-driven tests with fixtures in a `fixtures/` directory
- In mock expectations, avoid `mock.Anything` except for `context.Context` parameters - be explicit about expected values

## IDE Integration
- IntelliJ is configured to auto-optimize imports on save
- When adding new imports, ensure they are used in the same edit - unused imports will be removed automatically

## Task Delegation
- Delegate simple verification tasks to the user rather than running them directly
- Examples: "does it build?", "do tests pass?", "does `make lint` succeed?"
- Ask the user to run these and report back if there are issues
  
- ## Architecture

  ### Directory Structure
    - `cmd/` - Entry points: `lambda-based-api/` (prod) and `standalone-api/` (local dev)
    - `api/` - HTTP handlers, middleware, router setup (chi)
    - `auth/` - OAuth service, JWT creation/validation, KMS signing/encryption
    - `iracing/` - iRacing API client (OAuth token exchange, data API, docs proxy)
    - `store/` - DynamoDB data access layer (single-table design)
    - `correlation/` - Request tracing middleware
    - `frontend/` - Vue 3 + TypeScript SPA (Pinia for state, Vue Router)
    - `terraform/` - Infrastructure as Code (Lambda, API Gateway, DynamoDB, KMS, CloudFront)

  ### Key Flows

  **Authentication:** OAuth 2.0 with PKCE → iRacing tokens encrypted into JWT → JWT signed with KMS ECDSA key. Frontend proactively refreshes tokens before expiry.

  **Request path:** chi router → middleware stack (CORS, logging, correlation ID) → auth middleware (JWT validation, decrypt sensitive claims) → handler

  ### Data Model (DynamoDB Single-Table)
    - `driver#<id> / info` - Driver record (name, login stats)
    - `driver#<id> / note#<timestamp>#<session>#<lap>` - Driver notes
    - `track#<id> / info` - Track info
    - `global / counters` - Aggregate counts

  ### External Dependencies
    - **iRacing:** OAuth (oauth.iracing.com), Data API (members-ng.iracing.com)
    - **AWS:** Lambda, API Gateway, DynamoDB, KMS (JWT signing/encryption), Secrets Manager (iRacing OAuth creds), X-Ray, CloudFront, S3

  ### Patterns
    - Functional handlers: `NewXxxEndpoint(deps) → http.HandlerFunc`
    - Context carries: logger (zerolog), correlation ID, session claims
    - X-Ray tracing on all AWS SDK and HTTP clients
