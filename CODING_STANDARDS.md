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

#### Test Packages

Tests should be in the same package as the code they're testing (e.g., `package store`, not `package store_test`). This allows access to unexported fields when needed for test setup, like injecting mock time functions.

#### Assertions

Use testify's `assert` and `require` packages for assertions instead of manual `t.Errorf`/`t.Fatalf` calls.

Prefer direct struct comparisons over extracting individual fields into maps/slices when verifying results. This makes expected values clearer and produces better diff output on failures:

```go
// Good - clear expected values, better failure messages
assert.Equal(t, []DriverSession{
    {DriverID: 1001, SubsessionID: 3, TrackID: 100, StartTime: time.Unix(3000, 0)},
    {DriverID: 1001, SubsessionID: 2, TrackID: 100, StartTime: time.Unix(2000, 0)},
}, sessions)

// Avoid - obscures expected values, worse failure messages
ids := make([]int64, len(sessions))
for i, s := range sessions {
    ids[i] = s.SubsessionID
}
assert.Equal(t, []int64{3, 2}, ids)
```

#### Table-Driven Tests

Use table-driven tests with fixtures in a `fixtures/` directory for expected responses.

#### Mocks

**Always use mockery-generated mocks** - never write manual mock implementations. Run `make generate-mocks` to generate mocks for all interfaces.

- Generated mocks live alongside the interface in `mock_<interface_name>_test.go`
- Use `NewMock<InterfaceName>(t)` constructor - it auto-registers cleanup and expectation assertions
- No need to call `mockStore.AssertExpectations(t)` - it's handled automatically

#### Mock Expectations

- Use the strongly-typed `.EXPECT()` syntax instead of `.On("methodName", ...)` - this provides compile-time safety and better IDE support
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
        mockFoo.EXPECT().DoThing(mock.Anything, tc.fooServiceCall.arg1, tc.fooServiceCall.arg2).
            Return(tc.fooServiceCall.result, tc.fooServiceCall.err)
        // ...
    })
}
```

- Use a slice of calls only when the number of invocations varies between test cases

## Terraform

- Tags are configured at the provider level - don't add `tags` blocks on individual resources