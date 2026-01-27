//go:build integration

package ecoflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Integration tests for EcoFlow API
// Run with: go test -tags=integration -v ./meter/ecoflow/...
//
// Required environment variables:
//   ECOFLOW_SN         - Device serial number
//   ECOFLOW_ACCESS_KEY - API access key
//   ECOFLOW_SECRET_KEY - API secret key
//   ECOFLOW_URI        - API URI (optional, defaults to https://api-e.ecoflow.com)
//   ECOFLOW_DEVICE     - Device type: "stream" or "powerstream" (default: stream)

func skipIfNoCredentials(t *testing.T) {
	if os.Getenv("ECOFLOW_SN") == "" || os.Getenv("ECOFLOW_ACCESS_KEY") == "" || os.Getenv("ECOFLOW_SECRET_KEY") == "" {
		t.Skip("Skipping integration test: ECOFLOW_SN, ECOFLOW_ACCESS_KEY, ECOFLOW_SECRET_KEY required")
	}
}

func getTestConfig() map[string]interface{} {
	uri := os.Getenv("ECOFLOW_URI")
	if uri == "" {
		uri = "https://api-e.ecoflow.com"
	}

	return map[string]interface{}{
		"uri":       uri,
		"sn":        os.Getenv("ECOFLOW_SN"),
		"accessKey": os.Getenv("ECOFLOW_ACCESS_KEY"),
		"secretKey": os.Getenv("ECOFLOW_SECRET_KEY"),
		"usage":     "battery",
		"cache":     "1s",
	}
}

func isStreamDevice() bool {
	return os.Getenv("ECOFLOW_DEVICE") != "powerstream"
}

// testDevice holds common test infrastructure
type testDevice struct {
	helper *request.Helper
	uri    string
	sn     string
}

func newTestDevice(t *testing.T) *testDevice {
	config := getTestConfig()

	uri := config["uri"].(string)
	sn := config["sn"].(string)
	accessKey := config["accessKey"].(string)
	secretKey := config["secretKey"].(string)

	log := util.NewLogger("ecoflow-test")
	helper := request.NewHelper(log)
	helper.Client.Transport = NewAuthTransport(helper.Client.Transport, accessKey, secretKey)

	return &testDevice{
		helper: helper,
		uri:    uri,
		sn:     sn,
	}
}

func (d *testDevice) quotaURL() string {
	return fmt.Sprintf("%s/iot-open/sign/device/quota/all?sn=%s", d.uri, d.sn)
}

// =============================================================================
// READ TESTS
// =============================================================================

// TestIntegration_ReadBatterySOC tests reading the battery state of charge
func TestIntegration_ReadBatterySOC(t *testing.T) {
	skipIfNoCredentials(t)

	config := getTestConfig()
	config["usage"] = "battery"

	var meter api.Meter
	var err error

	if isStreamDevice() {
		meter, err = NewStreamFromConfig(context.Background(), config)
	} else {
		meter, err = NewPowerStreamFromConfig(context.Background(), config)
	}

	if err != nil {
		t.Fatalf("Failed to create meter: %v", err)
	}

	// Test CurrentPower (battery power)
	power, err := meter.CurrentPower()
	if err != nil {
		t.Fatalf("Failed to read battery power: %v", err)
	}

	t.Logf("✅ Battery Power: %.2f W (positive=charging, negative=discharging)", power)

	// Test SOC if battery interface
	if battery, ok := meter.(api.Battery); ok {
		soc, err := battery.Soc()
		if err != nil {
			t.Fatalf("Failed to read SOC: %v", err)
		}

		if soc < 0 || soc > 100 {
			t.Errorf("SOC out of range: %.2f%% (expected 0-100)", soc)
		}

		t.Logf("✅ Battery SOC: %.1f%%", soc)
	}
}

