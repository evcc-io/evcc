package charger

import (
	"testing"

	"github.com/andig/evcc/util"
)

func TestWallbe(t *testing.T) {
	wb := NewWallbeFromConfig(util.NewLogger(""), nil)

	if wb.factor != 10 {
		t.Errorf("invalid factor: %d", wb.factor)
	}

	wb = NewWallbeFromConfig(util.NewLogger(""), map[string]interface{}{"legacy": true})

	if wb.factor != 1 {
		t.Errorf("invalid factor: %d", wb.factor)
	}
}
