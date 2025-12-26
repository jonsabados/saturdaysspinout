package iracing

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGlobalInfoCachingClient_GetTracks_MemoryCacheHit(t *testing.T) {
	httpClient := NewMockHTTPClient(t)
	metricsClient := NewMockMetricsClient(t)
	mockS3 := NewMockS3Client(t)

	client := NewClient(httpClient, metricsClient, WithBaseURL("https://test.iracing.com"))
	cachingClient := NewGlobalInfoCachingClient(client, mockS3, "test-bucket", time.Hour)

	tracksJSON := `[{"track_id":1,"track_name":"Test Track"}]`

	// First call: S3 cache miss, fetches from iRacing, stores in S3
	mockS3.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Key == "tracks"
	})).Return(nil, &types.NoSuchKey{}).Once()

	httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "/data/track/get")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"link":"https://s3.example.com/tracks"}`)),
	}, nil).Once()

	httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == "https://s3.example.com/tracks"
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(tracksJSON)),
	}, nil).Once()

	mockS3.EXPECT().PutObject(mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Key == "tracks"
	})).Return(&s3.PutObjectOutput{}, nil).Once()

	// First call
	tracks1, err := cachingClient.GetTracks(context.Background(), "test-token")
	require.NoError(t, err)
	require.Len(t, tracks1, 1)
	assert.Equal(t, int64(1), tracks1[0].TrackID)
	assert.Equal(t, "Test Track", tracks1[0].TrackName)

	// Second call: memory cache hit - no S3 or HTTP calls
	tracks2, err := cachingClient.GetTracks(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, tracks1, tracks2)
}

func TestGlobalInfoCachingClient_GetTracks_S3CacheHit(t *testing.T) {
	httpClient := NewMockHTTPClient(t)
	metricsClient := NewMockMetricsClient(t)
	mockS3 := NewMockS3Client(t)

	client := NewClient(httpClient, metricsClient, WithBaseURL("https://test.iracing.com"))
	cachingClient := NewGlobalInfoCachingClient(client, mockS3, "test-bucket", time.Hour)

	tracksJSON := `[{"track_id":2,"track_name":"Cached Track"}]`

	// S3 cache hit - no HTTP calls needed
	mockS3.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Key == "tracks"
	})).Return(&s3.GetObjectOutput{
		Body: io.NopCloser(strings.NewReader(tracksJSON)),
	}, nil).Once()

	tracks, err := cachingClient.GetTracks(context.Background(), "test-token")
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	assert.Equal(t, int64(2), tracks[0].TrackID)
	assert.Equal(t, "Cached Track", tracks[0].TrackName)
}

func TestGlobalInfoCachingClient_GetCars_MemoryCacheHit(t *testing.T) {
	httpClient := NewMockHTTPClient(t)
	metricsClient := NewMockMetricsClient(t)
	mockS3 := NewMockS3Client(t)

	client := NewClient(httpClient, metricsClient, WithBaseURL("https://test.iracing.com"))
	cachingClient := NewGlobalInfoCachingClient(client, mockS3, "test-bucket", time.Hour)

	carsJSON := `[{"car_id":42,"car_name":"Test Car"}]`

	// First call: S3 cache miss
	mockS3.EXPECT().GetObject(mock.Anything, mock.MatchedBy(func(input *s3.GetObjectInput) bool {
		return *input.Key == "cars"
	})).Return(nil, &types.NoSuchKey{}).Once()

	httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "/data/car/get")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"link":"https://s3.example.com/cars"}`)),
	}, nil).Once()

	httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == "https://s3.example.com/cars"
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(carsJSON)),
	}, nil).Once()

	mockS3.EXPECT().PutObject(mock.Anything, mock.MatchedBy(func(input *s3.PutObjectInput) bool {
		return *input.Key == "cars"
	})).Return(&s3.PutObjectOutput{}, nil).Once()

	// First call
	cars1, err := cachingClient.GetCars(context.Background(), "test-token")
	require.NoError(t, err)
	require.Len(t, cars1, 1)
	assert.Equal(t, int64(42), cars1[0].CarID)

	// Second call: memory cache hit
	cars2, err := cachingClient.GetCars(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, cars1, cars2)
}

func TestGlobalInfoCachingClient_NonCachedMethodsPassThrough(t *testing.T) {
	httpClient := NewMockHTTPClient(t)
	metricsClient := NewMockMetricsClient(t)
	mockS3 := NewMockS3Client(t)

	client := NewClient(httpClient, metricsClient, WithBaseURL("https://test.iracing.com"))
	cachingClient := NewGlobalInfoCachingClient(client, mockS3, "test-bucket", time.Hour)

	// GetUserInfo should pass through to the embedded client (no S3 caching)
	httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
		return strings.Contains(req.URL.String(), "/data/member/info")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"link":"https://s3.example.com/member"}`)),
	}, nil).Once()

	httpClient.EXPECT().Do(mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == "https://s3.example.com/member"
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"cust_id":123,"display_name":"Test User","member_since":"2024-01-01"}`)),
	}, nil).Once()

	userInfo, err := cachingClient.GetUserInfo(context.Background(), "test-token")
	require.NoError(t, err)
	assert.Equal(t, int64(123), userInfo.UserID)
}