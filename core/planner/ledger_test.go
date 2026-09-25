package planner

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// slot returns the plan slot at the given hour offset
func slot(plan api.Rates, start time.Time, hour int) api.Rate {
	return SlotAt(start.Add(time.Duration(hour)*time.Hour), plan)
}

// mockCircuit returns a circuit with the given limit and parent
func mockCircuit(ctrl *gomock.Controller, maxPower float64, parent api.Circuit) api.Circuit {
	c := api.NewMockCircuit(ctrl)
	c.EXPECT().GetMaxPower().Return(maxPower).AnyTimes()
	c.EXPECT().GetParent().Return(parent).AnyTimes()
	return c
}

// ledgerFixture is a shared circuit with hourly rates 20-60-10-80-40-90 and a target six hours out
type ledgerFixture struct {
	clock   *clock.Mock
	circuit api.Circuit
	tariff  api.Tariff
	target  time.Time
	ledger  *Ledger
}

func newLedgerFixture(t *testing.T, circuitPower float64) ledgerFixture {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	return ledgerFixture{
		clock:   clock,
		circuit: mockCircuit(ctrl, circuitPower, nil),
		tariff:  trf,
		target:  clock.Now().Add(6 * time.Hour),
		ledger:  NewLedger(),
	}
}

// planner returns a planner for a loadpoint on the shared circuit
func (f ledgerFixture) planner(id, priority int, maxPower float64) *Planner {
	return New(util.NewLogger("foo"), f.tariff, WithLedger(f.ledger, func() Owner {
		return Owner{Id: id, Priority: priority, Target: f.target, Circuit: f.circuit, MaxPower: maxPower, MinPower: 1380}
	}), func(p *Planner) { p.clock = f.clock })
}

func TestLedgerSharedCircuit(t *testing.T) {
	// 32 A single phase main circuit, 32 A charger with priority, 16 A charger without
	f := newLedgerFixture(t, 7360)
	fast := f.planner(0, 1, 7360)
	slow := f.planner(1, 0, 3680)
	now := f.clock.Now()

	// the slow charger plans first and takes the cheapest slots
	slowPlan := slow.Plan(2*time.Hour, 0, f.target, false)
	slow.Reserve(slowPlan)
	assert.Equal(t, 3680.0, slot(slowPlan, now, 2).Power)
	assert.Equal(t, 3680.0, slot(slowPlan, now, 0).Power)

	// the fast charger outranks it and takes the same slots regardless
	fastPlan := fast.Plan(2*time.Hour, 0, f.target, false)
	fast.Reserve(fastPlan)
	assert.Equal(t, 7360.0, slot(fastPlan, now, 2).Power)
	assert.Equal(t, 7360.0, slot(fastPlan, now, 0).Power)
	assert.Equal(t, 2*time.Hour, Duration(fastPlan))

	// the slow charger now plans around the fast charger: circuit is full in hours 0 and 2
	slowPlan = slow.Plan(2*time.Hour, 0, f.target, false)
	slow.Reserve(slowPlan)
	assert.True(t, slot(slowPlan, now, 0).IsZero())
	assert.True(t, slot(slowPlan, now, 2).IsZero())
	assert.Equal(t, 3680.0, slot(slowPlan, now, 4).Power)
	assert.Equal(t, 3680.0, slot(slowPlan, now, 1).Power)
	assert.Equal(t, 2*time.Hour, Duration(slowPlan))

	// the fast charger keeps its plan, the ledger has converged
	assert.Equal(t, fastPlan, fast.Plan(2*time.Hour, 0, f.target, false))

	// once the fast charger is done the slow charger gets the cheap slots back
	fast.Reserve(nil)
	slowPlan = slow.Plan(2*time.Hour, 0, f.target, false)
	assert.Equal(t, 3680.0, slot(slowPlan, now, 2).Power)
	assert.Equal(t, 3680.0, slot(slowPlan, now, 0).Power)
}

