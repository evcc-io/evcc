package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx("cached", NewCachedFromConfig)
}

// cachedPlugin caches the wrapped reading for the configured duration
type cachedPlugin struct {
	ctx   context.Context
	clock clock.Clock
	cache time.Duration
	value Config
}

// NewCachedFromConfig creates cached provider
func NewCachedFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	var cc struct {
		Cache time.Duration
		Value Config
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Cache <= 0 {
		return nil, errors.New("cache duration is required")
	}

	return &cachedPlugin{
		ctx:   ctx,
		clock: clock.New(),
		cache: cc.Cache,
		value: cc.Value,
	}, nil
}

func cachedGetter[T any](o *cachedPlugin, valuer func(ctx context.Context) (func() (T, error), error)) (func() (T, error), error) {
	value, err := valuer(o.ctx)
	if err != nil {
		return nil, fmt.Errorf("cached: %w", err)
	}

	var mu sync.Mutex
	var updated time.Time
	var val T

	return func() (T, error) {
		mu.Lock()
		defer mu.Unlock()

		// refresh on first call and once the cache has expired; failures are not cached
		if updated.IsZero() || o.clock.Since(updated) > o.cache {
			v, err := value()
			if err != nil {
				return v, err
			}
			val = v
			updated = o.clock.Now()
		}

		return val, nil
	}, nil
}

var _ StringGetter = (*cachedPlugin)(nil)

func (o *cachedPlugin) StringGetter() (func() (string, error), error) {
	return cachedGetter(o, o.value.StringGetter)
}

var _ FloatGetter = (*cachedPlugin)(nil)

func (o *cachedPlugin) FloatGetter() (func() (float64, error), error) {
	return cachedGetter(o, o.value.FloatGetter)
}

var _ IntGetter = (*cachedPlugin)(nil)

func (o *cachedPlugin) IntGetter() (func() (int64, error), error) {
	return cachedGetter(o, o.value.IntGetter)
}

var _ BoolGetter = (*cachedPlugin)(nil)

func (o *cachedPlugin) BoolGetter() (func() (bool, error), error) {
	return cachedGetter(o, o.value.BoolGetter)
}
