package logstash

import (
	"container/ring"
	"log/slog"
	"maps"
	"slices"
	"sync"
)

var DefaultHandler = New(10000)

func Areas() []string {
	return DefaultHandler.Areas()
}

func All(areas []string, level slog.Level, count int) []Entry {
	return DefaultHandler.All(areas, level, count)
}

type logger struct {
	mu   sync.RWMutex
	data *ring.Ring
	size int
	// length mirrors data.Len() so Add avoids an O(n) ring.Len() call on every
	// log line (see Add). Invariant: any code that changes the number of nodes
	// in data must keep length in sync.
	length int
}

func New(size int) *logger {
	l := &logger{
		data: ring.New(1),
		size: size,
	}
	l.length = l.data.Len() // keep length in sync with the initial ring
	return l
}

// Add appends a log entry to the ring buffer
func (l *logger) Add(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.data.Value = e

	// dynamically grow the ring until it reaches the configured size.
	// Track the length in O(1) instead of calling ring.Len(), which walks
	// the whole ring on every write — once the ring is full (size 10000)
	// that is 10000 pointer chases per log line and dominates CPU on weak
	// hardware (e.g. Victron Venus OS / ARMv7) under verbose trace logging.
	if l.length < l.size {
		l.data.Link(ring.New(1))
		l.length++
	}

	l.data = l.data.Next()
}

func (l *logger) Areas() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data

	areas := make(map[string]struct{})
	for range r.Len() {
		r = r.Prev()
		if e, ok := r.Value.(Entry); ok && e.Area != "" {
			areas[e.Area] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(areas))
}

func (l *logger) All(areas []string, level slog.Level, count int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data
	all := len(areas) == 0 && level <= LevelTrace

	res := make([]Entry, 0, r.Len())
	for range r.Len() {
		if e, ok := r.Value.(Entry); ok && (all || e.match(areas, level)) {
			res = append(res, e)
		}
		r = r.Next()
	}

	if count > 0 && len(res) > count {
		res = res[len(res)-count:]
	}

	return res
}
