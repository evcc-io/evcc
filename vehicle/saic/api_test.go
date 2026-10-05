package saic

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/saic/requests"
)

// TestTakeReturnsValueOnce verifies a stored value is returned exactly once,
// and every other state yields ErrMustRetry or starts a fresh request.
func TestTakeReturnsValueOnce(t *testing.T) {
	v := &API{log: util.NewLogger("saic")}

	// idle: no pending value and no poll -> caller must start a new query
	if _, _, query := v.take(); !query {
		t.Fatal("idle: expected query=true")
	}

	// background poll running -> ErrMustRetry, no new query, no value
	v.mu.Lock()
	v.state = stateRunning
	v.mu.Unlock()
	if _, err, query := v.take(); query || !errors.Is(err, api.ErrMustRetry) {
		t.Fatalf("running: got err=%v query=%v, want ErrMustRetry", err, query)
	}

	// a background value arrives
	var cs requests.ChargeStatus
	cs.RvsChargeStatus.Mileage = 4242
	v.store(cs)

	// returned exactly once with nil error
	res, err, query := v.take()
	if query || err != nil || res.RvsChargeStatus.Mileage != 4242 {
		t.Fatalf("first take: got mileage=%d err=%v query=%v, want value once", res.RvsChargeStatus.Mileage, err, query)
	}

	// consumed: the next call must start a fresh request again
	if _, _, query := v.take(); !query {
		t.Fatal("after consume: expected query=true")
	}
}
