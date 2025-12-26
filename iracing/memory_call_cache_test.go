package iracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithMemoryCache(t *testing.T) {
	testCases := []struct {
		name           string
		wrappedResult  testData
		wrappedErr     error
		expectedResult testData
		expectedErr    string
	}{
		{
			name:           "successful fetch - caches and returns value",
			wrappedResult:  testData{Value: "fetched"},
			expectedResult: testData{Value: "fetched"},
		},
		{
			name:        "wrapped function error - returns error without caching",
			wrappedErr:  errors.New("fetch failed"),
			expectedErr: "fetch failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			callCount := 0
			wrapped := func(ctx context.Context, accessToken string) (testData, error) {
				callCount++
				return tc.wrappedResult, tc.wrappedErr
			}

			cachedFunc := WithMemoryCache(wrapped)

			result, err := cachedFunc(context.Background(), "test-token")

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			assert.Equal(t, 1, callCount, "wrapped function should be called exactly once")
		})
	}
}

func TestWithMemoryCache_SecondCallUsesCachedValue(t *testing.T) {
	callCount := 0
	wrapped := func(ctx context.Context, accessToken string) (testData, error) {
		callCount++
		return testData{Value: "fetched"}, nil
	}

	cachedFunc := WithMemoryCache(wrapped)

	// First call - should fetch
	result1, err := cachedFunc(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, testData{Value: "fetched"}, result1)

	// Second call - should use cached value
	result2, err := cachedFunc(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, testData{Value: "fetched"}, result2)

	assert.Equal(t, 1, callCount, "wrapped function should only be called once")
}

func TestWithMemoryCache_ErrorDoesNotCache(t *testing.T) {
	callCount := 0
	shouldFail := true
	wrapped := func(ctx context.Context, accessToken string) (testData, error) {
		callCount++
		if shouldFail {
			return testData{}, errors.New("temporary failure")
		}
		return testData{Value: "success"}, nil
	}

	cachedFunc := WithMemoryCache(wrapped)

	// First call - fails
	_, err := cachedFunc(context.Background(), "test-token")
	require.Error(t, err)
	assert.Equal(t, 1, callCount)

	// Second call - should retry since first failed
	shouldFail = false
	result, err := cachedFunc(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, testData{Value: "success"}, result)
	assert.Equal(t, 2, callCount, "wrapped function should be called again after error")

	// Third call - should use cached value
	result, err = cachedFunc(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, testData{Value: "success"}, result)
	assert.Equal(t, 2, callCount, "wrapped function should not be called again after success")
}