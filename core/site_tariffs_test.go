package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/stretchr/testify/assert"
)

func TestFcstEnergyToAdd(t *testing.T) {
	// anchor "now" mid-slot so slot boundary is unambiguous
	now := time.Date(2026, 5, 10, 12, 7, 0, 0, time.UTC)
	currentSlot := now.Truncate(tariff.SlotDuration) // 12:00

	// constant-power forecast: 4 kW at every slot start, spanning yesterday → tomorrow
	solar := make(api.Rates, 0, 4*24*3)
	for i := -96; i < 2*96; i++ {
		start := currentSlot.Add(time.Duration(i) * tariff.SlotDuration)
		solar = append(solar, api.Rate{
			Start: start,
			End:   start.Add(tariff.SlotDuration),
			Value: 4000, // W
		})
	}

	const slotKWh = 4.0 * 0.25 // 4 kW × 15 min = 1 kWh per completed slot

	for _, tc := range []struct {
		name   string
		latest time.Time
		want   float64
	}{
		{
			"cold start: latest is zero → seed with previous completed slot",
			time.Time{},
			slotKWh,
		},
		{
			"up to date: latest is previous slot → no add",
			currentSlot.Add(-tariff.SlotDuration),
			0,
		},
		{
			"one full slot to add",
			currentSlot.Add(-2 * tariff.SlotDuration),
			slotKWh,
		},
		{
			"latest lags by 4 slots: only one slot added per call",
			currentSlot.Add(-4 * tariff.SlotDuration),
			slotKWh,
		},
		{
			"latest lags by hours: still only one slot per call",
			currentSlot.Add(-24 * tariff.SlotDuration),
			slotKWh,
		},
		{
			"partial in-progress slot is never added",
			currentSlot.Add(-tariff.SlotDuration), // up-to-date
			0,                                     // mid-slot 'now' must not contribute
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := fcstEnergyToAdd(solar, tc.latest, now)
			assert.InDelta(t, tc.want, got, 1e-9)
		})
	}
}
