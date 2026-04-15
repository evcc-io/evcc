package charger

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
)

func TestEvseWifi(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"list":[{"useMeter":true, "alwaysActive":true}]}`)
	}))
	defer ts.Close()

	wb, err := NewEVSEWifiFromConfig(map[string]any{
		"uri": ts.URL,
		"meter": map[string]any{
			"power":    true,
			"energy":   true,
			"currents": true,
			"voltages": true,
		},
	})
	if err != nil {
		t.Error(err)
	}

	if _, ok := api.Cap[api.Meter](wb); !ok {
		t.Error("missing api.Meter")
	}

	if _, ok := api.Cap[api.MeterImport](wb); !ok {
		t.Error("missing api.MeterImport")
	}

	if _, ok := api.Cap[api.PhaseCurrents](wb); !ok {
		t.Error("missing api.PhaseCurrents")
	}

	if _, ok := api.Cap[api.PhaseVoltages](wb); !ok {
		t.Error("missing api.PhaseVoltages")
	}
}

func TestEvseWifiEx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"list":[{"actualCurrentMA":600, "alwaysActive":true}]}`)
	}))
	defer ts.Close()

	wb, err := NewEVSEWifiFromConfig(map[string]any{
		"uri": ts.URL,
	})
	if err != nil {
		t.Error(err)
	}

	if _, ok := api.Cap[api.ChargerEx](wb); !ok {
		t.Error("missing api.ChargerEx")
	}
}
