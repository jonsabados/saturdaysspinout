package iracing

import (
	"context"
	"sync"
)

func WithMemoryCache[T any](toWrap func(ctx context.Context, accessToken string) (T, error)) func(ctx context.Context, accessToken string) (T, error) {
	mutex := sync.Mutex{}
	cached := false
	var value T

	return func(ctx context.Context, accessToken string) (T, error) {
		mutex.Lock()
		defer mutex.Unlock()

		if cached {
			return value, nil
		}
		got, err := toWrap(ctx, accessToken)
		if err != nil {
			return got, err
		}
		value = got
		cached = true
		return got, nil
	}
}
