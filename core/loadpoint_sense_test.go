package core

import (
	"sync"
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

// measuringMeter is a minimal, lock-free charge meter for concurrency tests.
type measuringMeter struct{}

func (measuringMeter) CurrentPower() (float64, error)               { return 3700, nil }
func (measuringMeter) Currents() (float64, float64, float64, error) { return 16, 16, 16, nil }

// TestSenseControlConcurrency exercises the concurrent access introduced by the
// sense/control split: the sense loop writes chargePower/chargeCurrents (under
// lp.Lock) while the control loop and API read them via the locked getters. It
// is meant to be run under -race to verify the locking discipline.
func TestSenseControlConcurrency(t *testing.T) {
	lp := newObserveLoadpoint(nil)
	lp.chargeMeter = measuringMeter{}
	lp.uiChan = nil // suppress publish so the tight loop does not block on the channel
	lp.enabled = true
	lp.offeredCurrent = 16

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// sense loop: writes charge power and currents
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				lp.UpdateChargePowerAndCurrents()
			}
		}
	}()

	// control loop / API: reads the sensed values via the locked getters
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = lp.GetChargePower()
					_ = lp.GetChargeCurrents()
					_ = lp.GetInflightPower()
				}
			}
		}()
	}

	time.Sleep(30 * time.Millisecond)
	close(stop)
	wg.Wait()
}

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

	lp.Observe()

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

	lp.Observe()

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

	lp.Observe()

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
	lp.Observe()

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
	lp.Observe()

	if !lp.welcomeCharge {
		t.Fatal("observe clobbered the latched welcome charge before control consumed it")
	}
}
