package charger

import (
	"testing"
)

func TestWallbe(t *testing.T) {
	wb, err := NewWallbeFromConfig(nil)
	if err != nil {
		t.Error(err)
	}

	if wb.factor != 10 {
		t.Errorf("invalid factor: %d", wb.factor)
	}

	wb, err = NewWallbeFromConfig(map[string]interface{}{"legacy": true})
	if err != nil {
		t.Error(err)
	}

	if wb.factor != 1 {
		t.Errorf("invalid factor: %d", wb.factor)
	}
}
