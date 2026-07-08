package server

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// TestPrometheus exercises the exporter end-to-end through a real registry
// Gather() and text encoding, covering scalar, bool, string (ignored) and
// 3-phase values. It also confirms that mixed label dimensions under the same
// metric name (per-phase values plus the aggregated sum) are exported without
// tripping the client_golang consistency checks.
func TestPrometheus(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := NewPrometheus(reg)

	// scalar
	p.record("gridPower", 1543, prometheus.Labels{})

	// 3-phase value -> per-phase (phase label) + aggregated sum (no phase label),
	// both under the SAME metric name evcc_loadpoint_charge_power
	p.record("loadpoint_chargePower", [3]float64{2453.3, 2000, 1900}, prometheus.Labels{"loadpoint": "Garage"})

	// bool + string(ignored)
	p.record("loadpoint_charging", true, prometheus.Labels{"loadpoint": "Garage"})
	p.record("loadpoint_mode", "pv", prometheus.Labels{"loadpoint": "Garage"})

	// nested struct field -> flattened key "battery_Soc"; separators must
	// collapse into a single underscore (evcc_battery_soc, not evcc_battery__soc)
	p.record("battery", struct{ Soc float64 }{Soc: 92}, prometheus.Labels{})

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather() returned error (consistency check failed): %v", err)
	}

	var b strings.Builder
	enc := expfmt.NewEncoder(&b, expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			t.Fatalf("encode %s: %v", mf.GetName(), err)
		}
	}
	out := b.String()

	// strings must not be exported
	if strings.Contains(out, "evcc_loadpoint_mode") {
		t.Error("string value was exported but should be skipped")
	}

	for _, want := range []string{
		`evcc_grid_power 1543`,
		`evcc_battery_soc 92`,
		`evcc_loadpoint_charging{loadpoint="Garage"} 1`,
		`evcc_loadpoint_charge_power{loadpoint="Garage",phase="1"} 2453.3`,
		`evcc_loadpoint_charge_power{loadpoint="Garage",phase="2"} 2000`,
		`evcc_loadpoint_charge_power{loadpoint="Garage",phase="3"} 1900`,
		`evcc_loadpoint_charge_power{loadpoint="Garage"} 6353.3`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing expected metric line:\n  %s\n--- full output ---\n%s", want, out)
		}
	}
}
