package templates_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestWattwatchersPower(t *testing.T) {
	tmpl, err := templates.ByName(templates.Meter, "wattwatchers")
	require.NoError(t, err)

	for _, tc := range []struct {
		name, usage, channels, maxage, body string
		age                                 time.Duration
		want                                float64
		wantErr                             bool
	}{
		{name: "grid default", usage: "grid", body: `[{"timestamp": TIMESTAMP, "pRealKw": [-1.5, 2]}]`, want: -1500},
		{name: "pv default", usage: "pv", body: `[{"timestamp": TIMESTAMP, "pRealKw": [-1.5, 2]}]`, want: 2000},
		{name: "three phase grid", usage: "grid", channels: "1,2,3", body: `[{"timestamp": TIMESTAMP, "pRealKw": [1, -2, 3, 4, 5, 6]}]`, want: 2000},
		{name: "three phase pv", usage: "pv", channels: "4,5,6", body: `[{"timestamp": TIMESTAMP, "pRealKw": [1, 2, 3, 4, 5, 6]}]`, want: 15000},
		{name: "duplicate channels", usage: "grid", channels: "1,1", body: `[{"timestamp": TIMESTAMP, "pRealKw": [1]}]`, want: 1000},
		{name: "latest entry", usage: "grid", body: `[{"timestamp": 1, "pRealKw": [99]}, {"timestamp": TIMESTAMP, "pRealKw": [0]}]`},
		{name: "stale", usage: "grid", age: 30 * time.Minute, body: `[{"timestamp": TIMESTAMP, "pRealKw": [-5]}]`, wantErr: true},
		{name: "custom maxage", usage: "grid", maxage: "10m", age: 5 * time.Minute, body: `[{"timestamp": TIMESTAMP, "pRealKw": [1]}]`, want: 1000},
		{name: "empty", usage: "grid", body: `[]`, wantErr: true},
		{name: "missing timestamp", usage: "grid", body: `[{"pRealKw": [1]}]`, wantErr: true},
		{name: "invalid timestamp", usage: "grid", body: `[{"timestamp": "bad", "pRealKw": [1]}]`, wantErr: true},
		{name: "missing channel", usage: "pv", body: `[{"timestamp": TIMESTAMP, "pRealKw": [1]}]`, wantErr: true},
		{name: "partial phases", usage: "grid", channels: "1,2,3", body: `[{"timestamp": TIMESTAMP, "pRealKw": [1, null, 3]}]`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.ReplaceAll(tc.body, "TIMESTAMP", strconv.FormatInt(time.Now().Add(-tc.age).Unix(), 10))
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.Header().Set("Content-Type", "application/json")
				_, err := fmt.Fprint(w, body)
				assert.NoError(t, err)
			}))
			defer srv.Close()

			values := map[string]any{"usage": tc.usage, "serial": "D123456789012", "token": "test-token"}
			if tc.channels != "" {
				values["channels"] = tc.channels
			}
			if tc.maxage != "" {
				values["maxage"] = tc.maxage
			}
			b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, values)
			require.NoError(t, err)
			var cc struct{ Power plugin.Config }
			require.NoError(t, yaml.Unmarshal(b, &cc))
			require.Equal(t, "https://api-v3.wattwatchers.com.au/short-energy/D123456789012?convert[energy]=kW", cc.Power.Other["uri"])
			cc.Power.Other["uri"] = srv.URL
			get, err := cc.Power.FloatGetter(t.Context())
			require.NoError(t, err)

			power, err := get()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, power)
		})
	}
}

func TestWattwatchersInvalidConfig(t *testing.T) {
	tmpl, err := templates.ByName(templates.Meter, "wattwatchers")
	require.NoError(t, err)
	for _, values := range []map[string]any{
		{"channels": "0"}, {"channels": "7"}, {"channels": "1,-1"},
		{"channels": "1.5"}, {"channels": "1,foo"}, {"channels": "1] | error(\"injected\")"},
		{"maxage": "0s"}, {"maxage": "-1m"}, {"maxage": "invalid"},
	} {
		_, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, values)
		assert.Error(t, err, values)
	}
}

func TestWattwatchersSharedCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, err := fmt.Fprintf(w, `[{"timestamp": %d, "pRealKw": [1, 2]}]`, time.Now().Unix())
		assert.NoError(t, err)
	}))
	defer srv.Close()
	tmpl, err := templates.ByName(templates.Meter, "wattwatchers")
	require.NoError(t, err)

	for _, usage := range []string{"grid", "pv"} {
		b, _, err := tmpl.RenderResult(templates.Meter, templates.RenderModeInstance, map[string]any{"usage": usage, "token": "test-token"})
		require.NoError(t, err)
		var cc struct{ Power plugin.Config }
		require.NoError(t, yaml.Unmarshal(b, &cc))
		assert.Equal(t, "30s", cc.Power.Other["cache"])
		cc.Power.Other["uri"] = srv.URL
		get, err := cc.Power.FloatGetter(t.Context())
		require.NoError(t, err)
		_, err = get()
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), calls.Load())
}
