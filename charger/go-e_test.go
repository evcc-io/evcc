package charger

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/evcc-io/evcc/api"
	goe "github.com/evcc-io/evcc/charger/go-e"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/require"
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
}

func TestGoEV2(t *testing.T) {
	h := new(handler)
	srv := httptest.NewTestServer(t, h)
	srv.Start()
	h.expect("/api/status?filter=alw")

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

	if _, ok := api.Cap[api.MeterEnergy](wb); !ok {
		t.Error("missing MeterEnergy api")
	}

	if _, ok := api.Cap[api.PhaseSwitcher](wb); !ok {
		t.Error("missing PhaseSwitcher api")
	}
}

type goEAPI struct {
	response  goe.Response
	updates   []string
	updateErr error
}

func (c *goEAPI) IsV2() bool {
	return true
}

func (c *goEAPI) Status() (goe.Response, error) {
	return c.response, nil
}

func (c *goEAPI) Update(payload string) error {
	c.updates = append(c.updates, payload)
	return c.updateErr
}

func TestGoEFaultDisable(t *testing.T) {
	res := &goe.StatusResponse2{Car: 5, Err: 5}
	conn := &goEAPI{response: res}
	wb := &GoE{api: conn}

	_, err := wb.Status()
	require.EqualError(t, err, "charger error: 5")
	require.Equal(t, []string{"frc=1"}, conn.updates)

	_, err = wb.Status()
	require.EqualError(t, err, "charger error: 5")
	require.Equal(t, []string{"frc=1"}, conn.updates)

	res.Car = 1
	status, err := wb.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)

	res.Car = 5
	_, err = wb.Status()
	require.EqualError(t, err, "charger error: 5")
	require.Equal(t, []string{"frc=1", "frc=1"}, conn.updates)
}

func TestGoEFaultReset(t *testing.T) {
	for _, tc := range []struct {
		car     int
		status  api.ChargeStatus
		updates int
	}{
		{1, api.StatusA, 2},
		{2, api.StatusC, 2},
		{3, api.StatusB, 2},
		{4, api.StatusB, 2},
		{0, api.StatusNone, 1},
		{6, api.StatusNone, 1},
	} {
		t.Run(fmt.Sprint(tc.car), func(t *testing.T) {
			res := &goe.StatusResponse2{Car: 5, Err: 5}
			conn := &goEAPI{response: res}
			wb := &GoE{api: conn}

			_, err := wb.Status()
			require.EqualError(t, err, "charger error: 5")

			res.Car = tc.car
			res.Err = 0
			status, err := wb.Status()
			require.Equal(t, tc.status, status)
			if tc.status == api.StatusNone {
				require.EqualError(t, err, fmt.Sprintf("car unknown result: %d", tc.car))
			} else {
				require.NoError(t, err)
			}

			res.Car = 5
			res.Err = 5
			_, err = wb.Status()
			require.EqualError(t, err, "charger error: 5")
			require.Len(t, conn.updates, tc.updates)
		})
	}
}

func TestGoEFaultDisableRetry(t *testing.T) {
	res := &goe.StatusResponse2{Car: 5, Err: 5}
	conn := &goEAPI{
		response:  res,
		updateErr: errors.New("failed"),
	}
	wb := &GoE{api: conn}

	_, err := wb.Status()
	require.EqualError(t, err, "disable charger after fault: failed")

	conn.updateErr = nil
	_, err = wb.Status()
	require.EqualError(t, err, "charger error: 5")
	require.Equal(t, []string{"frc=1", "frc=1"}, conn.updates)
}

func TestGoEFaultDisableHTTP(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
		err      string
	}{
		{"rejected", `{"frc":"write rejected"}`, "set frc: write rejected"},
		{"missing", `{}`, "set frc: missing confirmation"},
		{"false", `{"frc":false}`, "set frc: false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var attempts atomic.Int32
			srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/status":
					fmt.Fprint(w, `{"car":5,"err":5,"alw":false}`)
				case "/api/set":
					if r.URL.RawQuery != "frc=1" {
						t.Errorf("unexpected update: %s", r.URL.RawQuery)
					}
					if attempts.Add(1) == 1 {
						fmt.Fprint(w, tc.response)
					} else {
						fmt.Fprint(w, `{"frc":true}`)
					}
				default:
					http.NotFound(w, r)
				}
			}))
			srv.Start()
			wb := &GoE{api: goe.NewLocal(util.NewLogger("go-e"), srv.URL, 0)}

			status, err := wb.Status()
			require.Equal(t, api.StatusNone, status)
			require.EqualError(t, err, "disable charger after fault: "+tc.err)
			require.EqualValues(t, 1, attempts.Load())

			for range 2 {
				status, err = wb.Status()
				require.Equal(t, api.StatusNone, status)
				require.EqualError(t, err, "charger error: 5")
				require.EqualValues(t, 2, attempts.Load())
			}
		})
	}
}
