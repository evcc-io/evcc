package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
)

func TestWallbeLegacy(t *testing.T) {
	wbc, err := NewWallbeFromConfig(map[string]interface{}{"legacy": true})
	if err != nil {
		t.Error(err)
	}

	if _, ok := wbc.(api.ChargeTimer); !ok {
		t.Error("missing ChargeTimer api")
	}

	if _, ok := wbc.(api.Diagnosis); !ok {
		t.Error("missing Diagnosis api")
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
