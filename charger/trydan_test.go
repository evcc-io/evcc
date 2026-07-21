package charger

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

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

func TestTrydanAutoUnlock(t *testing.T) {
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

func TestTrydanAutoUnlockDisabled(t *testing.T) {
	sponsor.Subject = "foo"

	srv, ts := newTrydanTestServer(1, 1) // locked and paused
	defer srv.Close()

	wb, err := NewTrydan(srv.URL, 0, false) // autoUnlock off
	if err != nil {
		t.Fatal(err)
	}

	if err := wb.Enable(true); err != nil {
		t.Fatal(err)
	}
	if ts.wrote("Locked=0") {
		t.Error("Enable(true) must not touch Locked unless autoUnlock is enabled")
	}
}

func TestTrydanAutoUnlockNotNeeded(t *testing.T) {
	sponsor.Subject = "foo"

	srv, ts := newTrydanTestServer(0, 1) // already unlocked, paused
	defer srv.Close()

	wb, err := NewTrydan(srv.URL, 0, true)
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
