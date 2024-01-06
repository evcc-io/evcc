package pipe

import (
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

// Piper is the interface that data flow plugins must implement
type Piper interface {
	Pipe(in <-chan util.Param) <-chan util.Param
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

func (l *Deduplicator) pipe(in <-chan util.Param, out chan<- util.Param) {
	for p := range in {
		key := p.UniqueID()
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
func (l *Deduplicator) Pipe(in <-chan util.Param) <-chan util.Param {
	out := make(chan util.Param)
	go l.pipe(in, out)
	return out
}

// Dropper allows filtering of channel data by given criteria
type Dropper struct {
	filter []string
}

// NewDropper creates Dropper
func NewDropper(filter ...string) Piper {
	return &Dropper{filter}
}

func (l *Dropper) pipe(in <-chan util.Param, out chan<- util.Param) {
	for p := range in {
		if slices.Contains(l.filter, p.Key) {
			continue
		}

		out <- p
	}
}

// Pipe creates a new filtered output channel for given input channel
func (l *Dropper) Pipe(in <-chan util.Param) <-chan util.Param {
	out := make(chan util.Param)
	go l.pipe(in, out)
	return out
}
