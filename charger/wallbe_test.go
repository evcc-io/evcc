package charger

import (
	"testing"

	"github.com/mark-sch/evcc/api"
)

func TestWallbeLegacy(t *testing.T) {
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

	if _, ok = wbc.(api.ChargeTimer); !ok {
		t.Error("missing charge timer api")
	}

	if _, ok = wbc.(api.Diagnosis); !ok {
		t.Error("missing diagnosis api")
	}
}

func TestWallbeEx(t *testing.T) {
	wbc, err := NewWallbeFromConfig(map[string]interface{}{"legacy": false})
	if err != nil {
		t.Error(err)
	}

	if _, ok := wbc.(api.ChargerEx); !ok {
		t.Error("missing ChargerEx api")
	}

	if _, ok := wbc.(api.ChargeTimer); !ok {
		t.Error("missing ChargeTimer api")
	}

	if _, ok := wbc.(api.Diagnosis); !ok {
		t.Error("missing Diagnosis api")
	}
}
