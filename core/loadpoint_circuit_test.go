package core

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/util"
	"go.uber.org/mock/gomock"
)

// TestSetLimitNoSelfCapFromOwnReserve verifies a loadpoint is not capped down by
// its own in-flight reserve: while it ramps toward a higher setpoint, the
// circuit holds the reserve (intended - measured), but re-applying the same
// setpoint must not treat that reserve as foreign headroom and cap back to the
// metered draw (which oscillates on a circuit sized for a single charger).
func TestSetLimitNoSelfCapFromOwnReserve(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	circ, err := circuit.New(util.NewLogger("circ"), "test", maxA, 0, nil, 0) // maxCurrent 16A, no meter
	if err != nil {
		t.Fatal(err)
	}

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clk,
		bus:         evbus.New(),
		charger:     charger,
		chargeMeter: &Null{},
		wakeUpTimer: NewTimer(),
		circuit:     circ,
		minCurrent:  minA,
		maxCurrent:  maxA,
		phases:      1,
		interval:    time.Minute,
	}
	lp.status = api.StatusC
	lp.enabled = true
	lp.offeredCurrent = maxA                        // already actuated to 16A
	lp.chargeCurrents = []float64{minA, minA, minA} // car still ramping (6A)
	lp.chargePower = currentToPower(minA, 1)
	lp.actuatedAt = clk.Now() // reserve active: intended 16A - measured 6A = 10A

	// the circuit sees metered 6A + this loadpoint's own 10A reserve = 16A
	if err := circ.Update([]api.CircuitLoad{lp}); err != nil {
		t.Fatal(err)
	}

	// re-applying 16A must be a no-op (no charger write): the reserve is this
	// loadpoint's own contribution, not foreign headroom. A capped-down value
	// would call MaxCurrent(6) -> an unexpected mock call fails the test.
	if err := lp.setLimit(maxA); err != nil {
		t.Fatalf("setLimit: %v", err)
	}

	if lp.offeredCurrent != maxA {
		t.Fatalf("loadpoint capped itself via its own reserve: offeredCurrent=%v, want %v", lp.offeredCurrent, maxA)
	}
}
