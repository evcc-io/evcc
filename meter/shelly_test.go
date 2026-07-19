package meter

import (
	"testing"

	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/stretchr/testify/assert"
)

type fakeShellyGeneration struct{ gen int }

func (f fakeShellyGeneration) Enabled() (bool, error)         { return false, nil }
func (f fakeShellyGeneration) Enable(bool) error              { return nil }
func (f fakeShellyGeneration) CurrentPower() (float64, error) { return 0, nil }
func (f fakeShellyGeneration) TotalEnergy() (float64, error)  { return 0, nil }
func (f fakeShellyGeneration) ReturnEnergy() (float64, error) { return 0, nil }
func (f fakeShellyGeneration) IsThreePhase() bool             { return false }
func (f fakeShellyGeneration) Gen() int                       { return f.gen }

func TestShellyCurrentPowerForUsage(t *testing.T) {
	tests := []struct {
		name  string
		usage string
		gen   int
		power float64
		want  float64
	}{
		{name: "grid keeps sign", usage: "grid", gen: 1, power: -350, want: -350},
		{name: "gen1 pv uses absolute value", usage: "pv", gen: 1, power: -350, want: 350},
		{name: "gen2 pv uses absolute value for positive values", usage: "pv", gen: 2, power: 350, want: 350},
		{name: "gen2 pv uses absolute value for negative values", usage: "pv", gen: 2, power: -350, want: 350},
		{name: "gen3 pv turns the sign 1", usage: "pv", gen: 3, power: 350, want: -350},
		{name: "gen3 pv turns the sign", usage: "pv", gen: 3, power: -350, want: 350},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Shelly{
				usage: tc.usage,
				conn:  &shelly.Connection{Generation: fakeShellyGeneration{gen: tc.gen}},
			}
			assert.Equal(t, tc.want, m.currentPowerForUsage(tc.power))
		})
	}
}
