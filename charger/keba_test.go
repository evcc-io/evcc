package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestKeba(t *testing.T) {
	var wb api.Charger = NewKeba("foo", RFID{}, 0)

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing MeterEnergy interface")
	}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrents interface")
	}

	if _, ok := wb.(api.ChargeRater); !ok {
		t.Error("missing ChargeRater interface")
	}
}
