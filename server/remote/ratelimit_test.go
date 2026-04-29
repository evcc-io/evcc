package remote

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthRateLimiter(t *testing.T) {
	t.Run("allows requests under threshold", func(t *testing.T) {
		rl := newAuthRateLimiter()

		for range rl.max {
			assert.True(t, rl.allow())
			rl.fail()
		}
	})

	t.Run("blocks after threshold", func(t *testing.T) {
		rl := newAuthRateLimiter()

		for range rl.max {
			rl.fail()
		}

		assert.False(t, rl.allow())
	})

	t.Run("recovers after window expires", func(t *testing.T) {
		now := time.Now()
		var mu sync.Mutex

		rl := newAuthRateLimiter()
		rl.now = func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return now
		}

		for range rl.max {
			rl.fail()
		}

		assert.False(t, rl.allow())

		// advance past window
		mu.Lock()
		now = now.Add(rl.window + time.Second)
		mu.Unlock()

		assert.True(t, rl.allow())
	})

	t.Run("successful auth does not count as failure", func(t *testing.T) {
		rl := newAuthRateLimiter()

		// fill up to max-1 failures
		for range rl.max - 1 {
			rl.fail()
		}

		// allow should still work (no fail() call = successful auth)
		assert.True(t, rl.allow())

		// still under threshold
		assert.True(t, rl.allow())
	})
}
