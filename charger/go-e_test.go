package charger

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/sponsor"
)

type handler struct {
	uri string
}

func (h *handler) expect(uri string) {
	h.uri = uri
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}

	if path == h.uri {
		fmt.Fprint(w, "{}")
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "expected %s", h.uri)
	}
}

func TestGoEV1(t *testing.T) {
	srv := httptest.NewServer(new(handler))

	sponsor.Subject = "foo"

	wb, err := newGoEFromConfig(false, map[string]any{"uri": srv.URL})
	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.Meter); !ok {
		t.Error("missing Meter api")
	}

	if _, ok := wb.(api.PhaseCurrents); !ok {
		t.Error("missing PhaseCurrents api")
	}

	if _, ok := wb.(api.Identifier); !ok {
		t.Error("missing Identifier api")
	}
}

func TestGoEV2(t *testing.T) {
	srv := httptest.NewServer(new(handler))
	srv.Config.Handler.(*handler).expect("/api/status?filter=alw")

	sponsor.Subject = "foo"

	wb, err := newGoEFromConfig(false, map[string]any{"uri": srv.URL})
	if err != nil {
		t.Error(err)
	}

	if _, ok := wb.(api.Meter); !ok {
		t.Error("missing Meter api")
	}

	if _, ok := wb.(api.PhaseCurrents); !ok {
		t.Error("missing PhaseCurrents api")
	}

	if _, ok := wb.(api.Identifier); !ok {
		t.Error("missing Identifier api")
	}

	if _, ok := wb.(api.MeterEnergy); !ok {
		t.Error("missing MeterEnergy api")
	}

	if _, ok := wb.(api.PhaseSwitcher); !ok {
		t.Error("missing PhaseSwitcher api")
	}
}
