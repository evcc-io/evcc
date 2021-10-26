package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/vehicle/bluelink"
)

func TestHyundai(t *testing.T) {
	id := hyundaiConfig().CCSPApplicationID
	if _, ok := bluelink.Stamps[id]; !ok {
		t.Fatal("missing stamps:", id)
	}
}
