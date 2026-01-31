package cache

import (
	"sync"
)

// Cache provides thread-safe caching keyed by username
type Cache[T any] struct {
	mu    sync.Mutex
	cache map[string]T
}

// New creates a new Cache instance
func New[T any]() *Cache[T] {
	return &Cache[T]{
		cache: make(map[string]T),
	}
}

// GetOrCreate atomically gets or creates a cached object
func (c *Cache[T]) GetOrCreate(key string, createFn func() (T, error)) (T, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if res, ok := c.cache[key]; ok {
		return res, nil
	}

	res, err := createFn()
	if err != nil {
		var zero T
		return zero, err
	}

	c.cache[key] = res
	return res, nil
}
