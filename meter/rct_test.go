package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRCTTakeRSocStrategy verifies that a repeated BatteryNormal application
// (no intervening non-normal mode) is a no-op instead of a nil dereference (#31471).
func TestRCTTakeRSocStrategy(t *testing.T) {
	m := &RCT{}

	// nothing saved yet
	assert.Nil(t, m.takeRSocStrategy())

	m.setRSocStrategyIfAbsent(4)
	strategy := m.takeRSocStrategy()
	require.NotNil(t, strategy)
	assert.Equal(t, uint8(4), *strategy)

	// second take without an intervening set must not repeat the value or panic
	assert.Nil(t, m.takeRSocStrategy())
}

// TestRCTSetRSocStrategyIfAbsent verifies the saved value is never overwritten
// while still present, matching the original guard's intent.
func TestRCTSetRSocStrategyIfAbsent(t *testing.T) {
	m := &RCT{}

	m.setRSocStrategyIfAbsent(4)
	m.setRSocStrategyIfAbsent(7)

	strategy := m.takeRSocStrategy()
	require.NotNil(t, strategy)
	assert.Equal(t, uint8(4), *strategy)
}
