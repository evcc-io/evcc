package tasmota

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelOptions(t *testing.T) {
	tc := []struct {
		name   string
		status string
		sns    string
		meter  bool
		expect []service.Option
	}{
		{
			name:   "single relay charger",
			status: `{"Status":{"FriendlyName":["Pump"]},"StatusSTS":{"POWER":"ON"}}`,
			sns:    `{"StatusSNS":{"Time":"2023-02-05T20:31:48"}}`,
			expect: []service.Option{{Value: "1", Label: "Pump", Match: true}},
		},
		{
			name:   "single relay meter without energy",
			status: `{"Status":{"FriendlyName":["Pump"]},"StatusSTS":{"POWER":"ON"}}`,
			sns:    `{"StatusSNS":{"Time":"2023-02-05T20:31:48"}}`,
			meter:  true,
			expect: []service.Option{{Value: "1", Label: "Pump"}},
		},
		{
			name:   "dual relay with energy",
			status: `{"Status":{"FriendlyName":["Pump"," "]},"StatusSTS":{"POWER1":"ON","POWER2":"OFF"}}`,
			sns:    `{"StatusSNS":{"ENERGY":{"Power":[8.15,0]}}}`,
			meter:  true,
			expect: []service.Option{
				{Value: "1", Label: "Pump", Hint: "8 W", Match: true},
				{Value: "2", Label: "Channel 2", Hint: "0 W", Match: true},
			},
		},
		{
			name:   "energy only meter",
			status: `{"Status":{"FriendlyName":["Meter"]},"StatusSTS":{}}`,
			sns:    `{"StatusSNS":{"ENERGY":{"Power":[100,200,300]}}}`,
			meter:  true,
			expect: []service.Option{
				{Value: "1", Label: "Meter", Hint: "100 W", Match: true},
				{Value: "2", Label: "Channel 2", Hint: "200 W", Match: true},
				{Value: "3", Label: "Channel 3", Hint: "300 W", Match: true},
			},
		},
		{
			name:   "no channels",
			status: `{"Status":{},"StatusSTS":{}}`,
			sns:    `{"StatusSNS":{}}`,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			var st statusResponse
			require.NoError(t, json.Unmarshal([]byte(tc.status), &st))

			var sns StatusSNSResponse
			require.NoError(t, json.Unmarshal([]byte(tc.sns), &sns))

			assert.Equal(t, tc.expect, channelOptions(st, sns, tc.meter))
		})
	}
}

func TestGetChannels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("user") != "admin" || q.Get("password") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch q.Get("cmnd") {
		case "Status 0":
			fmt.Fprint(w, `{"Status":{"FriendlyName":["Pump"]},"StatusSTS":{"POWER":"ON"}}`)
		case "Status 8":
			fmt.Fprint(w, `{"StatusSNS":{"ENERGY":{"Power":47.11}}}`)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	get := func(query string) []service.Option {
		w := httptest.NewRecorder()
		getChannels(w, httptest.NewRequest("GET", "/channels?"+query, nil))
		require.Equal(t, http.StatusOK, w.Code)

		var res []service.Option
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
		return res
	}

	assert.Equal(t, []service.Option{{Value: "1", Label: "Pump", Hint: "47 W", Match: true}},
		get("uri="+srv.URL+"&user=admin&password=secret"))

	assert.Empty(t, get("uri="+srv.URL+"&user=admin&password=wrong"))
	assert.Empty(t, get("uri=http://127.0.0.1:1"))
	assert.Empty(t, get(""))
}
