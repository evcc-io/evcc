package charger

import (
	"math"
	"testing"

	"github.com/evcc-io/evcc/util/grid"
	"github.com/stretchr/testify/assert"
)

func TestCurrentAbovePower(t *testing.T) {
	t.Cleanup(func() { grid.SetIT(false) })

	for _, it := range []bool{false, true} {
		grid.SetIT(it)

		for power := 1000.0; power < 2000; power++ {
			current := currentAbovePower(power, 3)
			assert.Greater(t, float64(uint32(grid.CurrentToPower(current, 3))), power, "it=%v power=%v", it, power)
			assert.LessOrEqual(t, float64(uint32(grid.CurrentToPower(math.Round(current*100-1)/100, 3))), power, "it=%v power=%v not minimal", it, power)
		}
	}
}
