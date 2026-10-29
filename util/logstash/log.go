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

// item is a stored log entry together with its global write sequence, used to
// restore chronological order when merging the trace and non-trace buffers.
type item struct {
	seq   uint64
	entry Entry
}

// buffer is a ring of items that grows lazily until it reaches size.
type buffer struct {
	data *ring.Ring
	size int
	// length mirrors data.Len() to keep add O(1); any change to the number of
	// ring nodes must keep it in sync
	length int
}

func newBuffer(size int) *buffer {
	return &buffer{data: ring.New(1), size: size, length: 1}
}

func (b *buffer) add(i item) {
	b.data.Value = i

	// grow the ring until it reaches the configured size; ring.Len() is avoided
	// as it walks the whole ring, dominating CPU on weak hardware under trace
	if b.length < b.size {
		b.data.Link(ring.New(1))
		b.length++
	}

	b.data = b.data.Next()
}

// visit calls fn for every stored item, oldest first
func (b *buffer) visit(fn func(item)) {
	r := b.data
	for range r.Len() {
		if i, ok := r.Value.(item); ok {
			fn(i)
		}
		r = r.Next()
	}
}

type logger struct {
	mu  sync.RWMutex
	seq uint64
	// trace entries get their own budget so that chatty areas (mqtt, httpd) cannot
	// evict the far rarer debug/info/error entries from the visible log
	trace *buffer
	other *buffer
}

func New(size int) *logger {
	return &logger{
		trace: newBuffer(size),
		other: newBuffer(size),
	}
}

// Add appends a log entry to the ring buffer
func (l *logger) Add(e Entry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.seq++
	if e.Level < slog.LevelDebug {
		l.trace.add(item{l.seq, e})
	} else {
		l.other.add(item{l.seq, e})
	}
}

func (l *logger) Areas() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	areas := make(map[string]struct{})
	collect := func(i item) {
		if i.entry.Area != "" {
			areas[i.entry.Area] = struct{}{}
		}
	}
	l.trace.visit(collect)
	l.other.visit(collect)

	return slices.Sorted(maps.Keys(areas))
}

func (l *logger) All(areas []string, level slog.Level, count int) []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	all := len(areas) == 0 && level <= LevelTrace

	matching := func(b *buffer) []item {
		var res []item
		b.visit(func(i item) {
			if all || i.entry.match(areas, level) {
				res = append(res, i)
			}
		})
		return res
	}

	// trace entries can only match when trace is requested
	var trace []item
	if level < slog.LevelDebug {
		trace = matching(l.trace)
	}
	other := matching(l.other)

	// both buffers are chronologically ordered, merge them by sequence
	res := make([]Entry, 0, len(trace)+len(other))
	for len(trace) > 0 || len(other) > 0 {
		if len(other) == 0 || (len(trace) > 0 && trace[0].seq < other[0].seq) {
			res = append(res, trace[0].entry)
			trace = trace[1:]
		} else {
			res = append(res, other[0].entry)
			other = other[1:]
		}
	}

	if count > 0 && len(res) > count {
		res = res[len(res)-count:]
	}

	return res
}