func TestLedgerPartialShare(t *testing.T) {
	// 16 A charger with priority leaves half the circuit to the 32 A charger
	f := newLedgerFixture(t, 7360)
	slow := f.planner(0, 1, 3680)
	fast := f.planner(1, 0, 7360)
	now := f.clock.Now()

	slow.Reserve(slow.Plan(time.Hour, 0, f.target, false))

	// the cheapest hour is shared at half power and counts for 30 minutes, the rest comes from the next cheapest hour
	fastPlan := fast.Plan(time.Hour, 0, f.target, false)
	assert.Equal(t, api.Rates{
		{Start: now.Add(30 * time.Minute), End: now.Add(time.Hour), Value: 20, Power: 7360},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 10, Power: 3680},
	}, fastPlan)
	assert.Equal(t, 90*time.Minute, Duration(fastPlan))
}

func TestLedgerBelowMinPower(t *testing.T) {
	f := newLedgerFixture(t, 4600)
	first := f.planner(0, 0, 3680)
	second := f.planner(1, 0, 3680)
	now := f.clock.Now()

	first.Reserve(first.Plan(time.Hour, 0, f.target, false))

	// 920 W left in the cheapest hour is below the minimum, so the slot is skipped entirely
	secondPlan := second.Plan(time.Hour, 0, f.target, false)
	assert.True(t, slot(secondPlan, now, 2).IsZero())
	assert.Equal(t, 3680.0, slot(secondPlan, now, 0).Power)
	assert.Equal(t, time.Hour, Duration(secondPlan))
}

func TestLedgerContinuousKeepsWindow(t *testing.T) {
	f := newLedgerFixture(t, 7360)
	now := f.clock.Now()

	// the outranking loadpoint fills the circuit for five of the six hours
	f.ledger.Reserve(Owner{Id: 0, Priority: 1, Circuit: f.circuit}, api.Rates{{Start: now, End: now.Add(5 * time.Hour), Power: 7360}})

	// the continuous plan is the cheapest window at full power, not the simple plan
	plan := f.planner(1, 0, 7360).Plan(2*time.Hour, 0, f.target, true)
	assert.Equal(t, api.Rates{
		{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour), Value: 60, Power: 7360},
		{Start: now.Add(2 * time.Hour), End: now.Add(3 * time.Hour), Value: 10, Power: 7360},
	}, plan)
}

func TestLedgerStaleReservation(t *testing.T) {
	f := newLedgerFixture(t, 7360)
	f.ledger.clock = f.clock

	window := api.Rate{Start: f.clock.Now(), End: f.clock.Now().Add(time.Hour)}
	f.ledger.Reserve(Owner{Id: 0, Priority: 1, Circuit: f.circuit}, api.Rates{{Start: window.Start, End: window.End, Power: 7360}})

	other := Owner{Id: 1, Priority: 0, Circuit: f.circuit}
	assert.Equal(t, 0.0, f.ledger.Available(other, window))

	// a reservation not refreshed for a slot no longer counts
	f.clock.Add(tariff.SlotDuration + time.Second)
	assert.Equal(t, 7360.0, f.ledger.Available(other, window))
}

func TestLedgerParentCircuit(t *testing.T) {
	ctrl := gomock.NewController(t)
	parent := mockCircuit(ctrl, 5000, nil)
	child := mockCircuit(ctrl, 0, parent)

	ledger := NewLedger()
	now := time.Now()
	window := api.Rate{Start: now, End: now.Add(time.Hour)}

	// higher priority loadpoint on the parent circuit reserves 3 kW
	ledger.Reserve(Owner{Id: 0, Priority: 1, Circuit: parent}, api.Rates{{Start: now, End: now.Add(time.Hour), Power: 3000}})

	other := Owner{Id: 1, Priority: 0, Circuit: child}
	assert.Equal(t, 2000.0, ledger.Available(other, window))
	assert.Equal(t, 5000.0, ledger.Available(other, api.Rate{Start: now.Add(time.Hour), End: now.Add(2 * time.Hour)}))

	// the reservation of a lower ranked loadpoint is ignored
	assert.Equal(t, 5000.0, ledger.Available(Owner{Id: 2, Priority: 2, Circuit: child}, window))

	// no circuit means no limit
	assert.True(t, ledger.Available(Owner{Id: 3}, window) > 1e9)
}
