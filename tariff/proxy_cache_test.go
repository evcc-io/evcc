package tariff

import (
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestCachingErrors(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	tt := &testTariff{
		rates: api.Rates{api.Rate{
			Start: time.Now().Truncate(SlotDuration),
			End:   time.Now().Truncate(SlotDuration).Add(SlotDuration),
		}},
		typ: api.TariffTypePriceForecast,
	}
	registry.Add("test", func(map[string]any) (api.Tariff, error) {
		return tt, nil
	})

	trf, err := NewCachedFromConfig(t.Context(), "test", map[string]any{
		"features": []string{api.Cacheable.String()},
	})

	require.NoError(t, err)
	require.NotNil(t, trf)

	rr, err := trf.Rates()
	require.NoError(t, err)
	require.Equal(t, tt.rates, rr)

	t.Run("first error", func(t *testing.T) {
		tt.err = errors.New("http 429")
		rr, err = trf.Rates()
		require.Nil(t, rr)
		require.Error(t, err)
	})

	t.Run("first ok after error", func(t *testing.T) {
		tt.err = nil
		rr, err = trf.Rates()
		require.Equal(t, tt.rates, rr)
		require.NoError(t, err)
	})
}
