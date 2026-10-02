package grid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConversion(t *testing.T) {
	t.Cleanup(func() { IT = false })

	for _, tc := range []struct {
		it     bool
		phases int
		power  float64
	}{
		{false, 1, 2300},
		{false, 3, 6900},
		{true, 1, 2300},
		{true, 3, 3983.71},
	} {
		IT = tc.it

		power := CurrentToPower(10, tc.phases)
		assert.InDelta(t, tc.power, power, 0.01, "it=%v phases=%d", tc.it, tc.phases)
		assert.InDelta(t, 10, PowerToCurrent(power, tc.phases), 1e-9, "it=%v phases=%d", tc.it, tc.phases)
	}
}

func TestCurrentsToPower(t *testing.T) {
	t.Cleanup(func() { IT = false })

	for _, tc := range []struct {
		it         bool
		l1, l2, l3 float64
		power      float64
	}{
		{false, 10, 0, 0, 2300},
		{false, 10, 10, 10, 6900},
		{true, 10, 10, 0, 2300},
		{true, 0, 10, 10, 2300},
		{true, 10, 10, 10, 3983.71},
	} {
		IT = tc.it
		assert.InDelta(t, tc.power, CurrentsToPower(tc.l1, tc.l2, tc.l3), 0.01, "%+v", tc)
	}
}
