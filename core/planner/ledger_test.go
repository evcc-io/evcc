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

func TestLedgerSharedCircuit(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	// 32 A single phase main circuit
	circuit := api.NewMockCircuit(ctrl)
	circuit.EXPECT().GetMaxPower().Return(7360.0).AnyTimes()
	circuit.EXPECT().GetParent().Return(nil).AnyTimes()

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	target := clock.Now().Add(6 * time.Hour)
	ledger := NewLedger()

	owner := func(id, priority int, maxPower float64) func() Owner {
		return func() Owner {
			return Owner{Id: id, Priority: priority, Target: target, Circuit: circuit, MaxPower: maxPower, MinPower: 1380}
		}
	}

	// 32 A charger with priority, 16 A charger without
	fast := New(util.NewLogger("fast"), trf, WithLedger(ledger, owner(0, 1, 7360)), func(p *Planner) { p.clock = clock })
	slow := New(util.NewLogger("slow"), trf, WithLedger(ledger, owner(1, 0, 3680)), func(p *Planner) { p.clock = clock })

	// the slow charger plans first and takes the cheapest slots
	slowPlan := slow.Plan(2*time.Hour, 0, target, false)
	slow.Reserve(slowPlan)
	assert.Equal(t, 3680.0, slot(slowPlan, clock.Now(), 2).Power)
	assert.Equal(t, 3680.0, slot(slowPlan, clock.Now(), 0).Power)

	// the fast charger outranks it and takes the same slots regardless
	fastPlan := fast.Plan(2*time.Hour, 0, target, false)
	fast.Reserve(fastPlan)
	assert.Equal(t, 7360.0, slot(fastPlan, clock.Now(), 2).Power)
	assert.Equal(t, 7360.0, slot(fastPlan, clock.Now(), 0).Power)
	assert.Equal(t, 2*time.Hour, Duration(fastPlan))

	// the slow charger now plans around the fast charger: circuit is full in hours 0 and 2
	slowPlan = slow.Plan(2*time.Hour, 0, target, false)
	slow.Reserve(slowPlan)
	assert.True(t, slot(slowPlan, clock.Now(), 0).IsZero())
	assert.True(t, slot(slowPlan, clock.Now(), 2).IsZero())
	assert.Equal(t, 3680.0, slot(slowPlan, clock.Now(), 4).Power)
	assert.Equal(t, 3680.0, slot(slowPlan, clock.Now(), 1).Power)
	assert.Equal(t, 2*time.Hour, Duration(slowPlan))

	// the fast charger keeps its plan, the ledger has converged
	assert.Equal(t, fastPlan, fast.Plan(2*time.Hour, 0, target, false))

	// once the fast charger is done the slow charger gets the cheap slots back
	fast.Reserve(nil)
	slowPlan = slow.Plan(2*time.Hour, 0, target, false)
	assert.Equal(t, 3680.0, slot(slowPlan, clock.Now(), 2).Power)
	assert.Equal(t, 3680.0, slot(slowPlan, clock.Now(), 0).Power)
}

func TestLedgerPartialShare(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	circuit := api.NewMockCircuit(ctrl)
	circuit.EXPECT().GetMaxPower().Return(7360.0).AnyTimes()
	circuit.EXPECT().GetParent().Return(nil).AnyTimes()

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	target := clock.Now().Add(6 * time.Hour)
	ledger := NewLedger()

	// 16 A charger with priority leaves half the circuit to the 32 A charger
	slow := New(util.NewLogger("slow"), trf, WithLedger(ledger, func() Owner {
		return Owner{Id: 0, Priority: 1, Target: target, Circuit: circuit, MaxPower: 3680, MinPower: 1380}
	}), func(p *Planner) { p.clock = clock })
	fast := New(util.NewLogger("fast"), trf, WithLedger(ledger, func() Owner {
		return Owner{Id: 1, Priority: 0, Target: target, Circuit: circuit, MaxPower: 7360, MinPower: 1380}
	}), func(p *Planner) { p.clock = clock })

	slow.Reserve(slow.Plan(time.Hour, 0, target, false))

	// the cheapest hour is shared at half power and counts for 30 minutes, the rest comes from the next cheapest hour
	fastPlan := fast.Plan(time.Hour, 0, target, false)
	assert.Equal(t, api.Rates{
		{Start: clock.Now().Add(30 * time.Minute), End: clock.Now().Add(time.Hour), Value: 20, Power: 7360},
		{Start: clock.Now().Add(2 * time.Hour), End: clock.Now().Add(3 * time.Hour), Value: 10, Power: 3680},
	}, fastPlan)
	assert.Equal(t, 90*time.Minute, Duration(fastPlan))
}

