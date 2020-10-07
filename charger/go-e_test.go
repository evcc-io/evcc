package charger

import (
	"testing"

	"github.com/mark-sch/evcc/api"
)

// TestGoE tests interfaces
func TestGoE(t *testing.T) {
	var wb api.Charger
	wb, err := NewGoE("foo", "bar", 0)
	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrent interface")
	}

	if _, ok := wb.(api.ChargeRater); !ok {
		t.Error("missing ChargeRater interface")
	}
}
