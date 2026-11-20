package meter

import (
	"testing"

	"github.com/evcc-io/rct"
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

// TestRCTSetRSocStrategyExternal verifies External is never saved as the
// strategy to restore (#35719).
func TestRCTSetRSocStrategyExternal(t *testing.T) {
	m := &RCT{}

	m.setRSocStrategyIfAbsent(rct.SOCTargetExternal)
	assert.Nil(t, m.takeRSocStrategy())
}

// TestRCTSocStrategyToRestore verifies BatteryNormal resets a stuck External
// strategy but leaves other user-chosen strategies untouched (#35719).
func TestRCTSocStrategyToRestore(t *testing.T) {
	current := func(s uint8) func() (uint8, error) {
		return func() (uint8, error) { return s, nil }
	}

	m := &RCT{}

	// saved value wins
	m.setRSocStrategyIfAbsent(rct.SOCTargetSchedule)
	s, err := m.socStrategyToRestore(current(rct.SOCTargetExternal))
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, rct.SOCTargetSchedule, *s)

	// nothing saved, current External: reset to Internal
	s, err = m.socStrategyToRestore(current(rct.SOCTargetExternal))
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, rct.SOCTargetInternal, *s)

	// nothing saved, current other: no write
	s, err = m.socStrategyToRestore(current(rct.SOCTargetConstant))
	require.NoError(t, err)
	assert.Nil(t, s)
}
