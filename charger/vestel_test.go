package charger

import (
	"testing"

	"github.com/andig/evcc/api"
)
// todo vestel!
func TestVestelLegacy(t *testing.T) {
	wbc, err := NewVestelFromConfig(map[string]interface{}{"legacy": true})
	if err != nil {
		t.Error(err)
	}

	wb, ok := wbc.(*Vestel)
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

func TestVestelEx(t *testing.T) {
	wbc, err := NewVestelFromConfig(map[string]interface{}{"legacy": false})
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
