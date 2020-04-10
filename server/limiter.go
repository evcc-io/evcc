package server

import (
	"time"

	"github.com/andig/evcc/core"
	"github.com/benbjohnson/clock"
)

// Piper is the interface that data flow plugins must implement
type Piper interface {
	Pipe(in <-chan core.Param) <-chan core.Param
}

type cacheItem struct {
	updated time.Time
	val     interface{}
}

// Deduplicator allows filtering of channel data by given criteria
type Deduplicator struct {
	clock    clock.Clock
	interval time.Duration
	filter   map[string]interface{}
	cache    map[string]cacheItem
}

// NewDeduplicator creates Deduplicator
func NewDeduplicator(interval time.Duration, filter ...string) Piper {
	l := &Deduplicator{
		clock:    clock.New(),
		interval: interval,
		filter:   make(map[string]interface{}),
		cache:    make(map[string]cacheItem),
	}

	for _, f := range filter {
		l.filter[f] = struct{}{}
	}

	return l
}

func (l *Deduplicator) pipe(in <-chan core.Param, out chan<- core.Param) {
	for p := range in {
		// use loadpoint + param.Key as lookup key to value cache
		key := p.LoadPoint + "." + p.Key
		item, cached := l.cache[key]
		_, filtered := l.filter[p.Key]

		// forward if not cached
		if !cached || !filtered || filtered &&
			(l.clock.Since(item.updated) >= l.interval || p.Val != item.val) {
			l.cache[key] = cacheItem{updated: l.clock.Now(), val: p.Val}
			out <- p
		}
	}
}

// Pipe creates a new filtered output channel for given input channel
func (l *Deduplicator) Pipe(in <-chan core.Param) <-chan core.Param {
	out := make(chan core.Param)
	go l.pipe(in, out)
	return out
}

// Limiter allows filtering of channel data by given criteria
type Limiter struct {
	clock    clock.Clock
	interval time.Duration
	cache    map[string]cacheItem
}

// NewLimiter creates limiter
func NewLimiter(interval time.Duration) Piper {
	l := &Limiter{
		clock:    clock.New(),
		interval: interval,
		cache:    make(map[string]cacheItem),
	}

	return l
}

func (l *Limiter) pipe(in <-chan core.Param, out chan<- core.Param) {
	for p := range in {
		// use loadpoint + param.Key as lookup key to value cache
		key := p.LoadPoint + "." + p.Key
		item, cached := l.cache[key]

		// forward if not cached or expired
		if !cached || l.clock.Since(item.updated) >= l.interval {
			l.cache[key] = cacheItem{updated: l.clock.Now(), val: p.Val}
			out <- p
		}
	}
}

// Pipe creates a new filtered output channel for given input channel
func (l *Limiter) Pipe(in <-chan core.Param) <-chan core.Param {
	out := make(chan core.Param)
	go l.pipe(in, out)
	return out
}
