package logstash

import (
	"testing"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/assert"
)

var (
	s1 = "[test1 ] TRACE test1"
	s2 = "[test2 ] ERROR test2"
	s3 = "[test1 ] TRACE test3"
)

func TestLog(t *testing.T) {
	log := New(3)

	// old to new
	log.Write([]byte(s1))
	log.Write([]byte(s2))
	log.Write([]byte(s3))

	idx := log.data

	assert.Equal(t, []string{s1, s2, s3}, log.All(nil, jww.LevelTrace, 0))
	assert.Equal(t, []string{s1, s2, s3}, log.All([]string{}, jww.LevelTrace, 0))

	assert.Equal(t, []string{s1, s3}, log.All([]string{"test1"}, jww.LevelTrace, 0))
	assert.Equal(t, []string{s1, s2, s3}, log.All(nil, jww.LevelTrace, 0))
	assert.Equal(t, []string{s3}, log.All(nil, jww.LevelTrace, 1))

	assert.Equal(t, []string{}, log.All(nil, jww.LevelFatal, 0))

	assert.Equal(t, idx, log.data, "data should not be changed after All() call")
	assert.Equal(t, []string{"test1", "test2"}, log.Areas())
}

// TestRingGrowsThenCaps writes more lines than the configured size and verifies
// the ring grows up to size and then keeps only the most recent size entries.
// This guards the grow/cap accounting in Write (length tracked in O(1)).
func TestRingGrowsThenCaps(t *testing.T) {
	const size = 3
	log := New(size)

	lines := []string{
		"[test ] TRACE line1",
		"[test ] TRACE line2",
		"[test ] TRACE line3",
		"[test ] TRACE line4",
		"[test ] TRACE line5",
	}
	for _, l := range lines {
		log.Write([]byte(l))
	}

	// only the last `size` lines survive, in chronological order
	got := log.All(nil, jww.LevelTrace, 0)
	assert.Len(t, got, size, "ring must not exceed configured size")
	assert.Equal(t, lines[len(lines)-size:], got)
}

// TestRingSkipCacheLinesDoesNotGrow verifies that "[cache ]" lines take the
// early-return path: they are not stored and must neither grow the ring nor the
// length counter, which has to stay in sync with data.Len().
func TestRingSkipCacheLinesDoesNotGrow(t *testing.T) {
	log := New(3)

	// grow the ring past a single node first, so a stray cursor advance on the
	// cache path would actually be observable (Next() on a 1-node ring is a no-op)
	log.Write([]byte(s1))
	log.Write([]byte(s2))

	stored := log.All(nil, jww.LevelTrace, 0)
	lenBefore := log.length
	cursorBefore := log.data

	for range 10 {
		log.Write([]byte("[cache ] cache line"))
	}

	assert.Equal(t, stored, log.All(nil, jww.LevelTrace, 0), "cache lines must not change stored content")
	assert.Equal(t, log.data.Len(), log.length, "length counter must stay in sync with the ring")
	assert.Equal(t, lenBefore, log.length, "cache lines must not grow the ring")
	assert.Same(t, cursorBefore, log.data, "cache lines must not advance the write cursor")
}

func BenchmarkLog(b *testing.B) {
	log := New(10000)

	// old to new
	log.Write([]byte(s1))
	log.Write([]byte(s2))
	log.Write([]byte(s3))

	for b.Loop() {
		log.All(nil, jww.LevelTrace, 1)
	}
}
