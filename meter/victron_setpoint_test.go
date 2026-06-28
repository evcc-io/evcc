package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
)

// TestVictronGridSetpoint verifies the opt-in batterycontrol: gridsetpoint variant renders,
// parses, and wires the BatteryHoldPower capability instead of the limit-SOC BatteryController.
func TestVictronGridSetpoint(t *testing.T) {
	values := map[string]any{
		"template":          "victron-energy",
		"usage":             "battery",
		"host":              "localhost",
		"batterycontrol":    "gridsetpoint",
		"capacity":          32,
		"minsoc":            20,
		"maxsoc":            100,
		"maxchargepower":    6500,
		"maxdischargepower": 10000,
	}

	m, err := NewFromConfig(t.Context(), "template", values)
	if err != nil {
		t.Fatal(err)
	}

	if !api.HasCap[api.BatteryHoldPower](m) {
		t.Error("expected BatteryHoldPower capability for gridsetpoint variant")
	}
	if api.HasCap[api.BatteryController](m) {
		t.Error("did not expect BatteryController (limit-SOC) for gridsetpoint variant")
	}

	// default limitsoc variant must still wire the limit-SOC BatteryController
	values["batterycontrol"] = "limitsoc"
	m, err = NewFromConfig(t.Context(), "template", values)
	if err != nil {
		t.Fatal(err)
	}
	if !api.HasCap[api.BatteryController](m) {
		t.Error("expected BatteryController for default limitsoc variant")
	}
	if api.HasCap[api.BatteryHoldPower](m) {
		t.Error("did not expect BatteryHoldPower for limitsoc variant")
	}
}
