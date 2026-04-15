package tasmota

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test StatusSNS response of all known Tasmota flavours
func TestUnmarshalStatusSNSResponse(t *testing.T) {
	var res StatusSNSResponse

	// Test cases for #6082
	jsonstr := `{"StatusSNS":{"Time":"2023-02-05T20:31:48","ENERGY":{"TotalStartTime":"2023-02-05T11:04:13","Total":1290.3960,"Yesterday":0.8540,"Today":0.1730,"Power":47.11,"ApparentPower":0.0,"ReactivePower":0.0,"Factor":0.00,"Voltage":0.00,"Current":0.000}}}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
	if power, err := res.StatusSNS.Energy.Power.Value(1); err == nil && power != 47.11 {
		t.Error("StatusSNS.Energy.Power.Value(1) != 47.11")
	}

	// Test case for #5731
	jsonstr = `{"StatusSNS":{"Time":"2023-01-09T18:57:39","Switch1":"ON","Switch2":"OFF","ANALOG":{"Temperature":49.6},"ENERGY":{"TotalStartTime":"2023-01-09T13:59:15","Total":0.077,"Yesterday":0.000,"Today":0.077,"Power":[8.15,0],"ApparentPower":[5,0],"ReactivePower":[4,0],"Factor":[0.00,0.00],"Frequency":50,"Voltage":237,"Current":[0.000,0.000]},"TempUnit":"C"}}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
	if power, err := res.StatusSNS.Energy.Power.Value(1); err == nil && power != 8.15 {
		t.Error("StatusSNS.Energy.Power.Value(1) != 8.15")
	}

	// Test case for #3787
	jsonstr = `{"StatusSNS":{"Time":"2022-07-07T13:01:11","HTU21":{"Temperature":25.2,"Humidity":45.5},"SML":{"Total_in":34507.4761,"Total_out":14737.1422,"Power_curr":-894,"Meter_number":"0901454d48000000000"},"Gas":{"Count":1.84},"TempUnit":"C"}}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
	assert.Equal(t, float64(-894), *res.StatusSNS.SML.PowerCurr)

	// Test case for #26857
	jsonstr = `{"StatusSNS":{"Time":"2026-01-21T11:07:31","SML":{"Total_in":2687.3687,"Total_out":582.4569,"Power_curr":-36,"Meter_Id":"0a01454652240487bd2a","power_l1":60,"power_l2":-111,"power_l3":14,"voltage_l1":235.3,"voltage_l2":235.7,"voltage_l3":235.7,"current_l1":0.98,"current_l2":1.47,"current_l3":1.18,"phase_angle_L2_L1":238,"phase_angle_L3_L1":118,"phase_angle_L1":23.0,"phase_angle_L2":108.0,"phase_angle_L3":350.0,"Frequenz":49.9}}}`
	require.NoError(t, json.Unmarshal([]byte(jsonstr), &res))
	assert.Equal(t, float64(-36), *res.StatusSNS.SML.PowerCurr)
	assert.Equal(t, float64(60), *res.StatusSNS.SML.PowerL1)
	assert.Equal(t, float64(-111), *res.StatusSNS.SML.PowerL2)
	assert.Equal(t, float64(14), *res.StatusSNS.SML.PowerL3)
	assert.Equal(t, float64(235.3), *res.StatusSNS.SML.VoltageL1)
	assert.Equal(t, float64(235.7), *res.StatusSNS.SML.VoltageL2)
	assert.Equal(t, float64(235.7), *res.StatusSNS.SML.VoltageL3)
	assert.Equal(t, float64(0.98), *res.StatusSNS.SML.CurrentL1)
	assert.Equal(t, float64(1.47), *res.StatusSNS.SML.CurrentL2)
	assert.Equal(t, float64(1.18), *res.StatusSNS.SML.CurrentL3)
}
