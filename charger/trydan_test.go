package charger

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util/sponsor"
)

func TestTrydanPhases(t *testing.T) {
	chargeMode := trydanChargeModeMono

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/RealTimeData":
			fmt.Fprintf(w, `{"ChargeMode":%d}`, chargeMode)
		case "/write/ChargeMode=0":
			chargeMode = trydanChargeModeMono
			fmt.Fprint(w, "OK")
		case "/write/ChargeMode=1":
			chargeMode = trydanChargeModeThree
			fmt.Fprint(w, "OK")
		default:
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "unexpected request: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	sponsor.Subject = "foo"

	wb, err := NewTrydan(srv.URL, 0)
	if err != nil {
		t.Fatal(err)
	}

	c := wb.(*Trydan)

	if phases, err := c.GetPhases(); err != nil || phases != 1 {
		t.Errorf("GetPhases() = %d, %v; want 1, nil", phases, err)
	}

	if err := c.Phases1p3p(3); err != nil {
		t.Fatal(err)
	}
	c.statusG.Reset()

	if phases, err := c.GetPhases(); err != nil || phases != 3 {
		t.Errorf("GetPhases() after switch to 3p = %d, %v; want 3, nil", phases, err)
	}

	if err := c.Phases1p3p(1); err != nil {
		t.Fatal(err)
	}
	c.statusG.Reset()

	if phases, err := c.GetPhases(); err != nil || phases != 1 {
		t.Errorf("GetPhases() after switch to 1p = %d, %v; want 1, nil", phases, err)
	}
}

func TestTrydanPhasesMixed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/RealTimeData" {
			fmt.Fprintf(w, `{"ChargeMode":%d}`, trydanChargeModeMixed)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	sponsor.Subject = "foo"

	wb, err := NewTrydan(srv.URL, 0)
	if err != nil {
		t.Fatal(err)
	}

	c := wb.(*Trydan)

	if phases, err := c.GetPhases(); err != nil || phases != 0 {
		t.Errorf("GetPhases() in mixed mode = %d, %v; want 0, nil", phases, err)
	}
}
