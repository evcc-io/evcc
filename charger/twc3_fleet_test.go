package charger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// newTestFleet wires a twc3Fleet to a stub Fleet command server that always reports
// success, and returns a pointer to the recorded enable_schedule values sent. Note:
// enable_schedule == true means the schedule is active (charging blocked), false
// means the schedule is off (charging allowed) - the inverse of "enable charging".
func newTestFleet(t *testing.T, lockMaybeActive bool) (*twc3Fleet, *[]bool) {
	t.Helper()

	var (
		mu   sync.Mutex
		sent []bool
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cmd wcScheduleCmd
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			t.Errorf("decode request: %v", err)
		}
		mu.Lock()
		sent = append(sent, cmd.CommandProperties.Message.Wc.ConfigureChargeScheduleRequest.Config.EnableSchedule)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"response":{"ConfigureChargeScheduleResponse":{"error":1}}}`))
	}))
	t.Cleanup(srv.Close)

	f := &twc3Fleet{
		client:          request.NewHelper(util.NewLogger("test")),
		commandURL:      srv.URL,
		din:             "test-din",
		lockMaybeActive: lockMaybeActive,
	}
	return f, &sent
}

func TestTwc3FleetSetCharging(t *testing.T) {
	// variant B: setCharging always re-asserts the schedule, even on repeated enable,
	// so evcc's corrective sync calls actually reach the hardware.
	t.Run("always re-asserts", func(t *testing.T) {
		f, sent := newTestFleet(t, false)
		if err := f.setCharging(true); err != nil {
			t.Fatal(err)
		}
		if err := f.setCharging(true); err != nil {
			t.Fatal(err)
		}
		// two enable calls -> two Fleet requests, both "schedule off" (charging allowed)
		if want := []bool{false, false}; !slices.Equal(*sent, want) {
			t.Errorf("enable_schedule sent = %v, want %v", *sent, want)
		}
	})

	t.Run("disable then enable tracks lock state", func(t *testing.T) {
		f, sent := newTestFleet(t, false)

		if err := f.setCharging(false); err != nil {
			t.Fatal(err)
		}
		if !f.lockMaybeActive {
			t.Error("lockMaybeActive should be true after disable")
		}

		if err := f.setCharging(true); err != nil {
			t.Fatal(err)
		}
		if f.lockMaybeActive {
			t.Error("lockMaybeActive should be false after enable")
		}

		// disable -> schedule active (true); enable -> schedule off (false)
		if want := []bool{true, false}; !slices.Equal(*sent, want) {
			t.Errorf("enable_schedule sent = %v, want %v", *sent, want)
		}
	})
}

func TestTwc3FleetClearLock(t *testing.T) {
	t.Run("clears once when locked", func(t *testing.T) {
		f, sent := newTestFleet(t, true)

		if err := f.clearLock(); err != nil {
			t.Fatal(err)
		}
		if f.lockMaybeActive {
			t.Error("lockMaybeActive should be false after clearLock")
		}

		// second call is a no-op (no redundant Fleet request)
		if err := f.clearLock(); err != nil {
			t.Fatal(err)
		}
		if want := []bool{false}; !slices.Equal(*sent, want) {
			t.Errorf("enable_schedule sent = %v, want %v", *sent, want)
		}
	})

	t.Run("no-op when not locked", func(t *testing.T) {
		f, sent := newTestFleet(t, false)

		if err := f.clearLock(); err != nil {
			t.Fatal(err)
		}
		if len(*sent) != 0 {
			t.Errorf("expected no Fleet request, got %v", *sent)
		}
	})
}
