package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestNRGKickBLE(t *testing.T) {
	var wb api.Charger = &NRGKickBLE{}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrents interface")
	}
}
