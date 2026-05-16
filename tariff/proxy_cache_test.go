package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestRatesValidTomorrowCoverage(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	start := time.Date(2026, 5, 15, 22, 0, 0, 0, time.UTC) // Europe/Berlin midnight in UTC

	makeRates := func(slots int) api.Rates {
		res := make(api.Rates, slots)
		for i := range slots {
			slotStart := start.Add(time.Duration(i) * SlotDuration)
			res[i] = api.Rate{
				Start: slotStart,
				End:   slotStart.Add(SlotDuration),
			}
		}
		return res
	}

	until := now.Add(24*time.Hour + SlotDuration)

	assert.False(t, ratesValid(makeRates(96), until))  // only today
	assert.True(t, ratesValid(makeRates(192), until)) // today + tomorrow
}

