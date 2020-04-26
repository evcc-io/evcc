package charger

import "testing"

func TestWallbe(t *testing.T) {
	wb := NewWallbeFromConfig(nil, nil)

	if wb.factor != 10 {
		t.Errorf("invalid factor: %d", wb.factor)
	}

	wb = NewWallbeFromConfig(nil, map[string]interface{}{"legacy": true})

	if wb.factor != 1 {
		t.Errorf("invalid factor: %d", wb.factor)
	}
}
