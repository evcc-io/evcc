package tariff

import (
	"context"
	"fmt"
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

func TestNewFromConfigWrapsMixedSlots(t *testing.T) {
	now := time.Now().Truncate(SlotDuration)

	// day-ahead 15m slots followed by hourly forecast
	rr := append(makeRates(now, SlotDuration, 4, 0), makeRates(now.Add(time.Hour), time.Hour, 2, 0)...)
	registry.Add("test-mixed", func(map[string]any) (api.Tariff, error) {
		return &testTariff{rates: rr, typ: api.TariffTypePriceForecast}, nil
	})

	tf, err := NewFromConfig(context.TODO(), "test-mixed", nil)
	require.NoError(t, err)

	res, err := tf.Rates()
	require.NoError(t, err)
	require.Len(t, res, 12)
	assert.Equal(t, now.Add(3*time.Hour), res[len(res)-1].End)
}

func TestCustomForecastKeepsEstimate(t *testing.T) {
	now := time.Now().Truncate(SlotDuration).UTC()
	slot := func(i int, estimate string) string {
		return fmt.Sprintf(`{"start":%q,"end":%q,"value":0.2%s}`,
			now.Add(time.Duration(i)*SlotDuration).Format(time.RFC3339),
			now.Add(time.Duration(i+1)*SlotDuration).Format(time.RFC3339), estimate)
	}

	tf, err := NewFromConfig(context.TODO(), api.Custom, map[string]any{
		"forecast": map[string]any{
			"source": "const",
			"value":  "[" + slot(0, "") + "," + slot(1, `,"estimate":true`) + "]",
		},
	})
	require.NoError(t, err)

	res, err := tf.Rates()
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.False(t, res[0].Estimate)
	assert.True(t, res[1].Estimate)
}
