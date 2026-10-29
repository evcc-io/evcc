package templates_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func teslaBLEHTTPConfig(t *testing.T, address, timeout string) map[string]plugin.Config {
	t.Helper()

	u, err := url.Parse(address)
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)

	tmpl, err := templates.ByName(templates.Vehicle, "tesla-ble")
	require.NoError(t, err)
	values := map[string]any{"url": u.Scheme + "://" + host, "port": port, "vin": "TESTVIN"}
	if timeout != "" {
		values["timeout"] = timeout
	}
	b, _, err := tmpl.RenderResult(templates.Vehicle, templates.RenderModeInstance, values)
	require.NoError(t, err)

	var rendered map[string]any
	require.NoError(t, yaml.Unmarshal(b, &rendered))
	configs := make(map[string]plugin.Config)
	for _, name := range []string{"chargeEnable", "maxcurrent", "wakeup"} {
		data, err := yaml.Marshal(rendered[name])
		require.NoError(t, err)
		var config plugin.Config
		require.NoError(t, yaml.Unmarshal(data, &config))
		configs[name] = config
	}
	return configs
}

func TestTeslaBLECommandTimeout(t *testing.T) {
	for _, tc := range []struct {
		name, timeout string
		want          time.Duration
	}{
		{"default", "", 30 * time.Second},
		{"override", "75s", 75 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for name, config := range teslaBLEHTTPConfig(t, "http://127.0.0.1:8080", tc.timeout) {
				t.Run(name, func(t *testing.T) {
					p, err := plugin.NewHTTPPluginFromConfig(t.Context(), config.Other)
					require.NoError(t, err)
					assert.Equal(t, tc.want, p.(*plugin.HTTP).Client.Timeout)
				})
			}
		})
	}
}

func TestTeslaBLECommandTimeoutApplied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	for name, config := range teslaBLEHTTPConfig(t, server.URL, "50ms") {
		t.Run(name, func(t *testing.T) {
			p, err := plugin.NewHTTPPluginFromConfig(t.Context(), config.Other)
			require.NoError(t, err)
			var send func() error
			if name == "maxcurrent" {
				set, err := p.(plugin.IntSetter).IntSetter("maxcurrent")
				require.NoError(t, err)
				send = func() error { return set(16) }
			} else {
				set, err := p.(plugin.BoolSetter).BoolSetter("chargeenable")
				require.NoError(t, err)
				send = func() error { return set(false) }
			}
			err = send()
			require.Error(t, err)
			var timeout net.Error
			require.ErrorAs(t, err, &timeout)
			assert.True(t, timeout.Timeout())
		})
	}
}

func TestTeslaBLECommandBeyondDefaultHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/1/vehicles/TESTVIN/command/charge_stop", r.URL.Path)
		select {
		case <-r.Context().Done():
		case <-time.After(11 * time.Second):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := teslaBLEHTTPConfig(t, server.URL, "")["chargeEnable"]
	p, err := plugin.NewHTTPPluginFromConfig(t.Context(), config.Other)
	require.NoError(t, err)
	set, err := p.(plugin.BoolSetter).BoolSetter("chargeenable")
	require.NoError(t, err)
	require.NoError(t, set(false))
}
