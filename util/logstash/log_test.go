package logstash

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func entry(area string, level slog.Level, msg string) Entry {
	return Entry{Time: time.Now(), Area: area, Level: level, Message: msg}
}

var (
	e1 = entry("test1", LevelTrace, "test1")
	e2 = entry("test2", slog.LevelError, "test2")
	e3 = entry("test1", LevelTrace, "test3")
)

func TestLog(t *testing.T) {
	log := New(3)

	// old to new
	log.Add(e1)
	log.Add(e2)
	log.Add(e3)

	trace, other := log.trace.data, log.other.data

	assert.Equal(t, []Entry{e1, e2, e3}, log.All(nil, LevelTrace, 0))
	assert.Equal(t, []Entry{e1, e2, e3}, log.All([]string{}, LevelTrace, 0))

	assert.Equal(t, []Entry{e1, e3}, log.All([]string{"test1"}, LevelTrace, 0))
	assert.Equal(t, []Entry{e3}, log.All(nil, LevelTrace, 1))

	assert.Equal(t, []Entry{}, log.All(nil, LevelFatal, 0))

	assert.Same(t, trace, log.trace.data, "data should not be changed after All() call")
	assert.Same(t, other, log.other.data, "data should not be changed after All() call")
	assert.Equal(t, []string{"test1", "test2"}, log.Areas())
}

// TestTraceBudget verifies that a flood of trace entries cannot evict non-trace
// entries, which is what made the log view show mqtt/httpd only (#32673).
func TestTraceBudget(t *testing.T) {
	log := New(3)

	log.Add(e2)
	for i := range 100 {
		log.Add(entry("mqtt", LevelTrace, fmt.Sprintf("send %d", i)))
	}

	assert.Equal(t, []Entry{e2}, log.All(nil, slog.LevelDebug, 0))
	assert.Equal(t, []string{"mqtt", "test2"}, log.Areas())
}

// TestTraceMerge verifies that trace and non-trace entries are returned in write order.
func TestTraceMerge(t *testing.T) {
	log := New(10)

	e4 := entry("test2", slog.LevelInfo, "test4")
	for _, e := range []Entry{e1, e2, e3, e4} {
		log.Add(e)
	}

	assert.Equal(t, []Entry{e1, e2, e3, e4}, log.All(nil, LevelTrace, 0))
	assert.Equal(t, []Entry{e2, e4}, log.All(nil, slog.LevelDebug, 0))
}

func BenchmarkAdd(b *testing.B) {
	log := New(1000)
	e := entry("mqtt", LevelTrace, "send evcc/site/pv/power 1234")
	b.ReportAllocs()
	for b.Loop() {
		log.Add(e)
	}
}

// TestRingGrowsThenCaps adds more entries than the configured size and verifies
// the ring grows up to size and then keeps only the most recent size entries.
func TestRingGrowsThenCaps(t *testing.T) {
	const size = 3
	log := New(size)

	var entries []Entry
	for i := range 5 {
		e := entry("test", LevelTrace, fmt.Sprintf("line%d", i+1))
		entries = append(entries, e)
		log.Add(e)
	}

	// only the last `size` entries survive, in chronological order
	got := log.All(nil, LevelTrace, 0)
	assert.Len(t, got, size, "ring must not exceed configured size")
	assert.Equal(t, entries[len(entries)-size:], got)
}

func TestEntryString(t *testing.T) {
	e := Entry{
		Time:    time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC),
		Area:    "lp-1",
		Level:   slog.LevelWarn,
		Message: "hello",
		Attrs:   map[string]string{"component": "loadpoint", "title": "Garage 1"},
	}

	assert.Equal(t, "[lp-1  ] WARN 2026/07/21 10:00:00 hello component=loadpoint title=\"Garage 1\"\n", e.String())
}

func TestEntryJSON(t *testing.T) {
	e := Entry{
		Time:    time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC),
		Area:    "site",
		Level:   LevelTrace,
		Message: "hello",
	}

	b, err := e.MarshalJSON()
	assert.NoError(t, err)
	assert.JSONEq(t, `{"time":"2026-07-21T10:00:00Z","area":"site","level":"trace","message":"hello"}`, string(b))
}

func BenchmarkLog(b *testing.B) {
	log := New(10000)

	// old to new
	log.Add(e1)
	log.Add(e2)
	log.Add(e3)

	for b.Loop() {
		log.All(nil, LevelTrace, 1)
	}
}
