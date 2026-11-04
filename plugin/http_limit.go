package plugin

import (
	"fmt"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"
)

var (
	httpMu                sync.Mutex
	httpMutexes           = map[string]*sync.Mutex{}
	httpConcurrencyGroups = map[string]*httpConcurrencyLimiter{}
)

type httpConcurrencyConfig struct {
	Group string
	Limit *int64
}

type httpConcurrencyLimiter struct {
	*semaphore.Weighted
	group string
	limit int64
}

func (c *httpConcurrencyConfig) limiter() (*httpConcurrencyLimiter, error) {
	if c == nil {
		return nil, nil
	}

	group := strings.TrimSpace(c.Group)
	if group == "" {
		return nil, fmt.Errorf("http concurrency: group is required")
	}
	limit := int64(1)
	if c.Limit != nil {
		limit = *c.Limit
	}
	if limit < 1 {
		return nil, fmt.Errorf("http concurrency group %q: limit must be positive", group)
	}

	httpMu.Lock()
	defer httpMu.Unlock()

	if limiter, ok := httpConcurrencyGroups[group]; ok {
		if limiter.limit != limit {
			return nil, fmt.Errorf("http concurrency group %q: limit %d conflicts with existing limit %d", group, limit, limiter.limit)
		}
		return limiter, nil
	}

	limiter := &httpConcurrencyLimiter{
		Weighted: semaphore.NewWeighted(limit),
		group:    group,
		limit:    limit,
	}
	httpConcurrencyGroups[group] = limiter
	return limiter, nil
}

func muForKey(key string) *sync.Mutex {
	httpMu.Lock()
	defer httpMu.Unlock()

	if mu, ok := httpMutexes[key]; ok {
		return mu
	}

	mu := new(sync.Mutex)
	httpMutexes[key] = mu
	return mu
}
