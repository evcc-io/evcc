package meter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShellyCurrentPowerForUsage(t *testing.T) {
	tests := []struct {
		name   string
		usage  string
		signed bool
		power  float64
		want   float64
	}{
		{name: "grid keeps sign", usage: "grid", power: -350, want: -350},
		{name: "unsigned pv uses absolute value", usage: "pv", power: -350, want: 350},
		{name: "unsigned pv keeps positive values", usage: "pv", power: 350, want: 350},
		{name: "signed pv inverts positive values", usage: "pv", signed: true, power: 350, want: -350},
		{name: "signed pv inverts negative values", usage: "pv", signed: true, power: -350, want: 350},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Shelly{usage: tc.usage}
			assert.Equal(t, tc.want, m.currentPowerForUsage(tc.power, tc.signed))
		})
	}
}
