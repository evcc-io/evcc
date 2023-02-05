package tasmota

import (
	"encoding/json"
	"testing"
)

// Test StatusSNS response of all known Tasmota flavours
func TestUnmarshalStatusSNSResponse(t *testing.T) {
	var res StatusSNSResponse

	// Test cases for #6082
	jsonstr := `{"StatusSNS":{"Time":"2023-02-05T20:31:48","ENERGY":{"TotalStartTime":"2023-02-05T11:04:13","Total":1290.3960,"Yesterday":0.8540,"Today":0.1730,"Power":47.11,"ApparentPower":0.0,"ReactivePower":0.0,"Factor":0.00,"Voltage":0.00,"Current":0.000}}}`
	if err := json.Unmarshal([]byte(jsonstr), &res); err != nil {
		t.Error(err)
	}
	if power, err := res.StatusSNS.Energy.Power.Channel(1); err == nil && power != 47.11 {
		t.Error("StatusSNS.Energy.Power.Channel(1) != 47.11")
	}

	// Test case for #5731
	jsonstr = `{"StatusSNS":{"Time":"2023-01-09T18:57:39","Switch1":"ON","Switch2":"OFF","ANALOG":{"Temperature":49.6},"ENERGY":{"TotalStartTime":"2023-01-09T13:59:15","Total":0.077,"Yesterday":0.000,"Today":0.077,"Power":[8.15,0],"ApparentPower":[5,0],"ReactivePower":[4,0],"Factor":[0.00,0.00],"Frequency":50,"Voltage":237,"Current":[0.000,0.000]},"TempUnit":"C"}}`
	if err := json.Unmarshal([]byte(jsonstr), &res); err != nil {
		t.Error(err)
	}
	if power, err := res.StatusSNS.Energy.Power.Channel(1); err == nil && power != 8.15 {
		t.Error("StatusSNS.Energy.Power.Channel(1) != 8.15")
	}

	// Test case for #3787
	jsonstr = `{"StatusSNS":{"Time":"2022-07-07T13:01:11","HTU21":{"Temperature":25.2,"Humidity":45.5},"SML":{"Total_in":34507.4761,"Total_out":14737.1422,"Power_curr":-894,"Meter_number":"0901454d48000000000"},"Gas":{"Count":1.84},"TempUnit":"C"}}`
	if err := json.Unmarshal([]byte(jsonstr), &res); err != nil {
		t.Error(err)
	}
	if res.StatusSNS.SML.PowerCurr != -894 {
		t.Error("res.StatusSNS.SML.PowerCurr != -894")
	}
}
