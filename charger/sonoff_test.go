package charger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSonoff(t *testing.T) {
	var on bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/rpc", r.URL.Path)

		var req struct {
			Method string `json:"method"`
			Params struct {
				Id int  `json:"id"`
				On bool `json:"on"`
			} `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, 1, req.Params.Id)

		var res any
		switch req.Method {
		case "Switch.GetStatus":
			res = map[string]any{"result": map[string]any{"id": 1, "on": on}}
		case "Switch.Set":
			on = req.Params.On
			res = map[string]any{"result": map[string]any{}}
		case "Meter.GetStatus":
			// 1100W, 123.45kWh
			res = map[string]any{"result": map[string]any{
				"power": 110000, "total_energy": 12345,
			}}
		default:
			res = map[string]any{"error": map[string]any{"code": -12805, "message": "internal error"}}
		}

		require.NoError(t, json.NewEncoder(w).Encode(res))
	}))
	defer srv.Close()

	c, err := NewSonoff(embed{}, srv.URL, "", "", 1, 0, 0)
	require.NoError(t, err)

	enabled, err := c.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	require.NoError(t, c.Enable(true))

	enabled, err = c.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	power, err := c.(api.Meter).CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 1100.0, power)

	me, ok := api.Cap[api.MeterEnergy](c)
	require.True(t, ok)

	energy, err := me.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 123.45, energy)

	status, err := c.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
}
