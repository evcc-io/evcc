package tariff

import (
	"context"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortSlot(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	_, ok := shortSlot(nil)
	assert.False(t, ok)

	_, ok = shortSlot(makeRates(now, SlotDuration, 4, 0))
	assert.False(t, ok, "15m slots")

	_, ok = shortSlot(makeRates(now, time.Hour, 4, 0))
	assert.False(t, ok, "1h slots")

	// truncated leading slot is legitimate
	rr := append(api.Rates{{Start: now.Add(-5 * time.Minute), End: now}}, makeRates(now, SlotDuration, 4, 0)...)
	_, ok = shortSlot(rr)
	assert.False(t, ok, "partial first slot")

	r, ok := shortSlot(makeRates(now, 5*time.Minute, 4, 0))
	assert.True(t, ok, "5m slots")
	assert.Equal(t, 5*time.Minute, r.End.Sub(r.Start))
}

func TestNewFromConfigRejectsShortSlots(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	registry.Add("test-short", func(map[string]any) (api.Tariff, error) {
		return &testTariff{rates: makeRates(now, 5*time.Minute, 4, 0), typ: api.TariffTypeSolar}, nil
	})
	registry.Add("test-15m", func(map[string]any) (api.Tariff, error) {
		return &testTariff{rates: makeRates(now, SlotDuration, 4, 0), typ: api.TariffTypeSolar}, nil
	})

	_, err := NewFromConfig(context.TODO(), "test-short", nil)
	require.ErrorContains(t, err, "shorter than 15m0s")

	_, err = NewFromConfig(context.TODO(), "test-15m", nil)
	require.NoError(t, err)
}
