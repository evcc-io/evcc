package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/tariff"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchSoc(t *testing.T) {
	atLeast50 := func(soc float32) bool { return soc >= 50 }

	eos := time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC)
	timestamps := []time.Time{eos.Add(-5 * time.Minute), eos, eos.Add(tariff.SlotDuration)}
	dt := []int{300, 900, 900}

	for _, tc := range []struct {
		ts       []float32
		expected time.Time
	}{
		// slot 0 is the remainder of the current slot, it ends at the boundary
		{[]float32{50, 0, 0}, eos},
		{[]float32{0, 50, 0}, eos.Add(tariff.SlotDuration)},
		{[]float32{0, 0, 50}, eos.Add(2 * tariff.SlotDuration)},
		{[]float32{50, 50, 50}, eos},
		// no match
		{[]float32{0, 0, 0}, time.Time{}},
		{[]float32{0, 0, 0, 50}, time.Time{}},
		{nil, time.Time{}},
	} {
		assert.Equal(t, tc.expected, matchSoc(tc.ts, timestamps, dt, atLeast50), "%v", tc.ts)
	}
	assert.Zero(t, matchSoc([]float32{50}, nil, dt, atLeast50))
	assert.Zero(t, matchSoc([]float32{50}, timestamps, nil, atLeast50))
	assert.Equal(t, eos.Add(time.Minute), matchSoc([]float32{0, 50}, timestamps, []int{300, 60}, atLeast50))

	result := optimizerResult{
		Req: optimizer.OptimizationInput{
			TimeSeries: optimizer.TimeSeries{Dt: dt},
			Batteries:  []optimizer.BatteryConfig{{}},
		},
		Res:     optimizer.OptimizationResult{Batteries: []optimizer.BatteryResult{{StateOfCharge: []float32{50, 0, 50}}}},
		Details: requestDetails{Timestamps: timestamps},
	}
	err := result.pruneExpired(eos.Add(4 * time.Second))
	require.NoError(t, err)
	assert.Equal(t, eos.Add(2*tariff.SlotDuration), matchSoc(result.Res.Batteries[0].StateOfCharge, result.Details.Timestamps, result.Req.TimeSeries.Dt, atLeast50))
}