func TestLedgerBelowMinPower(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	circuit := api.NewMockCircuit(ctrl)
	circuit.EXPECT().GetMaxPower().Return(4600.0).AnyTimes()
	circuit.EXPECT().GetParent().Return(nil).AnyTimes()

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	target := clock.Now().Add(6 * time.Hour)
	ledger := NewLedger()

	first := New(util.NewLogger("first"), trf, WithLedger(ledger, func() Owner {
		return Owner{Id: 0, Priority: 0, Target: target, Circuit: circuit, MaxPower: 3680, MinPower: 1380}
	}), func(p *Planner) { p.clock = clock })
	second := New(util.NewLogger("second"), trf, WithLedger(ledger, func() Owner {
		return Owner{Id: 1, Priority: 0, Target: target, Circuit: circuit, MaxPower: 3680, MinPower: 1380}
	}), func(p *Planner) { p.clock = clock })

	first.Reserve(first.Plan(time.Hour, 0, target, false))

	// 920 W left in the cheapest hour is below the minimum, so the slot is skipped entirely
	secondPlan := second.Plan(time.Hour, 0, target, false)
	assert.True(t, slot(secondPlan, clock.Now(), 2).IsZero())
	assert.Equal(t, 3680.0, slot(secondPlan, clock.Now(), 0).Power)
	assert.Equal(t, time.Hour, Duration(secondPlan))
}

func TestLedgerContinuousKeepsWindow(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	circuit := api.NewMockCircuit(ctrl)
	circuit.EXPECT().GetMaxPower().Return(7360.0).AnyTimes()
	circuit.EXPECT().GetParent().Return(nil).AnyTimes()

	trf := api.NewMockTariff(ctrl)
	trf.EXPECT().Rates().AnyTimes().Return(rates([]float64{20, 60, 10, 80, 40, 90}, clock.Now(), time.Hour), nil)

	target := clock.Now().Add(6 * time.Hour)
	ledger := NewLedger()

	// the outranking loadpoint fills the circuit for five of the six hours
	ledger.Reserve(Owner{Id: 0, Priority: 1, Circuit: circuit}, api.Rates{{Start: clock.Now(), End: clock.Now().Add(5 * time.Hour), Power: 7360}})

	p := New(util.NewLogger("foo"), trf, WithLedger(ledger, func() Owner {
		return Owner{Id: 1, Priority: 0, Target: target, Circuit: circuit, MaxPower: 7360, MinPower: 1380}
	}), func(p *Planner) { p.clock = clock })

	// the continuous plan is the cheapest window at full power, not the simple plan
	plan := p.Plan(2*time.Hour, 0, target, true)
	assert.Equal(t, api.Rates{
		{Start: clock.Now().Add(time.Hour), End: clock.Now().Add(2 * time.Hour), Value: 60, Power: 7360},
		{Start: clock.Now().Add(2 * time.Hour), End: clock.Now().Add(3 * time.Hour), Value: 10, Power: 7360},
	}, plan)
}

func TestLedgerStaleReservation(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	circuit := api.NewMockCircuit(ctrl)
	circuit.EXPECT().GetMaxPower().Return(7360.0).AnyTimes()
	circuit.EXPECT().GetParent().Return(nil).AnyTimes()

	ledger := NewLedger()
	ledger.clock = clock

	window := api.Rate{Start: clock.Now(), End: clock.Now().Add(time.Hour)}
	ledger.Reserve(Owner{Id: 0, Priority: 1, Circuit: circuit}, api.Rates{{Start: window.Start, End: window.End, Power: 7360}})

	other := Owner{Id: 1, Priority: 0, Circuit: circuit}
	assert.Equal(t, 0.0, ledger.Available(other, window))

	// a reservation not refreshed for a slot no longer counts
	clock.Add(tariff.SlotDuration + time.Second)
	assert.Equal(t, 7360.0, ledger.Available(other, window))
}

func TestLedgerParentCircuit(t *testing.T) {
	ctrl := gomock.NewController(t)

	parent := api.NewMockCircuit(ctrl)
	parent.EXPECT().GetMaxPower().Return(5000.0).AnyTimes()
	parent.EXPECT().GetParent().Return(nil).AnyTimes()

	child := api.NewMockCircuit(ctrl)
	child.EXPECT().GetMaxPower().Return(0.0).AnyTimes()
	child.EXPECT().GetParent().Return(parent).AnyTimes()

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
