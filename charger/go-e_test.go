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
	uri  string
	body string
}

func (h *handler) expect(uri string) {
	h.uri = uri
	h.body = "{}"
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}

	if path == h.uri {
		fmt.Fprint(w, h.body)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "expected %s", h.uri)
	}
}

func TestGoEV1(t *testing.T) {
	srv := httptest.NewTestServer(t, new(handler))
	srv.Start()

	sponsor.Subject = "foo"

	wb, err := newGoEFromConfig(false, map[string]any{"uri": srv.URL})
	if err != nil {
		t.Error(err)
	}

	if _, ok := api.Cap[api.Meter](wb); !ok {
		t.Error("missing Meter api")
	}

	if _, ok := api.Cap[api.PhaseCurrents](wb); !ok {
		t.Error("missing PhaseCurrents api")
	}

	if _, ok := api.Cap[api.Identifier](wb); !ok {
		t.Error("missing Identifier api")
	}

	if _, ok := api.Cap[api.StatusReasoner](wb); ok {
		t.Error("unexpected StatusReasoner api")
	}
}

func TestGoEV2(t *testing.T) {
	h := new(handler)
	srv := httptest.NewTestServer(t, h)
	srv.Start()
	h.expect("/api/status?filter=alw")

	sponsor.Subject = "foo"

	wb, err := newGoEFromConfig(false, map[string]any{"uri": srv.URL, "cache": "0s"})
	if err != nil {
		t.Error(err)
	}

	if _, ok := api.Cap[api.Meter](wb); !ok {
		t.Error("missing Meter api")
	}

	if _, ok := api.Cap[api.PhaseCurrents](wb); !ok {
		t.Error("missing PhaseCurrents api")
	}

	if _, ok := api.Cap[api.Identifier](wb); !ok {
		t.Error("missing Identifier api")
	}

	if _, ok := api.Cap[api.MeterEnergy](wb); !ok {
		t.Error("missing MeterEnergy api")
	}

	if _, ok := api.Cap[api.PhaseSwitcher](wb); !ok {
		t.Error("missing PhaseSwitcher api")
	}

	sr, ok := api.Cap[api.StatusReasoner](wb)
	if !ok {
		t.Fatal("missing StatusReasoner api")
	}

	h.expect("/api/status?filter=alw,car,eto,nrg,wh,trx,cards,modelStatus")

	for _, tc := range []struct {
		body   string
		reason api.Reason
	}{
		{`{"modelStatus":2}`, api.ReasonWaitingForAuthorization},
		{`{"modelStatus":3}`, api.ReasonUnknown},
		{`{}`, api.ReasonUnknown},
	} {
		h.body = tc.body

		reason, err := sr.StatusReason()
		if err != nil {
			t.Fatal(err)
		}

		if reason != tc.reason {
			t.Errorf("%s: expected %v, got %v", tc.body, tc.reason, reason)
		}
	}
}
