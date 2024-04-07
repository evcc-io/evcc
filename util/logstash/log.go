package logstash

import (
	"container/ring"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

var defaultHandler = New(10000)

func Areas() []string {
	return defaultHandler.Areas()
}

func All(areas, levels []string) []string {
	return defaultHandler.All(areas, levels)
}

type logger struct {
	mu   sync.RWMutex
	data *ring.Ring
}

func New(size int) *logger {
	return &logger{
		data: ring.New(size),
	}
}

var _ io.Writer = (*logger)(nil)

func (l *logger) Write(p []byte) (n int, err error) {
	fmt.Println(" -- ", string(p))

	l.mu.Lock()
	defer l.mu.Unlock()

	l.data.Value = element{
		ts:  time.Now(),
		msg: string(p),
	}
	l.data = l.data.Next()

	return len(p), nil
}

func (l *logger) Areas() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data

	areas := make(map[string]struct{})
	for i := 0; i < r.Len(); i++ {
		if e, ok := r.Value.(element); ok {
			if a, _ := e.areaLevel(); a != "" {
				areas[a] = struct{}{}
			}
		}
		r = r.Next()
	}

	keys := maps.Keys(areas)
	slices.Sort(keys)
	return keys
}

func (l *logger) All(areas, levels []string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	r := l.data
	all := len(areas) == 0 && len(levels) == 0

	var res []string
	for i := 0; i < r.Len(); i++ {
		if e, ok := r.Value.(element); ok && !e.ts.IsZero() && (all || e.match(areas, levels)) {
			res = append(res, e.msg)
		}
		r = r.Next()
	}

	return res
}
