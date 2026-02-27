package toyota

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
)

func newTestProvider(chargingStatus string, refreshFn func() error) *Provider {
	log := util.NewLogger("test")

	p := &Provider{
		log:     log,
		refresh: refreshFn,
	}

	p.status = func() (Status, error) {
		res := Status{}
		res.Payload.ChargingStatus = chargingStatus
		res.Payload.BatteryLevel = 50
		p.triggerRefreshIfCharging(res)
		return res, nil
	}

	return p
}

func TestRefreshWhileCharging(t *testing.T) {
	var called int
	p := newTestProvider("charging", func() error {
		called++
		return nil
	})

	if _, err := p.Soc(); err != nil {
		t.Fatal(err)
	}

	if called != 1 {
		t.Fatalf("expected 1 refresh call, got %d", called)
	}
}

func TestNoRefreshWhenNotCharging(t *testing.T) {
	var called int

	for _, status := range []string{"idle", "complete", ""} {
		called = 0
		p := newTestProvider(status, func() error {
			called++
			return nil
		})

		if _, err := p.Soc(); err != nil {
			t.Fatal(err)
		}

		if called != 0 {
			t.Fatalf("expected no refresh for status %q, got %d calls", status, called)
		}
	}
}

func TestRefreshRateLimited(t *testing.T) {
	var called int
	p := newTestProvider("charging", func() error {
		called++
		return nil
	})

	if _, err := p.Soc(); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("expected 1 refresh call, got %d", called)
	}

	if _, err := p.Soc(); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("expected still 1 refresh call (rate limited), got %d", called)
	}

	p.lastRefresh = time.Now().Add(-16 * time.Minute)

	if _, err := p.Soc(); err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected 2 refresh calls after cooldown, got %d", called)
	}
}

func TestWakeUp(t *testing.T) {
	var called int
	p := newTestProvider("idle", func() error {
		called++
		return nil
	})

	if err := p.WakeUp(); err != nil {
		t.Fatal(err)
	}

	if called != 1 {
		t.Fatalf("expected 1 refresh call from WakeUp, got %d", called)
	}
}
