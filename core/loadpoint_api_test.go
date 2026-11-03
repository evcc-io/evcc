package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
)

func TestGetMaxPhaseCurrentPhaseSwitch(t *testing.T) {
	tc := []struct {
		name    string
		enabled bool
		since   time.Duration
		want    float64
	}{
		{"switching, enabled", true, 0, 7},
		{"switching, disabled", false, 0, 0.4},
		{"switch completed", true, 2 * phaseSwitchDuration, 0.4},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			clck := clock.NewMock()
			lp := &Loadpoint{
				clock:          clck,
				enabled:        tc.enabled,
				offeredCurrent: 7,
				chargeCurrents: []float64{0.4, 0.3, 0.3},
				phasesSwitched: clck.Now().Add(-tc.since),
			}

			assert.Equal(t, tc.want, lp.GetMaxPhaseCurrent())
		})
	}
}
