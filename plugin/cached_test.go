package plugin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/require"
)

func TestCachedHitAndExpiry(t *testing.T) {
	c := clock.NewMock()
	o := &cachedPlugin{ctx: context.Background(), clock: c, cache: time.Hour}

	var calls int
	get, err := cachedGetter(o, func(context.Context) (func() (int64, error), error) {
		return func() (int64, error) {
			calls++
			return int64(calls), nil
		}, nil
	})
	require.NoError(t, err)

	// first call reads, subsequent calls within the window are served from cache
	v, err := get()
	require.NoError(t, err)
	require.EqualValues(t, 1, v)

	c.Add(59 * time.Minute)
	v, err = get()
	require.NoError(t, err)
	require.EqualValues(t, 1, v)
	require.Equal(t, 1, calls)

	// once the cache expires the next call refreshes
	c.Add(2 * time.Minute)
	v, err = get()
	require.NoError(t, err)
	require.EqualValues(t, 2, v)
	require.Equal(t, 2, calls)
}

func TestCachedDoesNotCacheErrors(t *testing.T) {
	c := clock.NewMock()
	o := &cachedPlugin{ctx: context.Background(), clock: c, cache: time.Hour}

	var calls int
	get, err := cachedGetter(o, func(context.Context) (func() (int64, error), error) {
		return func() (int64, error) {
			calls++
			return 0, errors.New("boom")
		}, nil
	})
	require.NoError(t, err)

	// failures must not be cached, so every call retries the source
	for range 3 {
		_, err := get()
		require.Error(t, err)
	}
	require.Equal(t, 3, calls)
}
