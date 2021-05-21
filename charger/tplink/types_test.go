package tplink

import (
	"encoding/json"
	"testing"
)

func TestUnmarshalTPLinkSystemResponses(t *testing.T) {
	var sysresp SystemResponse

	// Test set_relay_state response
	jsonstr := `{"system":{"set_relay_state":{"err_code":0}}}`
	if err := json.Unmarshal([]byte(jsonstr), &sysresp); err != nil {
		t.Error(err)
	}
	if sysresp.System.SetRelayState.ErrCode != 0 {
		t.Error("SetRelayState.ErrCode")
	}

	// Test get_sysinfo response
	jsonstr = `{"system":{"get_sysinfo":{"err_code":0,"sw_ver":"1.2.6 Build 200727 Rel.120821","hw_ver":"1.0","type":"IOT.SMARTPLUGSWITCH","model":"HS110(EU)","mac":"50:C7:BF:42:60:9B","deviceId":"80068B6B73AAD8C4A4D0F7B9AB5F8B1B1838EEAC","hwId":"45E29DA8382494D2E82688B52A0B2EB5","fwId":"00000000000000000000000000000000","oemId":"3D341ECE302C0642C99E31CE2430544B","alias":"evcc-charger","dev_name":"Wi-Fi Smart Plug With Energy Monitoring","icon_hash":"","relay_state":1,"on_time":110,"active_mode":"schedule","feature":"TIM:ENE","updating":0,"rssi":-54,"led_off":0,"latitude":49.817090,"longitude":9.056194}}}`
	if err := json.Unmarshal([]byte(jsonstr), &sysresp); err != nil {
		t.Error(err)
	}
	if sysresp.System.GetSysinfo.ErrCode != 0 {
		t.Error("GetSysinfo.ErrCode")
	}
	if sysresp.System.GetSysinfo.SwVer != "1.2.6 Build 200727 Rel.120821" {
		t.Error("GetSysinfo.SwVer")
	}
	if sysresp.System.GetSysinfo.Model != "HS110(EU)" {
		t.Error("GetSysinfo.Model")
	}
	if sysresp.System.GetSysinfo.Alias != "evcc-charger" {
		t.Error("GetSysinfo.Alias")
	}
	if sysresp.System.GetSysinfo.RelayState != 1 {
		t.Error("GetSysinfo.RelayState")
	}
	if sysresp.System.GetSysinfo.Feature != "TIM:ENE" {
		t.Error("GetSysinfo.Feature")
	}

	// Test 1st emeter generation get_realtime response
	var emeresp EmeterResponse
	jsonstr = `{"emeter":{"get_realtime":{"current":0.033759,"voltage":234.824322,"power":3.121391,"total":0.015000,"err_code":0}}}`
	if err := json.Unmarshal([]byte(jsonstr), &emeresp); err != nil {
		t.Error(err)
	}
	if emeresp.Emeter.GetRealtime.Current != 0.033759 {
		t.Error("GetRealtime.Current")
	}
	if emeresp.Emeter.GetRealtime.Voltage != 234.824322 {
		t.Error("GetRealtime.Voltage")
	}
	if emeresp.Emeter.GetRealtime.Power != 3.121391 {
		t.Error("GetRealtime.Power")
	}
	if emeresp.Emeter.GetRealtime.Total != 0.015000 {
		t.Error("GetRealtime.Total")
	}
	if emeresp.Emeter.GetRealtime.ErrCode != 0 {
		t.Error("GetRealtime.ErrCode")
	}

	// Test 2nd emeter generation get_realtime response
	var emeresp2 EmeterResponse
	jsonstr = `{"emeter":{"get_realtime":{"voltage_mv":237119,"current_ma":218,"power_mw":31259,"total_wh":107,"err_code":0}}}`
	if err := json.Unmarshal([]byte(jsonstr), &emeresp2); err != nil {
		t.Error(err)
	}
	if emeresp2.Emeter.GetRealtime.CurrentMa != 218 {
		t.Error("GetRealtime.CurrentMa")
	}
	if emeresp2.Emeter.GetRealtime.VoltageMv != 237119 {
		t.Error("GetRealtime.VoltageMv")
	}
	if emeresp2.Emeter.GetRealtime.PowerMw != 31259 {
		t.Error("GetRealtime.PowerMw")
	}
	if emeresp2.Emeter.GetRealtime.TotalWh != 107 {
		t.Error("GetRealtime.TotalWh")
	}
	if emeresp2.Emeter.GetRealtime.ErrCode != 0 {
		t.Error("GetRealtime.ErrCode")
	}

	// Test 1st emeter generation get_daystat response
	var dstatresp DayStatResponse
	jsonstr = `{"emeter":{"get_daystat":{"day_list":[{"year":2021,"month":4,"day":22,"energy":0},{"year":2021,"month":4,"day":23,"energy":0.016000},{"year":2021,"month":4,"day":24,"energy":0.020000},{"year":2021,"month":4,"day":25,"energy":0.005000},{"year":2021,"month":4,"day":26,"energy":0.245000}],"err_code":0}}}`
	if err := json.Unmarshal([]byte(jsonstr), &dstatresp); err != nil {
		t.Error(err)
	}
	if dstatresp.Emeter.GetDaystat.DayList[len(dstatresp.Emeter.GetDaystat.DayList)-1].Year != 2021 {
		t.Error("GetDaystat.DayList[last].Year")
	}
	if dstatresp.Emeter.GetDaystat.DayList[len(dstatresp.Emeter.GetDaystat.DayList)-1].Month != 4 {
		t.Error("GetDaystat.DayList[last].Month")
	}
	if dstatresp.Emeter.GetDaystat.DayList[len(dstatresp.Emeter.GetDaystat.DayList)-1].Day != 26 {
		t.Error("GetDaystat.DayList[last].Day")
	}
	if dstatresp.Emeter.GetDaystat.DayList[len(dstatresp.Emeter.GetDaystat.DayList)-1].Energy != 0.245 {
		t.Error("GetDaystat.DayList[last].Energy")
	}
	if dstatresp.Emeter.GetDaystat.ErrCode != 0 {
		t.Error("GetDaystat.ErrCode")
	}

	// Test 2nd emeter generation get_daystat response
	var dstatresp2 DayStatResponse
	jsonstr = `{"emeter":{"get_daystat":{"day_list":[{"year":2021,"month":4,"day":18,"energy_wh":3},{"year":2021,"month":4,"day":19,"energy_wh":0},{"year":2021,"month":4,"day":20,"energy_wh":2029},{"year":2021,"month":4,"day":21,"energy_wh":1201},{"year":2021,"month":4,"day":22,"energy_wh":0},{"year":2021,"month":4,"day":23,"energy_wh":1059}],"err_code":0}}}`
	if err := json.Unmarshal([]byte(jsonstr), &dstatresp2); err != nil {
		t.Error(err)
	}
	if dstatresp2.Emeter.GetDaystat.DayList[len(dstatresp2.Emeter.GetDaystat.DayList)-1].Year != 2021 {
		t.Error("GetDaystat.DayList[last].Year")
	}
	if dstatresp2.Emeter.GetDaystat.DayList[len(dstatresp2.Emeter.GetDaystat.DayList)-1].Month != 4 {
		t.Error("GetDaystat.DayList[last].Month")
	}
	if dstatresp2.Emeter.GetDaystat.DayList[len(dstatresp2.Emeter.GetDaystat.DayList)-1].Day != 23 {
		t.Error("GetDaystat.DayList[last].Day")
	}
	if dstatresp2.Emeter.GetDaystat.DayList[len(dstatresp2.Emeter.GetDaystat.DayList)-1].EnergyWh != 1059 {
		t.Error("GetDaystat.DayList[last].EnergyWh")
	}
	if dstatresp2.Emeter.GetDaystat.ErrCode != 0 {
		t.Error("GetDaystat.ErrCode")
	}
}
