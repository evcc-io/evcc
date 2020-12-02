package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)

func TestWallbe(t *testing.T) {
	wbc, err := NewWallbeFromConfig(map[string]interface{}{"legacy": true})
	if err != nil {
		t.Error(err)
	}

	wb, ok := wbc.(*Wallbe)
	if !ok {
		t.Error("unexpected type")
	}

	if wb.factor != 1 {
		t.Errorf("invalid factor: %d", wb.factor)
	}

	wbc, err = NewWallbeFromConfig(map[string]interface{}{"legacy": false})
	if err != nil {
		t.Error(err)
	}

	if _, ok = wbc.(api.ChargerEx); !ok {
		t.Error("unexpected type")
	}
}
