package shelly

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Gen2+ status responses
func TestUnmarshalGen2StatusResponse(t *testing.T) {
	{
		// Switch.GetStatus Endpoint
		var res Gen2SwitchStatus
		jsonstr := `{"id":0, "source":"HTTP", "output":false, "apower":47.11, "voltage":232.0, "current":0.000, "pf":0.00, "aenergy":{"total":5.125,"by_minute":[0.000,0.000,0.000],"minute_ts":1675718520},"temperature":{"tC":25.3, "tF":77.5}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, 5.125, res.Aenergy.Total)
		assert.Equal(t, 47.11, res.Apower)
	}

	{
		// EM1.GetStatus Endpoint
		var res Gen2EM1Status
		jsonstr := `{"id":"0","current":1.473,"voltage":226.9,"act_power":-332.2,"aprt_power":335,"pf":0.99,"freq":50,"calibration":"factory"}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, -332.2, res.ActPower)
		assert.Equal(t, 1.473, res.Current)
		assert.Equal(t, 226.9, res.Voltage)
	}

	{
		// EM1Data.GetStatus Endpoint
		var res Gen2EM1Data
		jsonstr := `{"id":"0","total_act_energy":1264.15,"total_act_ret_energy":144792.28}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, 1264.15, res.TotalActEnergy)
		assert.Equal(t, 144792.28, res.TotalActRetEnergy)
	}
}
