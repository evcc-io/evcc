package charger

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/sponsor"
)

// trydanTestServer serves a minimal, stateful mock of the Trydan HTTP API,
// tracking every /write/ call so tests can assert on the exact sequence.
type trydanTestServer struct {
	mu     sync.Mutex
	locked int
	paused int
	calls  []string
}

func newTrydanTestServer(locked, paused int) (*httptest.Server, *trydanTestServer) {
	ts := &trydanTestServer{locked: locked, paused: paused}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.mu.Lock()
		defer ts.mu.Unlock()

		switch {
		case r.URL.Path == "/RealTimeData":
			fmt.Fprintf(w, `{"ChargeState":1,"ChargePower":0,"Dynamic":0,"Locked":%d,"Paused":%d}`, ts.locked, ts.paused)
		case strings.HasPrefix(r.URL.Path, "/write/"):
			kv := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/write/"), "=", 2)
			ts.calls = append(ts.calls, kv[0]+"="+kv[1])
			val, _ := strconv.Atoi(kv[1])
			switch kv[0] {
			case "Locked":
				ts.locked = val
			case "Paused":
				ts.paused = val
			}
			fmt.Fprint(w, "OK")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return srv, ts
}

func (ts *trydanTestServer) wrote(call string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for _, c := range ts.calls {
		if c == call {
			return true
		}
	}
	return false
}

// By default (autoUnlock false) evcc manages Locked itself: save/restore around a session.
func TestTrydanManagesLockByDefault(t *testing.T) {
	sponsor.Subject = "foo"

	srv, ts := newTrydanTestServer(1, 1) // locked and paused
	defer srv.Close()

	wb, err := NewTrydan(srv.URL, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := wb.Enable(true); err != nil {
		t.Fatal(err)
	}
	if !ts.wrote("Locked=0") {
		t.Error("expected Enable(true) to unlock a locked charger")
	}
	if !ts.wrote("Paused=0") {
		t.Error("expected Enable(true) to unpause")
	}

	if err := wb.Enable(false); err != nil {
		t.Fatal(err)
	}
	if !ts.wrote("Locked=1") {
		t.Error("expected Enable(false) to restore the lock it removed")
	}
}

// autoUnlock signals the charger already has its own auto-unlock mechanism, so evcc
// must leave Locked alone entirely and only manage Paused.
func TestTrydanAutoUnlockLeavesLockAlone(t *testing.T) {
	sponsor.Subject = "foo"

	srv, ts := newTrydanTestServer(1, 1) // locked and paused
	defer srv.Close()

	wb, err := NewTrydan(srv.URL, 0, true)
	if err != nil {
		t.Fatal(err)
	}

	if err := wb.Enable(true); err != nil {
		t.Fatal(err)
	}
	if ts.wrote("Locked=0") {
		t.Error("Enable(true) must not touch Locked when autoUnlock is set")
	}
}

func TestTrydanManagedLockNotNeeded(t *testing.T) {
	sponsor.Subject = "foo"

	srv, ts := newTrydanTestServer(0, 1) // already unlocked, paused
	defer srv.Close()

	wb, err := NewTrydan(srv.URL, 0, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := wb.Enable(true); err != nil {
		t.Fatal(err)
	}
	if ts.wrote("Locked=0") {
		t.Error("Enable(true) must not write Locked when it wasn't locked to begin with")
	}

	if err := wb.Enable(false); err != nil {
		t.Fatal(err)
	}
	if ts.wrote("Locked=1") {
		t.Error("Enable(false) must not lock a charger evcc never unlocked")
	}
}

func trydanTestServerWithBody(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
}

// ChargeState maps directly to api.ChargeStatus, except firmware 2.5.0 keeps it at
// "charging" even after Paused=1, so that specific combination must fall back to StatusB.
func TestTrydanStatus(t *testing.T) {
	sponsor.Subject = "foo"

	tests := []struct {
		name    string
		json    string
		want    api.ChargeStatus
		wantErr bool
	}{
		{"not connected", `{"ChargeState":0,"Paused":0}`, api.StatusA, false},
		{"connected, not charging", `{"ChargeState":1,"Paused":0}`, api.StatusB, false},
		{"charging", `{"ChargeState":2,"Paused":0}`, api.StatusC, false},
		// firmware 2.5.0 keeps ChargeState at "charging" even after Paused=1
		{"paused mid-session", `{"ChargeState":2,"Paused":1}`, api.StatusB, false},
		{"unknown state", `{"ChargeState":9,"Paused":0}`, api.StatusNone, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := trydanTestServerWithBody(tc.json)
			defer srv.Close()

			wb, err := NewTrydan(srv.URL, 0, false)
			if err != nil {
				t.Fatal(err)
			}

			status, err := wb.Status()
			if tc.wantErr != (err != nil) {
				t.Fatalf("got err %v, wantErr %v", err, tc.wantErr)
			}
			if status != tc.want {
				t.Errorf("got status %v, want %v", status, tc.want)
			}
		})
	}
}

// Currents()/Voltages() must trust a zero reading while idle, but treat it as
// unavailable (older firmware without these fields) whenever real power is flowing,
// since ChargePower>0 with all phases at zero is otherwise physically impossible.
func TestTrydanPhaseMeasurementsUnavailable(t *testing.T) {
	sponsor.Subject = "foo"

	tests := []struct {
		name            string
		json            string
		wantUnavailable bool
		wantCurrentL1   float64
		wantVoltageL1   float64
	}{
		{
			name: "idle, zero readings are trusted",
			json: `{"ChargeState":1,"ChargePower":0,
				"IntensityMeasure_L1":0,"IntensityMeasure_L2":0,"IntensityMeasure_L3":0,
				"VoltageMeasure_L1":0,"VoltageMeasure_L2":0,"VoltageMeasure_L3":0}`,
		},
		{
			name: "charging but all phases read zero - unsupported firmware",
			json: `{"ChargeState":2,"ChargePower":4600,
				"IntensityMeasure_L1":0,"IntensityMeasure_L2":0,"IntensityMeasure_L3":0,
				"VoltageMeasure_L1":0,"VoltageMeasure_L2":0,"VoltageMeasure_L3":0}`,
			wantUnavailable: true,
		},
		{
			name: "charging with real per-phase readings",
			json: `{"ChargeState":2,"ChargePower":4600,
				"IntensityMeasure_L1":20.5,"IntensityMeasure_L2":0,"IntensityMeasure_L3":0,
				"VoltageMeasure_L1":228.1,"VoltageMeasure_L2":0,"VoltageMeasure_L3":0}`,
			wantCurrentL1: 20.5,
			wantVoltageL1: 228.1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := trydanTestServerWithBody(tc.json)
			defer srv.Close()

			wb, err := NewTrydan(srv.URL, 0, false)
			if err != nil {
				t.Fatal(err)
			}

			pc, ok := api.Cap[api.PhaseCurrents](wb)
			if !ok {
				t.Fatal("missing api.PhaseCurrents")
			}
			pv, ok := api.Cap[api.PhaseVoltages](wb)
			if !ok {
				t.Fatal("missing api.PhaseVoltages")
			}

			i1, _, _, err := pc.Currents()
			switch {
			case tc.wantUnavailable && !errors.Is(err, api.ErrNotAvailable):
				t.Errorf("Currents: got err %v, want ErrNotAvailable", err)
			case !tc.wantUnavailable && err != nil:
				t.Fatal(err)
			case !tc.wantUnavailable && i1 != tc.wantCurrentL1:
				t.Errorf("Currents: got L1=%v, want %v", i1, tc.wantCurrentL1)
			}

			v1, _, _, err := pv.Voltages()
			switch {
			case tc.wantUnavailable && !errors.Is(err, api.ErrNotAvailable):
				t.Errorf("Voltages: got err %v, want ErrNotAvailable", err)
			case !tc.wantUnavailable && err != nil:
				t.Fatal(err)
			case !tc.wantUnavailable && v1 != tc.wantVoltageL1:
				t.Errorf("Voltages: got L1=%v, want %v", v1, tc.wantVoltageL1)
			}
		})
	}
}
