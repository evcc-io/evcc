package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestNRGKickConnect(t *testing.T) {
	var wb api.Charger
	wb, err := NewNRGKickConnect("foo", "bar", "baz")
	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing MeterEnergy interface")
	}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrent interface")
	}
}
