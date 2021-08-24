package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
)

func TestNRGKickBLE(t *testing.T) {
	var wb api.Charger = &NRGKickBLE{}

	if _, ok := wb.(api.MeterCurrent); !ok {
		t.Error("missing MeterCurrent interface")
	}
}
