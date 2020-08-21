package charger

import (
	"testing"
)

func TestWallbe(t *testing.T) {
	wbc, err := NewWallbeFromConfig(nil)
	if err != nil {
		t.Error(err)
	}

	wb, ok := wbc.(*Wallbe)
	if !ok {
		t.Error("unexpected type")
	}

	if wb.factor != 10 {
		t.Errorf("invalid factor: %d", wb.factor)
	}

	wbc, err = NewWallbeFromConfig(map[string]interface{}{"legacy": true})
	if err != nil {
		t.Error(err)
	}

	wb, ok = wbc.(*Wallbe)
	if !ok {
		t.Error("unexpected type")
	}

	if wb.factor != 1 {
		t.Errorf("invalid factor: %d", wb.factor)
	}
}
