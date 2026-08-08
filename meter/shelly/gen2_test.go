package shelly

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignedPower(t *testing.T) {
	assert.False(t, (&Connection{gen: 1}).SignedPower())
	assert.False(t, (&Connection{gen: 2}).SignedPower())
	assert.True(t, (&Connection{gen: 3}).SignedPower())
}

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

	{
		// ProOutputAddon.GetPeripherals Endpoint
		var res Gen2ProAddOnGetPeripherals
		channel := 0

		// Test with a valid switch ID
		jsonstr := `{"digital_out":{"switch:100":{}}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.NotEmpty(t, res.DigitalOut)
		assert.Equal(t, 100, parseAddOnSwitchID(channel, res))

		// Test with no AddOn installed
		res = Gen2ProAddOnGetPeripherals{}
		jsonstr = `{"code":404,"message":"No handler for ProOutputAddon.GetPeripherals"}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, 0, parseAddOnSwitchID(channel, res))

		// Test for empty digital_out map in AddOn response
		res = Gen2ProAddOnGetPeripherals{}
		jsonstr = `{"digital_out":{}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, 0, parseAddOnSwitchID(channel, res))

		// Test with multiple AddOns installed (only the first ID will be returned)
		res = Gen2ProAddOnGetPeripherals{}
		jsonstr = `{"digital_out":{"switch:100":{},"switch:101":{}}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, 100, parseAddOnSwitchID(channel, res))

		// Test for switch key <> 100
		res = Gen2ProAddOnGetPeripherals{}
		jsonstr = `{"digital_out":{"switch:abc":{}}}`
		require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
		assert.Equal(t, 0, parseAddOnSwitchID(channel, res))
	}
}

func TestSwitchEnergy(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		total, ret   float64
		hasReturnReg bool
	}{
		{
			// Shelly Plug S Gen3 as pv meter, https://github.com/evcc-io/evcc/issues/32213
			// no ret_aenergy register: aenergy is production, swapping it would report 0
			name:   "plug without reverse metering",
			status: `{"id":0,"source":"init","output":true,"apower":399.2,"voltage":240.8,"current":1.886,"aenergy":{"total":2574466.629,"by_minute":[7200.069,6121.044,5088.795],"minute_ts":1786033200},"temperature":{"tC":46.3,"tF":115.4}}`,
			total:  2574.466629,
		},
		{
			// aenergy holds both directions, so import is the difference
			name:         "switch with reverse metering",
			status:       `{"id":0,"output":true,"apower":-350,"aenergy":{"total":10000},"ret_aenergy":{"total":4000}}`,
			total:        6,
			ret:          4,
			hasReturnReg: true,
		},
		{
			// pure production: everything lands in ret_aenergy, import must not go negative
			name:         "switch measuring return only",
			status:       `{"id":0,"output":true,"aenergy":{"total":4000},"ret_aenergy":{"total":4000}}`,
			ret:          4,
			hasReturnReg: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var res Gen2SwitchStatus
			require.NoError(t, json.Unmarshal([]byte(tc.status), &res))

			assert.Equal(t, tc.hasReturnReg, res.Ret_Aenergy != nil, "ret_aenergy presence")

			total, ret := switchEnergy(res)
			assert.Equal(t, tc.total, total, "total energy")
			assert.Equal(t, tc.ret, ret, "return energy")

			// same values through the endpoint dispatch
			c := &gen2{
				methods: []string{"Switch.GetStatus"},
				switchstatus: util.ResettableCached(func() (Gen2SwitchStatus, error) {
					return res, nil
				}, time.Minute),
			}

			assert.Equal(t, tc.hasReturnReg, c.HasReturnEnergy(), "HasReturnEnergy")

			totalEnergy, err := c.TotalEnergy()
			require.NoError(t, err)
			assert.Equal(t, tc.total, totalEnergy, "TotalEnergy")

			returnEnergy, err := c.ReturnEnergy()
			require.NoError(t, err)
			assert.Equal(t, tc.ret, returnEnergy, "ReturnEnergy")
		})
	}
}

// a failed read yields the zero status, whose ret_aenergy register is nil too
func TestSwitchEnergyReadError(t *testing.T) {
	c := &gen2{
		methods: []string{"Switch.GetStatus"},
		switchstatus: util.ResettableCached(func() (Gen2SwitchStatus, error) {
			return Gen2SwitchStatus{}, errors.New("offline")
		}, time.Minute),
	}

	require.NotPanics(t, func() {
		total, err := c.TotalEnergy()
		require.Error(t, err)
		assert.Zero(t, total)

		ret, err := c.ReturnEnergy()
		require.Error(t, err)
		assert.Zero(t, ret)
	})
}
