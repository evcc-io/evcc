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

type logger struct {
	mu   sync.RWMutex
	data *ring.Ring
	size int
}

func New(size int) *logger {
	return &logger{
		data: ring.New(1),
		size: size,
	}
}

var _ io.Writer = (*logger)(nil)

func (l *logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !strings.HasPrefix(string(p), "[cache ]") {
		l.data.Value = element(string(p))

		// dynamically grow the ring
		if l.data.Len() < l.size {
			l.data.Link(ring.New(1))
		}

		l.data = l.data.Next()
	}

	return len(p), nil
}

func (l *logger) Size() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data
	var size int64

	for range r.Len() {
		if e, ok := r.Value.(element); ok {
			size += int64(len(e))
		}
		r = r.Next()
	}

	return size
}

func (l *logger) Areas() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data

	areas := make(map[string]struct{})
	for range r.Len() {
		r = r.Prev()
		if e, ok := r.Value.(element); ok && e != "" {
			if a, _ := e.areaLevel(); a != "" {
				areas[a] = struct{}{}
			}
		}
	}

	return slices.Sorted(maps.Keys(areas))
}

func (l *logger) All(areas []string, level jww.Threshold, count int) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data
	all := len(areas) == 0 && level == jww.LevelTrace

	res := make([]string, 0, r.Len())
	for range r.Len() {
		if e, ok := r.Value.(element); ok && e != "" && (all || e.match(areas, level)) {
			res = append(res, string(e))
		}
		r = r.Next()
	}

	if count > 0 && len(res) > count {
		res = res[len(res)-count:]
	}

	return res
}
