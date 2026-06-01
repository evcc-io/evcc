package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/messenger"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// drainParams collects all currently buffered published params keyed by name.
func drainParams(ch chan util.Param) map[string]any {
	m := make(map[string]any)
	for {
		select {
		case p := <-ch:
			m[p.Key] = p.Val
		default:
			return m
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

type intervalStub struct{ d time.Duration }

func (s intervalStub) GetInterval() time.Duration { return s.d }

// chargerWithInterval combines a mock charger with an IntervalGetter hint.
func chargerWithInterval(ctrl *gomock.Controller, d time.Duration) api.Charger {
	return &struct {
		*api.MockCharger
		intervalStub
	}{api.NewMockCharger(ctrl), intervalStub{d}}
}

// TestSenseCadence verifies the per-charger sense interval: a charger hint is
// honoured (when >= 1s), otherwise the derived base is used.
func TestSenseCadence(t *testing.T) {
	ctrl := gomock.NewController(t)
	base := 3 * time.Second

	tc := []struct {
		name    string
		charger api.Charger
		want    time.Duration
	}{
		{"no hint -> base", api.NewMockCharger(ctrl), base},
		{"slower hint -> hint", chargerWithInterval(ctrl, 30*time.Second), 30 * time.Second},
		{"sub-second hint ignored -> base", chargerWithInterval(ctrl, 500*time.Millisecond), base},
	}

	for _, c := range tc {
		lp := &Loadpoint{charger: c.charger}
		if got := lp.senseCadence(base); got != c.want {
			t.Errorf("%s: senseCadence = %v, want %v", c.name, got, c.want)
		}
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

// TestNotifyTriggersSense verifies that a push-capable charger's Notify requests
// an immediate sense, and that it never blocks when the channel is full.
func TestNotifyTriggersSense(t *testing.T) {
	senseChan := make(chan struct{}, 1)
	lp := &Loadpoint{log: util.NewLogger("foo"), senseChan: senseChan}

	lp.Notify()

	select {
	case <-senseChan:
	default:
		t.Fatal("expected an immediate sense request")
	}

	// must not block when nobody is draining the channel
	lp.senseChan = make(chan struct{}) // unbuffered, no reader
	done := make(chan struct{})
	go func() { lp.Notify(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Notify blocked")
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

// TestObservePublishesConnectionState verifies observe publishes the snappy-UI
// connection state (the reason the sense loop exists).
func TestObservePublishesConnectionState(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	uiChan := make(chan util.Param, 32)
	lp := newObserveLoadpoint(charger)
	lp.uiChan = uiChan // bidirectional handle so the test can drain it
	lp.status = api.StatusA

	charger.EXPECT().Status().Return(api.StatusC, nil) // connected and charging

	lp.observe()

	published := drainParams(uiChan)
	if v, ok := published[keys.Connected]; !ok || v != true {
		t.Errorf("Connected: got %v (present=%v), want true", v, ok)
	}
	if v, ok := published[keys.Charging]; !ok || v != true {
		t.Errorf("Charging: got %v (present=%v), want true", v, ok)
	}
}

// TestObserveSetsObserved verifies observe marks the loadpoint observed so
// control may actuate (and that a fresh loadpoint is not yet observed).
func TestObserveSetsObserved(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := newObserveLoadpoint(charger)
	lp.status = api.StatusA

	if lp.observed {
		t.Fatal("a fresh loadpoint must not be observed yet")
	}

	charger.EXPECT().Status().Return(api.StatusB, nil)
	lp.observe()

	if !lp.observed {
		t.Fatal("observe must mark the loadpoint observed")
	}
}

// TestObserveLatchesWelcomeCharge verifies a faster sense loop does not clobber
// a latched welcome charge before control consumes it.
func TestObserveLatchesWelcomeCharge(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := newObserveLoadpoint(charger)
	lp.status = api.StatusC
	lp.welcomeCharge = true // latched on an earlier connect tick

	// no status change -> updateChargerStatus reports welcomeCharge=false
	charger.EXPECT().Status().Return(api.StatusC, nil)
	lp.observe()

	if !lp.welcomeCharge {
		t.Fatal("observe clobbered the latched welcome charge before control consumed it")
	}
}
