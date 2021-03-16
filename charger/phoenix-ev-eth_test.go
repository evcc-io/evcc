package charger

import (
	"testing"

	"github.com/mark-sch/evcc/api"
)

func TestPhoenixEVEthDecorators(t *testing.T) {
	wb, err := NewPhoenixEVEthFromConfig(map[string]interface{}{
		"meter": map[string]interface{}{
			"power":    true,
			"energy":   true,
			"currents": true,
		},
	})
	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.Meter); !ok {
		t.Error("missing Meter api")
	}

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing MeterEnergy api")
	}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrent api")
	}

	if _, ok := wb.(api.ChargeTimer); !ok {
		t.Error("missing ChargeTimer api")
	}
}
