package charger

import (
	"testing"

	"github.com/mark-sch/evcc/api"
)

func TestOpenWBDecorators(t *testing.T) {
	// host not reachable
	wb, _ := NewOpenWBFromConfig(map[string]interface{}{
		"broker": "192.0.2.2",
	})

	if _, ok := wb.(api.Meter); !ok {
		t.Error("missing Meter api")
	}

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing MeterEnergy api")
	}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrent api")
	}
}
