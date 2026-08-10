package logstash

import (
	"container/ring"
	"io"
	"maps"
	"slices"
	"strings"
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
	b := &buffer{
		data: ring.New(1),
		size: size,
	}
	b.length = b.data.Len() // keep length in sync with the initial ring
	return b
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

func (l *logger) Write(p []byte) (n int, err error) {
	s := string(p)
	if strings.HasPrefix(s, "[cache ]") {
		return len(p), nil
	}

	e := element(s)
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
	count := func(e entry) { size += int64(len(e.text)) }
	l.trace.visit(count)
	l.other.visit(count)

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

	var trace, other []entry
	filter := func(dst *[]entry) func(entry) {
		return func(e entry) {
			if all || e.text.match(areas, level) {
				*dst = append(*dst, e)
			}
		}
	}

	// trace entries can only match when trace is requested
	if level == jww.LevelTrace {
		l.trace.visit(filter(&trace))
	}
	l.other.visit(filter(&other))

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
