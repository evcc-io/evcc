package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestNRGKickBLE(t *testing.T) {
	var wb api.Charger
	wb = NewNRGKickBLE("foo", "bar", 0)

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrents interface")
	}

	if _, ok := wb.(api.ChargeRater); !ok {
		t.Error("missing ChargeRater interface")
	}
}
