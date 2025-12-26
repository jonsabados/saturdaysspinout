package iracing

import (
	"context"
	"time"
)

type GlobalInfoCachingClient struct {
	*Client

	trackCache      func(ctx context.Context, accessToken string) ([]TrackInfo, error)
	trackAssetCache func(ctx context.Context, accessToken string) (map[int64]TrackAssets, error)

	carsCache      func(ctx context.Context, accessToken string) ([]CarInfo, error)
	carAssetsCache func(ctx context.Context, accessToken string) (map[int64]CarAssets, error)
}

func NewGlobalInfoCachingClient(toWrap *Client, s3Client S3Client, bucketName string, s3CacheDuration time.Duration) *GlobalInfoCachingClient {
	return &GlobalInfoCachingClient{
		Client:          toWrap,
		trackCache:      WithMemoryCache(WithS3Cache(s3Client, bucketName, "tracks", s3CacheDuration, toWrap.GetTracks)),
		trackAssetCache: WithMemoryCache(WithS3Cache(s3Client, bucketName, "trackAssets", s3CacheDuration, toWrap.GetTrackAssets)),
		carsCache:       WithMemoryCache(WithS3Cache(s3Client, bucketName, "cars", s3CacheDuration, toWrap.GetCars)),
		carAssetsCache:  WithMemoryCache(WithS3Cache(s3Client, bucketName, "carAssets", s3CacheDuration, toWrap.GetCarAssets)),
	}
}

func (g *GlobalInfoCachingClient) GetTracks(ctx context.Context, accessToken string) ([]TrackInfo, error) {
	return g.trackCache(ctx, accessToken)
}

func (g *GlobalInfoCachingClient) GetTrackAssets(ctx context.Context, accessToken string) (map[int64]TrackAssets, error) {
	return g.trackAssetCache(ctx, accessToken)
}

func (g *GlobalInfoCachingClient) GetCars(ctx context.Context, accessToken string) ([]CarInfo, error) {
	return g.carsCache(ctx, accessToken)
}

func (g *GlobalInfoCachingClient) GetCarAssets(ctx context.Context, accessToken string) (map[int64]CarAssets, error) {
	return g.carAssetsCache(ctx, accessToken)
}
