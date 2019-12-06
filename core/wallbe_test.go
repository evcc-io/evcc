package core

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestWallbe(t *testing.T) {
	var c api.Charger = NewWallbe("192.168.0.8:502")

	if _, ok := c.(api.ChargeController); !ok {
		t.Error("not a charge controller")
	}
}
