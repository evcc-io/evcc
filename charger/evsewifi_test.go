package charger

import (
	"testing"

	"github.com/mark-sch/evcc/api"
)

func TestEvseWifi(t *testing.T) {
	wb, err := NewEVSEWifiFromConfig(map[string]interface{}{
		"meter": map[string]interface{}{
			"power":    true,
			"energy":   true,
			"currents": true,
		},
	})

	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing api.MeterEnergy")
	}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing api.MeterCurrent")
	}

	if _, ok := wb.(api.ChargeTimer); !ok {
		t.Error("missing ChargeTimer api")
	}
}
