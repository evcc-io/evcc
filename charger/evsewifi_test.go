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

	wb, err := NewEVSEWifiFromConfig(map[string]interface{}{
		"uri": ts.URL,
		"meter": map[string]interface{}{
			"power":    true,
			"energy":   true,
			"currents": true,
		},
	})

	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing api.MeterEnergy")
	}

	if _, ok := wb.(api.PhaseCurrents); !ok {
		t.Error("missing api.PhaseCurrents")
	}

	if _, ok := wb.(api.ChargeTimer); !ok {
		t.Error("missing ChargeTimer api")
	}
}

func TestEvseWifiEx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{"list":[{"actualCurrentMA":600, "alwaysActive":true}]}`)
	}))
	defer ts.Close()

	wb, err := NewEVSEWifiFromConfig(map[string]interface{}{
		"uri": ts.URL,
	})

	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.ChargerEx); !ok {
		t.Error("missing api.ChargerEx")
	}
}
