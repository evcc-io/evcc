package tariff

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stromligningConfig() map[string]any {
	return map[string]any{
		"template":        "stromligning",
		"apikey":          "test-key",
		"productid":       "enkel_energi-enkel_kvartal",
		"supplierid":      "radius_c",
		"customergroupid": "C_egenproducent",
		"agreementstart":  "2025-03-01",
	}
}

func stromligningForecast(t *testing.T, values map[string]any) *plugin.Config {
	t.Helper()

	instance, err := templates.RenderInstance(templates.Tariff, values)
	require.NoError(t, err)
	assert.Equal(t, api.Custom, instance.Type)

	var cc struct {
		Forecast *plugin.Config
	}
	require.NoError(t, util.DecodeOther(instance.Other, &cc))
	require.NotNil(t, cc.Forecast)
	return cc.Forecast
}

func TestStromligning(t *testing.T) {
	for _, production := range []bool{false, true} {
		name := "consumption"
		if production {
			name = "production"
		}
		t.Run(name, func(t *testing.T) {
			values := stromligningConfig()
			if production {
				values["pricetype"] = "production"
				delete(values, "productid")
				delete(values, "customergroupid")
			}

			forecast := stromligningForecast(t, values)
			uri, ok := forecast.Other["uri"].(string)
			require.True(t, ok)
			u, err := url.Parse(uri)
			require.NoError(t, err)
			assert.Equal(t, "stromligning.dk", u.Host)

			expected := url.Values{"supplierId": {"radius_c"}, "lean": {"true"}}
			if production {
				assert.Equal(t, "/api/prices/production", u.Path)
			} else {
				assert.Equal(t, "/api/prices", u.Path)
				expected.Set("productId", "enkel_energi-enkel_kvartal")
				expected.Set("customerGroupId", "C_egenproducent")
				expected.Set("agreementStart", "2025-03-01")
			}
			assert.Equal(t, expected, u.Query())

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, u.Path, r.URL.Path)
				assert.Equal(t, expected, r.URL.Query())
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`[
					{"date":"2026-11-01T23:00:00.000Z","price":2.631529,"resolution":"15m"},
					{"date":"2026-11-01T23:15:00.000Z","price":-0.125,"resolution":"15m"},
					{"date":"2026-11-02T00:00:00Z","price":0,"resolution":"1h"}
				]`))
				assert.NoError(t, err)
			}))
			t.Cleanup(srv.Close)
			forecast.Other["uri"] = srv.URL + u.RequestURI()

			get, err := forecast.StringGetter(t.Context())
			require.NoError(t, err)
			data, err := get()
			require.NoError(t, err)

			var rates api.Rates
			require.NoError(t, json.Unmarshal([]byte(data), &rates))
			start := time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC)
			assert.Equal(t, api.Rates{
				{Start: start, End: start.Add(15 * time.Minute), Value: 2.631529},
				{Start: start.Add(15 * time.Minute), End: start.Add(30 * time.Minute), Value: -0.125},
				{Start: start.Add(time.Hour), End: start.Add(2 * time.Hour), Value: 0},
			}, rates)
		})
	}
}

func TestStromligningParameters(t *testing.T) {
	values := stromligningConfig()
	values["pricetype"] = "production"
	values["customergroupid"] = "group & special"
	forecast := stromligningForecast(t, values)
	uri, ok := forecast.Other["uri"].(string)
	require.True(t, ok)
	u, err := url.Parse(uri)
	require.NoError(t, err)
	assert.Equal(t, "group & special", u.Query().Get("customerGroupId"))
	assert.Empty(t, u.Query().Get("productId"))
	assert.Empty(t, u.Query().Get("agreementStart"))

	for _, key := range []string{"apikey", "supplierid", "productid"} {
		t.Run("missing "+key, func(t *testing.T) {
			values := stromligningConfig()
			values[key] = nil
			tmpl, err := templates.ByName(templates.Tariff, "stromligning")
			require.NoError(t, err)
			_, _, err = tmpl.RenderResult(templates.Tariff, templates.RenderModeUnitTest, values)
			require.ErrorContains(t, err, key)
		})
	}

	values = stromligningConfig()
	values["pricetype"] = "invalid"
	_, err = templates.RenderInstance(templates.Tariff, values)
	require.ErrorContains(t, err, "pricetype")
}

func TestStromligningErrors(t *testing.T) {
	forecast := stromligningForecast(t, stromligningConfig())
	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"unauthorized", `{}`, http.StatusUnauthorized},
		{"invalid json", `not json`, http.StatusOK},
		{"invalid date", `[{"date":"invalid","price":1,"resolution":"15m"}]`, http.StatusOK},
		{"invalid resolution", `[{"date":"2026-11-01T23:00:00Z","price":1,"resolution":"5m"}]`, http.StatusOK},
		{"missing price", `[{"date":"2026-11-01T23:00:00Z","resolution":"15m"}]`, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err)
			}))
			t.Cleanup(srv.Close)
			config := &plugin.Config{Source: forecast.Source, Other: maps.Clone(forecast.Other)}
			config.Other["uri"] = srv.URL
			get, err := config.StringGetter(t.Context())
			require.NoError(t, err)
			_, err = get()
			require.Error(t, err)
		})
	}
}
