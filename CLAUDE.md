# Claude Code Instructions

## Code Style

### Comments
- Only add comments that provide meaningful context not obvious from the code itself
- Avoid redundant comments that just restate the name (e.g., `// KMSSigner handles cryptographic operations via KMS`)
- Good comments explain *why*, not *what*

### Go
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