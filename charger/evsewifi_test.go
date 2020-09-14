package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestEvseWifi(t *testing.T) {
	evse, err := NewEVSEWifiFromConfig(map[string]interface{}{
		"uri": "foo",
		"meter": map[string]interface{}{
			"power":    true,
			"energy":   true,
			"currents": true,
		},
	})

	if err != nil {
		t.Error(err)
	}

	if _, ok := evse.(api.MeterEnergy); !ok {
		t.Error("missing api.MeterEnergy")
	}

	if _, ok := evse.(api.MeterCurrent); !ok {
		t.Error("missing api.MeterCurrent")
	}
}
