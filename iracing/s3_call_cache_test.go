package iracing

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type testData struct {
	Value string `json:"value"`
}

type getObjectCall struct {
	output *s3.GetObjectOutput
	err    error
}

type putObjectCall struct {
	err error
}

type wrappedFuncCall struct {
	result testData
	err    error
}

func TestWithS3Cache(t *testing.T) {
	bucketName := "test-bucket"
	key := "test-key"
	cacheTime := time.Hour

	testCases := []struct {
		name            string
		getObjectCall   getObjectCall
		putObjectCall   *putObjectCall
		wrappedFuncCall *wrappedFuncCall
		expectedResult  testData
		expectedErr     string
	}{
		{
			name: "cache hit - returns cached value",
			getObjectCall: getObjectCall{
				output: &s3.GetObjectOutput{
					Body: io.NopCloser(strings.NewReader(`{"value":"cached"}`)),
				},
			},
			expectedResult: testData{Value: "cached"},
		},
		{
			name: "cache miss (NoSuchKey) - fetches and caches",
			getObjectCall: getObjectCall{
				err: &types.NoSuchKey{},
			},
			wrappedFuncCall: &wrappedFuncCall{
				result: testData{Value: "fresh"},
			},
			putObjectCall:  &putObjectCall{},
			expectedResult: testData{Value: "fresh"},
		},
		{
			name: "cache stale (304 Not Modified) - fetches and caches",
			getObjectCall: getObjectCall{
				err: &smithyhttp.ResponseError{
					Response: &smithyhttp.Response{
						Response: &http.Response{
							StatusCode: http.StatusNotModified,
						},
					},
				},
			},
			wrappedFuncCall: &wrappedFuncCall{
				result: testData{Value: "refreshed"},
			},
			putObjectCall:  &putObjectCall{},
			expectedResult: testData{Value: "refreshed"},
		},
		{
			name: "GetObject unexpected error - returns error",
			getObjectCall: getObjectCall{
				err: errors.New("s3 unavailable"),
			},
			expectedErr: "calling GetObject: s3 unavailable",
		},
		{
			name: "wrapped function error - returns error",
			getObjectCall: getObjectCall{
				err: &types.NoSuchKey{},
			},
			wrappedFuncCall: &wrappedFuncCall{
				err: errors.New("upstream error"),
			},
			expectedErr: "upstream error",
		},
		{
			name: "PutObject error - returns error",
			getObjectCall: getObjectCall{
				err: &types.NoSuchKey{},
			},
			wrappedFuncCall: &wrappedFuncCall{
				result: testData{Value: "fresh"},
			},
			putObjectCall: &putObjectCall{
				err: errors.New("put failed"),
			},
			expectedErr: "writing to S3: put failed",
		},
		{
			name: "unmarshal error - returns error",
			getObjectCall: getObjectCall{
				output: &s3.GetObjectOutput{
					Body: io.NopCloser(strings.NewReader(`not json`)),
				},
			},
			expectedErr: "unmarshalling cached value:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockS3 := NewMockS3Client(t)

			mockS3.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
				return aws.ToString(input.Bucket) == bucketName && aws.ToString(input.Key) == key
			})).Return(tc.getObjectCall.output, tc.getObjectCall.err)

			if tc.putObjectCall != nil {
				mockS3.EXPECT().PutObject(mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
					return aws.ToString(input.Bucket) == bucketName &&
						aws.ToString(input.Key) == key &&
						aws.ToString(input.ContentType) == "application/json"
				})).Return(&s3.PutObjectOutput{}, tc.putObjectCall.err)
			}

			wrappedCallCount := 0
			wrapped := func(ctx context.Context, accessToken string) (testData, error) {
				wrappedCallCount++
				if tc.wrappedFuncCall != nil {
					return tc.wrappedFuncCall.result, tc.wrappedFuncCall.err
				}
				t.Fatal("wrapped function called unexpectedly")
				return testData{}, nil
			}

			cachedFunc := WithS3Cache(mockS3, bucketName, key, cacheTime, wrapped)

			result, err := cachedFunc(context.Background(), "test-token")

			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedResult, result)
			}

			if tc.wrappedFuncCall != nil {
				assert.Equal(t, 1, wrappedCallCount, "wrapped function should be called exactly once")
			} else {
				assert.Equal(t, 0, wrappedCallCount, "wrapped function should not be called")
			}
		})
	}
}

func TestWithS3Cache_SecondCallChecksS3Again(t *testing.T) {
	bucketName := "test-bucket"
	key := "test-key"
	cacheTime := time.Hour

	mockS3 := NewMockS3Client(t)

	// First call: cache miss
	mockS3.EXPECT().GetObject(mock.Anything, mock.Anything).
		Return(nil, &types.NoSuchKey{}).Once()
	mockS3.EXPECT().PutObject(mock.Anything, mock.Anything).
		Return(&s3.PutObjectOutput{}, nil).Once()

	// Second call: cache hit from S3
	mockS3.EXPECT().GetObject(mock.Anything, mock.Anything).
		Return(&s3.GetObjectOutput{
			Body: io.NopCloser(strings.NewReader(`{"value":"cached"}`)),
		}, nil).Once()

	wrappedCallCount := 0
	wrapped := func(ctx context.Context, accessToken string) (testData, error) {
		wrappedCallCount++
		return testData{Value: "fresh"}, nil
	}

	cachedFunc := WithS3Cache(mockS3, bucketName, key, cacheTime, wrapped)

	// First call - should fetch and cache
	result1, err := cachedFunc(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, testData{Value: "fresh"}, result1)

	// Second call - should get from S3 cache
	result2, err := cachedFunc(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, testData{Value: "cached"}, result2)

	assert.Equal(t, 1, wrappedCallCount, "wrapped function should only be called once")
}