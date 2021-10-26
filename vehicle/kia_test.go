package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/vehicle/bluelink"
)

func TestKia(t *testing.T) {
	id := kiaConfig().CCSPApplicationID
	if _, ok := bluelink.Stamps[id]; !ok {
		t.Fatal("missing stamps:", id)
	}
}
