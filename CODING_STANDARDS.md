# Coding Standards

This document captures the coding conventions and patterns used in this project.

## Go

### General Patterns

- Prefer functional handler pattern (higher-order functions returning `http.HandlerFunc`) over struct-based handlers
- Interfaces belong with the consumer, not the implementation - accept interfaces, return concrete types

### JSON Tags

- **Frontend-facing structs:** JavaScript conventions (`json:"driverId"` - lowercase `Id`)
- **Backend/internal structs** (SQS messages, etc.): Go conventions (`json:"driverID"` - uppercase `ID`)

### Comments

- Only add comments that provide meaningful context not obvious from the code itself
- Avoid redundant comments that just restate the name (e.g., `// KMSSigner handles cryptographic operations via KMS`)
- Good comments explain *why*, not *what*

### Testing

#### Assertions

Use testify's `assert` and `require` packages for assertions instead of manual `t.Errorf`/`t.Fatalf` calls.

#### Table-Driven Tests

Use table-driven tests with fixtures in a `fixtures/` directory for expected responses.

#### Mock Expectations

- Avoid `mock.Anything` except for `context.Context` parameters - be explicit about expected values
- Define a struct type for each mock call with its expected inputs and outputs:

```go
type fooServiceCall struct {
    arg1   string
    arg2   int
    result string
    err    error
}

testCases := []struct {
    name string
    // inputs...

    fooServiceCall fooServiceCall

    // expected outputs...
}{
    {
        name: "successful case",
        fooServiceCall: fooServiceCall{
            arg1:   "expected-arg",
            arg2:   42,
            result: "expected-result",
        },
        // ...
    },
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        mockFoo := NewMockFooService(t)
        mockFoo.On("DoThing", mock.Anything, tc.fooServiceCall.arg1, tc.fooServiceCall.arg2).
            Return(tc.fooServiceCall.result, tc.fooServiceCall.err)
        // ...
    })
}
```

- Use a slice of calls only when the number of invocations varies between test cases

## Terraform

- Tags are configured at the provider level - don't add `tags` blocks on individual resources