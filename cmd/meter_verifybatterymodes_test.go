package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/require"
)

func TestBatteryModeTransitions(t *testing.T) {
	all := []api.BatteryMode{api.BatteryNormal, api.BatteryHold, api.BatteryCharge, api.BatteryHoldCharge, api.BatteryDischarge}

	for i := range all {
		modes := all[:i+1]
		n := len(modes)
		seq := batteryModeTransitions(modes)
		require.Len(t, seq, n*(n-1), "n=%d", n)

		seen := make(map[[2]api.BatteryMode]int)
		from := modes[0]
		for _, to := range seq {
			require.NotEqual(t, from, to, "n=%d", n)
			seen[[2]api.BatteryMode{from, to}]++
			from = to
		}

		if n > 1 {
			require.Equal(t, modes[0], from, "n=%d: must end at start mode", n)
		}

		for _, a := range modes {
			for _, b := range modes {
				if a != b {
					require.Equal(t, 1, seen[[2]api.BatteryMode{a, b}], "n=%d: %s -> %s", n, a, b)
				}
			}
		}
	}
}
