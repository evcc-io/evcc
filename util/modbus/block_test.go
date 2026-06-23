package modbus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTTL = 2 * time.Second

func TestBlockContains(t *testing.T) {
	b := Block{Register: 37107, Count: 16}

	assert.True(t, b.Contains(37107, 2), "first register")
	assert.True(t, b.Contains(37121, 2), "last register exactly at block end")
	assert.False(t, b.Contains(37106, 2), "before block start")
	assert.False(t, b.Contains(37122, 2), "overruns block end")
}

func TestBlockByteOffset(t *testing.T) {
	b := Block{Register: 37107, Count: 16}

	assert.Equal(t, 0, b.ByteOffset(37107))
	assert.Equal(t, 12, b.ByteOffset(37113))
	assert.Equal(t, 28, b.ByteOffset(37121))
}

func TestCacheGetMiss(t *testing.T) {
	c := NewCache(testTTL)
	_, ok := c.get("nope")
	assert.False(t, ok)
}

func TestCachePutGet(t *testing.T) {
	c := NewCache(testTTL)
	c.put("k", []byte{1, 2, 3})
	got, ok := c.get("k")
	require.True(t, ok)
	assert.Equal(t, []byte{1, 2, 3}, got)
}

// TestCacheFetchSingleFlight verifies that concurrent fetches for the same key
// collapse into a single load and leave the cache warm.
func TestCacheFetchSingleFlight(t *testing.T) {
	c := NewCache(testTTL)
	key := "10.0.0.1::1/3/37107/16"
	want := []byte{0xde, 0xad, 0xbe, 0xef}

	var calls atomic.Int32
	var once sync.Once
	entered := make(chan struct{})
	release := make(chan struct{})
	load := func() ([]byte, error) {
		calls.Add(1)
		once.Do(func() { close(entered) })
		<-release // hold the flight open while followers pile up
		return want, nil
	}

	const n = 8
	var wg sync.WaitGroup
	payloads := make([][]byte, n)

	wg.Go(func() {
		got, _, err := c.Fetch(key, load)
		assert.NoError(t, err)
		payloads[0] = got
	})

	<-entered // ensure the flight is open before followers join

	for i := 1; i < n; i++ {
		wg.Go(func() {
			got, _, err := c.Fetch(key, load)
			assert.NoError(t, err)
			payloads[i] = got
		})
	}

	close(release)
	wg.Wait()

	assert.Equal(t, int32(1), calls.Load(), "concurrent reads of the same block must share one exchange")
	for i := range n {
		assert.Equal(t, want, payloads[i])
	}

	// cache is now warm: a subsequent read hits without loading again
	got, ok, err := c.Fetch(key, func() ([]byte, error) {
		t.Fatal("must not load on warm cache")
		return nil, nil
	})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, want, got)
}
