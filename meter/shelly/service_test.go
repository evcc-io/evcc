package shelly

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelOptions(t *testing.T) {
	channels := []channel{
		{id: 0, name: " Garden ", relay: true, metered: true, power: 635.4},
		{id: 1, metered: true, power: 12.6},
		{id: 2, relay: true},
	}

	// charger: relays match
	assert.Equal(t, []service.Option{
		{Value: "0", Label: "Garden", Hint: "635 W", Match: true},
		{Value: "1", Label: "Meter 2", Hint: "13 W"},
		{Value: "2", Label: "Switch 3", Match: true},
	}, channelOptions(channels, false))

	// meter: metered channels match
	assert.Equal(t, []service.Option{
		{Value: "0", Label: "Garden", Hint: "635 W", Match: true},
		{Value: "1", Label: "Meter 2", Hint: "13 W", Match: true},
		{Value: "2", Label: "Switch 3"},
	}, channelOptions(channels, true))

	assert.Empty(t, channelOptions(nil, true))
}

func TestGen1Channels(t *testing.T) {
	var res Gen1Status

	// Shelly EM: one relay, two emeters
	require.NoError(t, json.Unmarshal([]byte(`{"relays":[{"ison":true}],"emeters":[{"power":100.5},{"power":-20}]}`), &res))
	assert.Equal(t, []channel{
		{id: 0, relay: true, metered: true, power: 100.5},
		{id: 1, metered: true, power: -20.0},
	}, gen1Channels(res))

	// Shelly 1: relay without meter
	res = Gen1Status{}
	require.NoError(t, json.Unmarshal([]byte(`{"relays":[{"ison":false}]}`), &res))
	assert.Equal(t, []channel{{id: 0, relay: true}}, gen1Channels(res))
}

func TestGen2Channels(t *testing.T) {
	tc := []struct {
		name           string
		status, config string
		expected       []channel
	}{
		{
			"pro 4pm, switch 1 unnamed",
			`{"sys":{},"switch:0":{"id":0,"output":true,"apower":635.4},"switch:1":{"id":1,"output":false,"apower":0}}`,
			`{"switch:0":{"id":0,"name":"Garden"},"switch:1":{"id":1,"name":null}}`,
			[]channel{
				{id: 0, name: "Garden", relay: true, metered: true, power: 635.4},
				{id: 1, relay: true, metered: true, power: 0.0},
			},
		},
		{
			"plus 1, no power metering",
			`{"switch:0":{"id":0,"output":true}}`,
			`{"switch:0":{"id":0,"name":"Pump"}}`,
			[]channel{{id: 0, name: "Pump", relay: true}},
		},
		{
			"pro em, switch and em1 share id 0",
			`{"switch:0":{"id":0,"output":true},"em1:0":{"id":0,"act_power":-12.3},"em1:1":{"id":1,"act_power":4.5}}`,
			`{"switch:0":{"id":0,"name":"Relay"},"em1:0":{"id":0,"name":"Grid"},"em1:1":{"id":1,"name":"PV"}}`,
			[]channel{
				{id: 0, name: "Grid", relay: true, metered: true, power: -12.3},
				{id: 1, name: "PV", metered: true, power: 4.5},
			},
		},
		{
			"pro 3em triphase",
			`{"em:0":{"id":0,"total_act_power":1500}}`,
			`{"em:0":{"id":0,"name":null}}`,
			[]channel{{id: 0, metered: true, power: 1500.0}},
		},
		{
			"pm mini",
			`{"pm1:0":{"id":0,"apower":42}}`,
			`{"pm1:0":{"id":0,"name":"Heater"}}`,
			[]channel{{id: 0, name: "Heater", metered: true, power: 42.0}},
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			var status, config map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(tc.status), &status))
			require.NoError(t, json.Unmarshal([]byte(tc.config), &config))

			assert.Equal(t, tc.expected, gen2Channels(status, config))
		})
	}
}

func TestGetChannels(t *testing.T) {
	srv := shellyServer(t, map[string]string{
		"Shelly.GetStatus": `{"sys":{},"switch:0":{"id":0,"output":true,"apower":635.4},"switch:1":{"id":1,"output":false}}`,
		"Shelly.GetConfig": `{"switch:0":{"id":0,"name":" Garden "},"switch:1":{"id":1,"name":null}}`,
	})
	defer srv.Close()

	get := func(query string) []service.Option {
		w := httptest.NewRecorder()
		getChannels(w, httptest.NewRequest("GET", "/channels?"+query, nil))
		require.Equal(t, 200, w.Code)

		var res []service.Option
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		return res
	}

	uri := "uri=" + url.QueryEscape(srv.URL)

	assert.Equal(t, []service.Option{
		{Value: "0", Label: "Garden", Hint: "635 W", Match: true},
		{Value: "1", Label: "Switch 2", Match: true},
	}, get(uri))

	assert.Equal(t, []service.Option{
		{Value: "0", Label: "Garden", Hint: "635 W", Match: true},
		{Value: "1", Label: "Switch 2"},
	}, get(uri+"&metered=1"))

	// incomplete form and unreachable device yield an empty list
	assert.Empty(t, get(""))
	assert.Empty(t, get("uri=http://127.0.0.1:1"))
}