// TestIntegration_ReadChargingPower tests reading the charging power
func TestIntegration_ReadChargingPower(t *testing.T) {
	skipIfNoCredentials(t)

	device := newTestDevice(t)
	uri := device.quotaURL()

	if isStreamDevice() {
		var res response[StreamData]
		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to fetch data: %v", err)
		}

		if res.Code != "0" {
			t.Fatalf("API error: %s - %s", res.Code, res.Message)
		}

		data := res.Data
		t.Logf("✅ Raw Stream Data:")
		t.Logf("   PV Power (Ladegeschwindigkeit PV): %.2f W", data.PowGetPvSum)
		t.Logf("   Grid Power: %.2f W", data.PowGetSysGrid)
		t.Logf("   Battery Power: %.2f W (negative=discharge, positive=charge)", data.PowGetBpCms)
		t.Logf("   Battery SOC: %.1f%%", data.CmsBattSoc)
		t.Logf("   System Load: %.2f W", data.PowGetSysLoad)
		t.Logf("   Relay AC1: %v", data.Relay2Onoff)
		t.Logf("   Relay AC2: %v", data.Relay3Onoff)
	} else {
		var res response[PowerStreamData]
		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to fetch data: %v", err)
		}

		if res.Code != "0" {
			t.Fatalf("API error: %s - %s", res.Code, res.Message)
		}

		data := res.Data
		t.Logf("✅ Raw PowerStream Data:")
		t.Logf("   PV1 Power: %.2f W", data.Pv1InputWatts)
		t.Logf("   PV2 Power: %.2f W", data.Pv2InputWatts)
		t.Logf("   Total PV (Ladegeschwindigkeit): %.2f W", data.Pv1InputWatts+data.Pv2InputWatts)
		t.Logf("   Battery Power: %.2f W (positive=discharge, negative=charge)", data.BatWatts)
		t.Logf("   Inverter Output: %.2f W", data.InvOutputWatts)
		t.Logf("   Battery SOC: %d%%", data.BatSoc)
	}
}

// TestIntegration_ReadDischargePower tests reading the discharge power
func TestIntegration_ReadDischargePower(t *testing.T) {
	skipIfNoCredentials(t)

	config := getTestConfig()
	config["usage"] = "battery"

	var meter api.Meter
	var err error

	if isStreamDevice() {
		meter, err = NewStreamFromConfig(context.Background(), config)
	} else {
		meter, err = NewPowerStreamFromConfig(context.Background(), config)
	}

	if err != nil {
		t.Fatalf("Failed to create meter: %v", err)
	}

	power, err := meter.CurrentPower()
	if err != nil {
		t.Fatalf("Failed to read power: %v", err)
	}

	// Interpret based on evcc convention
	if power > 0 {
		t.Logf("✅ Entladegeschwindigkeit: 0 W (Batterie lädt mit %.2f W)", power)
	} else if power < 0 {
		t.Logf("✅ Entladegeschwindigkeit: %.2f W", -power)
	} else {
		t.Logf("✅ Batterie idle (weder Laden noch Entladen)")
	}
}

// =============================================================================
// CONTROL TESTS
// =============================================================================

// setRelayRequest for Stream relay control
type setRelayRequest struct {
	SN     string `json:"sn"`
	Params struct {
		Relay2Onoff *bool `json:"relay2Onoff,omitempty"`
		Relay3Onoff *bool `json:"relay3Onoff,omitempty"`
	} `json:"params"`
}

type setResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// setRelay controls AC relay switches on Stream devices
func setRelay(device *testDevice, relay int, state bool) error {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", device.uri)

	req := setRelayRequest{SN: device.sn}
	if relay == 1 {
		req.Params.Relay2Onoff = &state
	} else if relay == 2 {
		req.Params.Relay3Onoff = &state
	} else {
		return fmt.Errorf("invalid relay number: %d (must be 1 or 2)", relay)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPut, uri, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	var res setResponse
	if err := device.helper.DoJSON(httpReq, &res); err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("API error: %s - %s", res.Code, res.Message)
	}

	return nil
}

// setPowerStreamWattsRequest for PowerStream control
type setPowerStreamWattsRequest struct {
	SN      string `json:"sn"`
	CmdCode string `json:"cmdCode"`
	Params  struct {
		PermanentWatts int `json:"permanentWatts"`
	} `json:"params"`
}

// setPowerStreamWatts sets the custom load power for PowerStream
func setPowerStreamWatts(device *testDevice, watts int) error {
	uri := fmt.Sprintf("%s/iot-open/sign/device/quota", device.uri)

	req := setPowerStreamWattsRequest{
		SN:      device.sn,
		CmdCode: "WN511_SET_PERMANENT_WATTS_PACK",
	}
	req.Params.PermanentWatts = watts * 10

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPut, uri, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	var res setResponse
	if err := device.helper.DoJSON(httpReq, &res); err != nil {
		return err
	}

	if res.Code != "0" {
		return fmt.Errorf("API error: %s - %s", res.Code, res.Message)
	}

	return nil
}

