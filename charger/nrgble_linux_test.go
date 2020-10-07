package charger

import (
	"testing"

	"github.com/mark-sch/evcc/api"
)

func TestNRGKickBLE(t *testing.T) {
	var wb api.Charger = &NRGKickBLE{}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrent interface")
	}
}
