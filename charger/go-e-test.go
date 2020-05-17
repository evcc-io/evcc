package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

// TestGoE tests interfaces
func TestGoE(t *testing.T) {
	var wb api.Charger = NewGoE("foo")

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrents interface")
	}

	if _, ok := wb.(api.ChargeRater); !ok {
		t.Error("missing ChargeRater interface")
	}
}
