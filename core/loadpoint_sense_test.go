package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
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

func newSenseLoadpoint(charger api.Charger) (*Loadpoint, chan *Loadpoint, chan util.Param) {
	lpChan := make(chan *Loadpoint, 1)  // mirrors production cap(1)
	uiChan := make(chan util.Param, 16) // buffered so publish doesn't block

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		charger:     charger,
		chargeMeter: &Null{}, // silence nil panics, returns zero power
		lpChan:      lpChan,
		uiChan:      uiChan,
	}

	return lp, lpChan, uiChan
}

// TestSenseTriggersOnStatusChange verifies that a sensed charger status which
// differs from the authoritative status requests an immediate control pass and
// publishes the derived connection state.
func TestSenseTriggersOnStatusChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp, lpChan, uiChan := newSenseLoadpoint(charger)
	lp.status = api.StatusA // authoritative: disconnected

	charger.EXPECT().Status().Return(api.StatusB, nil) // sensed: connected, not charging

	lp.Sense()

	select {
	case got := <-lpChan:
		if got != lp {
			t.Fatalf("unexpected loadpoint on update channel")
		}
	default:
		t.Fatal("expected update request on status change")
	}

	published := drainParams(uiChan)
	if v, ok := published[keys.Connected]; !ok || v != true {
		t.Errorf("Connected: got %v (present=%v), want true", v, ok)
	}
	if v, ok := published[keys.Charging]; !ok || v != false {
		t.Errorf("Charging: got %v (present=%v), want false", v, ok)
	}
}

// TestSenseNoTriggerWithoutStatusChange verifies that an unchanged status does
// not request a redundant control pass.
func TestSenseNoTriggerWithoutStatusChange(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp, lpChan, _ := newSenseLoadpoint(charger)
	lp.status = api.StatusC // authoritative: charging

	charger.EXPECT().Status().Return(api.StatusC, nil) // sensed: unchanged

	lp.Sense()

	select {
	case <-lpChan:
		t.Fatal("unexpected update request without status change")
	default:
	}
}

// TestSenseRetriggersUntilControlCatchesUp verifies the self-healing behavior:
// while the authoritative status lags, every sense tick re-requests an update,
// covering a dropped cap(1) trigger.
func TestSenseRetriggersUntilControlCatchesUp(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp, lpChan, _ := newSenseLoadpoint(charger)
	lp.status = api.StatusA

	charger.EXPECT().Status().Return(api.StatusB, nil).Times(2)

	// first sense triggers and fills the cap(1) channel
	lp.Sense()
	select {
	case <-lpChan:
	default:
		t.Fatal("expected first update request")
	}

	// authoritative status still lags -> second sense triggers again
	lp.Sense()
	select {
	case <-lpChan:
	default:
		t.Fatal("expected re-trigger while status still differs")
	}
}