// TestIntegration_ControlRelay tests enabling/disabling AC relays (Stream only)
func TestIntegration_ControlRelay(t *testing.T) {
	skipIfNoCredentials(t)

	if !isStreamDevice() {
		t.Skip("Relay control only available for Stream devices")
	}

	if os.Getenv("ECOFLOW_ALLOW_CONTROL") != "true" {
		t.Skip("Skipping control test: set ECOFLOW_ALLOW_CONTROL=true to enable")
	}

	device := newTestDevice(t)
	uri := device.quotaURL()

	// Read current state
	var res response[StreamData]
	if err := device.helper.GetJSON(uri, &res); err != nil {
		t.Fatalf("Failed to read state: %v", err)
	}

	originalRelay1 := res.Data.Relay2Onoff
	t.Logf("Current Relay AC1 state: %v", originalRelay1)

	// Toggle relay
	newState := !originalRelay1
	t.Logf("Setting Relay AC1 to: %v", newState)

	if err := setRelay(device, 1, newState); err != nil {
		t.Fatalf("Failed to set relay: %v", err)
	}

	// Wait for state change
	time.Sleep(2 * time.Second)

	// Verify change
	if err := device.helper.GetJSON(uri, &res); err != nil {
		t.Fatalf("Failed to read state after change: %v", err)
	}

	if res.Data.Relay2Onoff != newState {
		t.Errorf("Relay state not changed: expected %v, got %v", newState, res.Data.Relay2Onoff)
	} else {
		t.Logf("✅ Relay AC1 changed to: %v", res.Data.Relay2Onoff)
	}

	// Restore original state
	t.Logf("Restoring Relay AC1 to: %v", originalRelay1)
	if err := setRelay(device, 1, originalRelay1); err != nil {
		t.Errorf("Failed to restore relay state: %v", err)
	}

	t.Logf("✅ Relay control test completed")
}

// TestIntegration_ControlCharging tests controlling charging via relay (Stream)
func TestIntegration_ControlCharging(t *testing.T) {
	skipIfNoCredentials(t)

	if !isStreamDevice() {
		t.Skip("Charging control via relay only available for Stream devices")
	}

	if os.Getenv("ECOFLOW_ALLOW_CONTROL") != "true" {
		t.Skip("Skipping control test: set ECOFLOW_ALLOW_CONTROL=true to enable")
	}

	device := newTestDevice(t)

	t.Log("Note: Stream charging is controlled via grid connection and relay states")
	t.Log("To stop charging: disconnect from grid or disable relays")
	t.Log("To start charging: connect to grid with relays enabled")

	uri := device.quotaURL()
	var res response[StreamData]
	if err := device.helper.GetJSON(uri, &res); err != nil {
		t.Fatalf("Failed to read state: %v", err)
	}

	data := res.Data
	t.Logf("Current state:")
	t.Logf("  Battery Power: %.2f W", data.PowGetBpCms)
	t.Logf("  Relay AC1 (charging enabled): %v", data.Relay2Onoff)
	t.Logf("  Relay AC2 (charging enabled): %v", data.Relay3Onoff)

	// EcoFlow Stream: negative = discharge, positive = charge
	if data.PowGetBpCms > 0 {
		t.Logf("✅ Ladung aktiv mit %.2f W", data.PowGetBpCms)
	} else if data.PowGetBpCms < 0 {
		t.Logf("✅ Entladung aktiv mit %.2f W", -data.PowGetBpCms)
	} else {
		t.Logf("✅ Batterie idle")
	}
}

// TestIntegration_ControlDischarging tests controlling discharging
func TestIntegration_ControlDischarging(t *testing.T) {
	skipIfNoCredentials(t)

	if os.Getenv("ECOFLOW_ALLOW_CONTROL") != "true" {
		t.Skip("Skipping control test: set ECOFLOW_ALLOW_CONTROL=true to enable")
	}

	device := newTestDevice(t)

	if isStreamDevice() {
		t.Log("Stream device: Discharging is controlled by load on AC outputs")
		t.Log("Enable relay + connect load = discharge starts")
		t.Log("Disable relay = discharge stops")

		uri := device.quotaURL()
		var res response[StreamData]
		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to read state: %v", err)
		}

		data := res.Data
		t.Logf("Current discharge state:")
		t.Logf("  Battery Power: %.2f W (negative=discharge)", data.PowGetBpCms)
		t.Logf("  System Load: %.2f W", data.PowGetSysLoad)

	} else {
		t.Log("PowerStream device: Discharging controlled via permanentWatts")

		uri := device.quotaURL()
		var res response[PowerStreamData]
		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to read state: %v", err)
		}

		currentWatts := res.Data.PermanentWatts
		t.Logf("Current permanent watts: %d W", currentWatts)

		// Set discharge to 100W as test
		t.Log("Setting discharge to 100W...")
		if err := setPowerStreamWatts(device, 100); err != nil {
			t.Fatalf("Failed to set discharge power: %v", err)
		}

		time.Sleep(2 * time.Second)

		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to read state after change: %v", err)
		}

		t.Logf("✅ New permanent watts: %d W", res.Data.PermanentWatts)

		// Restore original
		t.Logf("Restoring to original: %d W", currentWatts)
		if err := setPowerStreamWatts(device, currentWatts); err != nil {
			t.Errorf("Failed to restore: %v", err)
		}
	}

	t.Log("✅ Discharge control test completed")
}

