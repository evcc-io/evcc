package charger

import (
	"testing"

	"github.com/evcc-io/evcc/charger/semp"
	"github.com/stretchr/testify/assert"
)

// TestBenderSEMPPhaseSwitching uses characteristics reported by two identical Mennekes AMTRON Professional
// (fw 5.22.5): the min power follows the active phases, a charger charging on 3 phases is not detected
func TestBenderSEMPPhaseSwitching(t *testing.T) {
	for _, tc := range []struct {
		min, max int
		expected bool
	}{
		{1386, 11088, true},  // idle on 1 phase
		{1380, 11088, true},  // idle on 1 phase, other voltage
		{4158, 11088, false}, // charging on 3 phases
		{4140, 11088, false},
		{0, 11088, false},
		{1386, 4600, false},
	} {
		assert.Equal(t, tc.expected, sempPhaseSwitching(semp.Characteristics{
			MinPowerConsumption: tc.min,
			MaxPowerConsumption: tc.max,
		}), "%+v", tc)
	}
}
