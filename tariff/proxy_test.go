package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hourlyTestRates returns one hour of slots with distinct values
func hourlyTestRates(values ...float64) api.Rates {
	start := time.Now().Truncate(time.Hour)

	res := make(api.Rates, 0, len(values))
	for i, v := range values {
		ts := start.Add(time.Duration(i) * SlotDuration)
		res = append(res, api.Rate{Start: ts, End: ts.Add(SlotDuration), Value: v})
	}

	return res
}

func rateValues(rr api.Rates) []float64 {
	res := make([]float64, 0, len(rr))
	for _, r := range rr {
		res = append(res, r.Value)
	}
	return res
}

func TestProxyAverageAndCacheable(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	cfg := func() map[string]any {
		return map[string]any{"interval": "1h", "features": []string{"average", "cacheable"}}
	}

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return hourlyTestRates(1, 2, 3, 4), nil }

	tf, err := NewProxyFromConfig(t.Context(), "cachetest", cfg())
	require.NoError(t, err)

	rr, err := tf.Rates()
	require.NoError(t, err)
	assert.Equal(t, []float64{2.5, 2.5, 2.5, 2.5}, rateValues(rr), "rates are averaged")
	assert.Equal(t, api.TariffTypePriceForecast, tf.Type())

	// the cache holds the tariff's original rates, not the averaged ones
	p := tf.(*average).Tariff.(*cachingProxy)
	assert.Equal(t, []float64{1, 2, 3, 4}, rateValues(p.cached.Rates))

	// restart with api down: cached rates are served through the average proxy
	cacheTestDown.Store(true)
	tf, err = NewProxyFromConfig(t.Context(), "cachetest", cfg())
	require.NoError(t, err)

	rr, err = tf.Rates()
	require.NoError(t, err)
	assert.Equal(t, []float64{2.5, 2.5, 2.5, 2.5}, rateValues(rr))
	assert.Nil(t, tf.(*average).Tariff.(*cachingProxy).tariff, "cache served without contacting upstream")
}

func TestProxySingleFeature(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	cacheTestDown.Store(false)
	cacheTestRates = func() (api.Rates, error) { return hourlyTestRates(1, 2, 3, 4), nil }

	// average only
	tf, err := NewProxyFromConfig(t.Context(), "cachetest", map[string]any{"features": []string{"average"}})
	require.NoError(t, err)
	require.IsType(t, new(average), tf)
	require.IsType(t, new(cacheTestTariff), tf.(*average).Tariff)

	// cacheable only
	tf, err = NewProxyFromConfig(t.Context(), "cachetest", map[string]any{"features": []string{"cacheable"}})
	require.NoError(t, err)
	require.IsType(t, new(cachingProxy), tf)

	// no proxy features
	tf, err = NewProxyFromConfig(t.Context(), "cachetest", map[string]any{})
	require.NoError(t, err)
	require.IsType(t, new(cacheTestTariff), tf)
}

// TestTemplateProxyFeatures guards against templates that declare a features key
// of their own and would produce duplicate yaml keys once a proxy feature is set
func TestTemplateProxyFeatures(t *testing.T) {
	for _, tmpl := range templates.ByClass(templates.Tariff) {
		average, _ := tmpl.ParamByName("average")
		cacheable, _ := tmpl.ParamByName("cacheable")
		if average < 0 || cacheable < 0 {
			continue
		}

		t.Run(tmpl.Template, func(t *testing.T) {
			values := tmpl.Defaults(templates.RenderModeUnitTest)
			values["template"] = tmpl.Template
			values["average"] = true
			values["cacheable"] = true

			inst, err := templates.RenderInstance(templates.Tariff, values)
			require.NoError(t, err)

			features, ok := inst.Other["features"].([]any)
			require.True(t, ok, "features: %v", inst.Other["features"])
			assert.Contains(t, features, "average")
			assert.Contains(t, features, "cacheable")

			seen := make(map[any]bool, len(features))
			for _, f := range features {
				assert.False(t, seen[f], "duplicate feature %v", f)
				seen[f] = true
			}
		})
	}
}
