package iracing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog"
)

type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// WithS3Cache wraps a function with an S3 caching layer. Note that multiple instances of the same wrapped function (distributed system or otherwise)
// will only synchronize per instance, which is NBD as it's not the end of the world if the cached value is written multiple times.
func WithS3Cache[T any](client S3Client, bucketName string, key string, cacheTime time.Duration, toWrap func(ctx context.Context, accessToken string) (T, error)) func(ctx context.Context, accessToken string) (T, error) {
	mutex := sync.Mutex{}

	fetchAndCache := func(ctx context.Context, accessToken string) (T, error) {
		result, err := toWrap(ctx, accessToken)
		if err != nil {
			return result, err
		}

		marshalled, err := json.Marshal(result)
		if err != nil {
			// note, we -could- theoretically not error but that would mean we're trying to cache stuff we can't marshall so fail fast
			return result, fmt.Errorf("marshalling value to cache: %w", err)
		}
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(bucketName),
			Key:         aws.String(key),
			Body:        bytes.NewReader(marshalled),
			ContentType: aws.String("application/json"),
		})
		// fail fast again - the AWS clients have retries in them, so it means stuff is borked, and we're probably lacking IAM perms (or S3 is down, in which case the world is fucked)
		if err != nil {
			return result, fmt.Errorf("writing to S3: %w", err)
		}

		return result, nil
	}

	return func(ctx context.Context, accessToken string) (T, error) {
		mutex.Lock()
		defer mutex.Unlock()

		earliestTime := time.Now().Add(-cacheTime)

		s3Res, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket:          aws.String(bucketName),
			Key:             aws.String(key),
			IfModifiedSince: aws.Time(earliestTime),
		})
		var ret T

		if err != nil {
			// we're using the objects timestamp as a proxy for a TTL, so a not modified error means the value is too old
			notModified := isNotModifiedError(err)
			// and noSuchKey error means the value has never been fetched
			noSuchKey := !notModified && isNoSuchKeyError(err)
			if !notModified && !noSuchKey {
				return ret, fmt.Errorf("calling GetObject: %w", err)
			}
			zerolog.Ctx(ctx).Debug().Str("bucket", bucketName).Str("key", key).Bool("notModified", notModified).Bool("noSuchKey", noSuchKey).Msg("fetching fresh value")
			return fetchAndCache(ctx, accessToken)
		}
		defer s3Res.Body.Close()

		marshalled, err := io.ReadAll(s3Res.Body)
		if err != nil {
			return ret, fmt.Errorf("reading object body: %w", err)
		}

		err = json.Unmarshal(marshalled, &ret)
		if err != nil {
			return ret, fmt.Errorf("unmarshalling cached value: %w", err)
		}
		return ret, nil
	}
}

func isNotModifiedError(err error) bool {
	var respErr *smithyhttp.ResponseError
	return errors.As(err, &respErr) && respErr.HTTPStatusCode() == http.StatusNotModified
}

func isNoSuchKeyError(err error) bool {
	var noSuchKey *types.NoSuchKey
	return errors.As(err, &noSuchKey)
}
