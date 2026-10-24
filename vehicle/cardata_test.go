package vehicle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCardataCache(t *testing.T) {
	for _, tc := range []struct {
		broker, streaming bool
		cache             time.Duration
	}{
		{false, false, 15 * time.Minute},
		{false, true, 30 * time.Minute},
		{true, false, 2 * time.Hour},
		{true, true, 2 * time.Hour},
	} {
		assert.Equal(t, tc.cache, cardataCache(tc.broker, tc.streaming), "broker %v, streaming %v", tc.broker, tc.streaming)
	}
}
