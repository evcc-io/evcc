package goe

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

type handler struct {
	uri      string
	response string
}

func (h *handler) expect(uri string) {
	h.uri = uri
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	fmt.Println(path)

	if path == h.uri {
		if h.response != "" {
			fmt.Fprint(w, h.response)
		} else {
			fmt.Fprint(w, "{}")
		}
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "expected %s", h.uri)
	}
}

func TestLocalV1(t *testing.T) {
	h := &handler{}
	srv := httptest.NewTestServer(t, h)
	srv.Start()

	// h.expect("/api/status?filter=alw")
	local := NewLocal(util.NewLogger("foo"), srv.URL, 0)

	h.expect("/status")
	if _, err := local.Status(); err != nil {
		t.Error(err)
	}

	h.expect("/mqtt?payload=foo=bar")
	if err := local.Update("foo=bar"); err != nil {
		t.Error(err)
	}
}

func TestLocalV2(t *testing.T) {
	h := &handler{}
	srv := httptest.NewTestServer(t, h)
	srv.Start()

	h.expect("/api/status?filter=alw")
	local := NewLocal(util.NewLogger("foo"), srv.URL, 0)

	h.expect("/api/status?filter=alw,car,err,eto,nrg,wh,trx,cards")
	if _, err := local.Status(); err != nil {
		t.Error(err)
	}

	h.expect("/api/set?foo=bar")
	h.response = `{"foo":true}`
	if err := local.Update("foo=bar"); err != nil {
		t.Error(err)
	}
}

func TestLocalV2UpdateResponse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payload  string
		response string
		err      string
	}{
		{"confirmed", "frc=1", `{"frc":true}`, ""},
		{"rejected", "frc=1", `{"frc":"write rejected"}`, "set frc: write rejected"},
		{"false", "frc=1", `{"frc":false}`, "set frc: false"},
		{"missing", "frc=1", `{}`, "set frc: missing confirmation"},
		{"null response", "frc=1", `null`, "set frc: missing confirmation"},
		{"null value", "frc=1", `{"frc":null}`, "set frc: <nil>"},
		{"wrong key", "frc=1", `{"amp":true}`, "set frc: missing confirmation"},
		{"string true", "frc=1", `{"frc":"true"}`, "set frc: true"},
		{"numeric true", "frc=1", `{"frc":1}`, "set frc: 1"},
		{"current", "amp=6", `{"amp":true}`, ""},
		{"phases", "psm=1", `{"psm":true}`, ""},
		{"multiple", "frc=1&amp=6", `{"frc":true,"amp":true}`, ""},
		{"partial", "frc=1&amp=6", `{"frc":true}`, "set amp: missing confirmation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/status":
					fmt.Fprint(w, `{"alw":false}`)
				case "/api/set":
					if r.URL.RawQuery != tc.payload {
						t.Errorf("unexpected update: %s", r.URL.RawQuery)
					}
					fmt.Fprint(w, tc.response)
				default:
					http.NotFound(w, r)
				}
			}))
			srv.Start()

			local := NewLocal(util.NewLogger("foo"), srv.URL, 0)
			require.True(t, local.IsV2())
			err := local.Update(tc.payload)
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.err)
			}
		})
	}
}
