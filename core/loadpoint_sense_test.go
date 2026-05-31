package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/messenger"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// TestSenseInterval verifies the sense cadence derived from the control interval.
func TestSenseInterval(t *testing.T) {
	tc := []struct {
		in, want time.Duration
	}{
		{30 * time.Second, 3 * time.Second},
		{10 * time.Second, 1 * time.Second},
		{5 * time.Second, 1 * time.Second}, // clamped to 1s minimum
		{0, 1 * time.Second},
	}

	for _, c := range tc {
		if got := senseInterval(c.in); got != c.want {
			t.Errorf("senseInterval(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func newObserveLoadpoint(charger api.Charger) *Loadpoint {
	return &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clock.NewMock(),
		bus:         evbus.New(),
		charger:     charger,
		chargeMeter: &Null{}, // silence nil panics, returns zero power
		uiChan:      make(chan util.Param, 32),
		lpChan:      make(chan *Loadpoint, 1),
		pushChan:    make(chan messenger.Event, 8),
		minCurrent:  minA,
		maxCurrent:  maxA,
	}
}

// TestObserveTriggersOnStatusChange verifies that observe authoritatively updates
// the status and requests a prompt control pass when it changed.
func TestObserveTriggersOnStatusChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := newObserveLoadpoint(charger)
	lp.status = api.StatusA

	charger.EXPECT().Status().Return(api.StatusB, nil)

	lp.observe()

	if lp.GetStatus() != api.StatusB {
		t.Fatalf("status not updated: %v", lp.GetStatus())
	}
	if !lp.pendingControl.Load() {
		t.Fatal("status change must request a control pass")
	}
}

// TestObserveNoTriggerWithoutStatusChange verifies that an unchanged status does
// not request a redundant control pass.
func TestObserveNoTriggerWithoutStatusChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := newObserveLoadpoint(charger)
	lp.status = api.StatusC // charging

	charger.EXPECT().Status().Return(api.StatusC, nil) // unchanged

	lp.observe()

	if lp.pendingControl.Load() {
		t.Fatal("unchanged status must not request a control pass")
	}
}
