package logstash

import (
	"bytes"
	"container/ring"
	"io"
	"maps"
	"slices"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
)

var DefaultHandler = New(10000)

func Areas() []string {
	return DefaultHandler.Areas()
}

func All(areas []string, level jww.Threshold, count int) []string {
	return DefaultHandler.All(areas, level, count)
}

func Size() int64 {
	return DefaultHandler.Size()
}

// entry is a stored log line together with its global write sequence, used to
// restore chronological order when merging the trace and non-trace buffers.
type entry struct {
	seq  uint64
	text element
}

// buffer is a ring of entries that grows lazily until it reaches size.
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

func (b *buffer) add(e entry) {
	b.data.Value = e

	// grow the ring until it reaches the configured size; ring.Len() is avoided
	// as it walks the whole ring, dominating CPU on weak hardware under trace
	if b.length < b.size {
		b.data.Link(ring.New(1))
		b.length++
	}

	b.data = b.data.Next()
}

// visit calls fn for every stored entry, oldest first
func (b *buffer) visit(fn func(entry)) {
	r := b.data
	for range r.Len() {
		if e, ok := r.Value.(entry); ok && e.text != "" {
			fn(e)
		}
		r = r.Next()
	}
}

type logger struct {
	mu  sync.RWMutex
	seq uint64
	// trace lines get their own budget so that chatty areas (mqtt, httpd) cannot
	// evict the far rarer debug/info/error lines from the visible log
	trace *buffer
	other *buffer
}

func New(size int) *logger {
	return &logger{
		trace: newBuffer(size),
		other: newBuffer(size),
	}
}

var _ io.Writer = (*logger)(nil)

func (l *logger) Write(p []byte) (int, error) {
	if bytes.HasPrefix(p, []byte("[cache ]")) {
		return len(p), nil
	}

	e := element(p)
	_, level := e.areaLevel()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.seq++
	if level == jww.LevelTrace {
		l.trace.add(entry{l.seq, e})
	} else {
		l.other.add(entry{l.seq, e})
	}

	return len(p), nil
}

func (l *logger) Size() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var size int64
	sum := func(e entry) { size += int64(len(e.text)) }
	l.trace.visit(sum)
	l.other.visit(sum)

	return size
}

func (l *logger) Areas() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	areas := make(map[string]struct{})
	collect := func(e entry) {
		if a, _ := e.text.areaLevel(); a != "" {
			areas[a] = struct{}{}
		}
	}
	l.trace.visit(collect)
	l.other.visit(collect)

	return slices.Sorted(maps.Keys(areas))
}

func (l *logger) All(areas []string, level jww.Threshold, count int) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	all := len(areas) == 0 && level == jww.LevelTrace

	matching := func(b *buffer) []entry {
		var res []entry
		b.visit(func(e entry) {
			if all || e.text.match(areas, level) {
				res = append(res, e)
			}
		})
		return res
	}

	// trace entries can only match when trace is requested
	var trace []entry
	if level == jww.LevelTrace {
		trace = matching(l.trace)
	}
	other := matching(l.other)

	// both buffers are chronologically ordered, merge them by sequence
	res := make([]string, 0, len(trace)+len(other))
	for len(trace) > 0 || len(other) > 0 {
		if len(other) == 0 || (len(trace) > 0 && trace[0].seq < other[0].seq) {
			res = append(res, string(trace[0].text))
			trace = trace[1:]
		} else {
			res = append(res, string(other[0].text))
			other = other[1:]
		}
	}

	if count > 0 && len(res) > count {
		res = res[len(res)-count:]
	}

	return res
}