// =============================================================================
// SUMMARY TEST
// =============================================================================

// TestIntegration_FullStatus prints a complete status report
func TestIntegration_FullStatus(t *testing.T) {
	skipIfNoCredentials(t)

	device := newTestDevice(t)

	t.Log("======================================================")
	t.Logf("EcoFlow Device Status Report")
	t.Logf("Device: %s", device.sn)
	t.Logf("Type: %s", map[bool]string{true: "Stream", false: "PowerStream"}[isStreamDevice()])
	t.Log("======================================================")

	uri := device.quotaURL()

	if isStreamDevice() {
		var res response[StreamData]
		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to fetch: %v", err)
		}

		if res.Code != "0" {
			t.Fatalf("API error: %s - %s", res.Code, res.Message)
		}

		data := res.Data
		t.Log("")
		t.Log("📊 BATTERY STATUS")
		t.Logf("   Ladestand (SOC): %.1f%%", data.CmsBattSoc)

		// EcoFlow Stream: negative = discharge, positive = charge
		if data.PowGetBpCms > 0 {
			t.Logf("   Ladegeschwindigkeit: %.0f W", data.PowGetBpCms)
			t.Logf("   Entladegeschwindigkeit: 0 W")
		} else if data.PowGetBpCms < 0 {
			t.Logf("   Ladegeschwindigkeit: 0 W")
			t.Logf("   Entladegeschwindigkeit: %.0f W", -data.PowGetBpCms)
		} else {
			t.Logf("   Ladegeschwindigkeit: 0 W")
			t.Logf("   Entladegeschwindigkeit: 0 W")
		}

		t.Log("")
		t.Log("⚡ POWER FLOW")
		t.Logf("   PV: %.0f W", data.PowGetPvSum)
		t.Logf("   Grid: %.0f W", data.PowGetSysGrid)
		t.Logf("   Load: %.0f W", data.PowGetSysLoad)

		t.Log("")
		t.Log("🔌 RELAY STATUS")
		t.Logf("   AC1 (Ladung): %s", map[bool]string{true: "AN ✅", false: "AUS ❌"}[data.Relay2Onoff])
		t.Logf("   AC2 (Entladung): %s", map[bool]string{true: "AN ✅", false: "AUS ❌"}[data.Relay3Onoff])

	} else {
		var res response[PowerStreamData]
		if err := device.helper.GetJSON(uri, &res); err != nil {
			t.Fatalf("Failed to fetch: %v", err)
		}

		if res.Code != "0" {
			t.Fatalf("API error: %s - %s", res.Code, res.Message)
		}

		data := res.Data
		t.Log("")
		t.Log("📊 BATTERY STATUS")
		t.Logf("   Ladestand (SOC): %d%%", data.BatSoc)

		if data.BatWatts < 0 {
			t.Logf("   Ladegeschwindigkeit: %.0f W", -data.BatWatts)
			t.Logf("   Entladegeschwindigkeit: 0 W")
		} else if data.BatWatts > 0 {
			t.Logf("   Ladegeschwindigkeit: 0 W")
			t.Logf("   Entladegeschwindigkeit: %.0f W", data.BatWatts)
		} else {
			t.Logf("   Ladegeschwindigkeit: 0 W")
			t.Logf("   Entladegeschwindigkeit: 0 W")
		}

		t.Log("")
		t.Log("⚡ POWER FLOW")
		t.Logf("   PV1: %.0f W", data.Pv1InputWatts)
		t.Logf("   PV2: %.0f W", data.Pv2InputWatts)
		t.Logf("   Inverter Output: %.0f W", data.InvOutputWatts)
		t.Logf("   Permanent Watts: %d W", data.PermanentWatts)

		t.Log("")
		t.Log("⚙️ SETTINGS")
		t.Logf("   Inverter: %s", map[int]string{0: "AUS", 1: "AN"}[data.InvOnOff])
		t.Logf("   Lower Limit: %d%%", data.LowerLimit)
		t.Logf("   Upper Limit: %d%%", data.UpperLimit)
	}

	t.Log("")
	t.Log("✅ Status report completed")
}
