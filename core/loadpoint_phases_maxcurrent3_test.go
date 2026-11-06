package core

// phase-dependent max current limit.
//
// phasesmaxcurrent3 caps the charge current on 3 phases (e.g. TWC3: 1p 18 A,
// 3p 16 A) while leaving 1-phase charging at maxCurrent untouched. Without
// the parameter the behavior is identical to upstream.

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEffectiveMaxCurrentPhases(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := &struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{
		api.NewMockCharger(ctrl),
		api.NewMockPhaseSwitcher(ctrl),
	}

	tcs := []struct {
		desc     string
		phases   int
		max3     float64 // PhasesMaxCurrent3
		expected float64
	}{
		{"1p, no 3p limit → maxCurrent 18A", 1, 0, 18},
		{"3p, no 3p limit → maxCurrent 18A (upstream behavior)", 3, 0, 18},
		{"1p, 3p limit 16A → 18A (limit only applies on 3 phases)", 1, 16, 18},
		{"3p, 3p limit 16A → 16A", 3, 16, 16},
		{"3p, 3p limit 12A → 12A", 3, 12, 12},
		{"3p, 3p limit 20A above maxCurrent → 18A (cap never raises)", 3, 20, 18},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			lp := &Loadpoint{
				log:               util.NewLogger("foo"),
				charger:           charger,
				minCurrent:        2,
				maxCurrent:        18,
				phases:            tc.phases,
				measuredPhases:    tc.phases,
				PhasesMaxCurrent3: tc.max3,
			}
			require.Equal(t, tc.expected, lp.effectiveMaxCurrent(), tc.desc)
		})
	}
}
