package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The solar share of a session is derived in two steps that fail in different ways, so they are tested
// separately:
//
//  1. `Site.greenShare` answers *what the site looks like right now* — a pure function of the meter state.
//     TestGreenShare (site_test.go) covers its arithmetic; TestGreenShareNightBelow pins the two
//     definitional choices that made our measurements look wrong and that are deliberate, not bugs.
//
//  2. `EnergyMetrics` turns a moving counter into a share. Until 2026-11-04 it applied the *snapshot* taken
//     when the counter moved to the whole step, which is wrong for every device whose counter moves in
//     bursts — see EnergyMetrics.Sample. TestEnergyMetricsSample writes that defect as the timelines we
//     measured and asserts both the old and the new answer, so the difference is on the record.

const (
	gridTariff   = 0.33 // EUR/kWh, the plant's grid tariff
	feedinTariff = 0.08 // EUR/kWh, the feed-in tariff
)

// effectivePriceAt mirrors Site.effectivePrice, which needs tariff objects this test does not build. It is
// stated here so a change to the mix shows up as a failing expectation instead of passing silently.
func effectivePriceAt(greenShare float64) *float64 {
	v := gridTariff*(1-greenShare) + feedinTariff*greenShare
	return &v
}

// TestGreenShareNightBelow pins what the *instantaneous* share says in the situations that produced the
// wrong numbers — including one deliberate definitional choice of evcc's that is easy to mistake for a bug.
func TestGreenShareNightBelow(t *testing.T) {
	t.Run("night, grid only", func(t *testing.T) {
		// 1200 W purchased, the pump takes 500 of it, 700 W is the rest of the house
		s := &Site{siteState: siteState{gridPower: 1200, pvPower: 0}}

		// the house's share — what a heating device is given
		assert.Equal(t, 0.0, s.greenShare(0, 700))
	})

	t.Run("night, battery discharging counts as green", func(t *testing.T) {
		// **This is evcc's definition, pinned so it is a decision and not a surprise**: `greenPower` is
		// `max(0, pv) + max(0, battery.Power)`, so a battery *discharging* at night is green power. The
		// energy may have been bought off the grid the day before; the share cannot tell.
		s := &Site{siteState: siteState{gridPower: 0, pvPower: 0, battery: types.BatteryState{Power: 1200}}}
		assert.Equal(t, 1.0, s.greenShare(0, 700), "battery discharge is counted as green")

		// the mirror case — battery charging on a dark evening — answers 0, which booked whole days at 0 %
		s = &Site{siteState: siteState{gridPower: 1200, pvPower: 0, battery: types.BatteryState{Power: -1200}}}
		assert.Equal(t, 0.0, s.greenShare(0, 700))
	})

	t.Run("no loadpoint drawing is a 0/0 answered as 1", func(t *testing.T) {
		// power == 0 makes the quotient NaN, and greenShare resolves that to 1 whenever any green power
		// exists. For a self-metering device read at the wrong instant this was the 98 % session.
		s := &Site{siteState: siteState{gridPower: 0, pvPower: 800}}

		assert.Equal(t, 1.0, s.greenShare(500, 500), "NaN resolved to 1 — the reason 0011 exists")
		// the house share has no such hole: homePower is never 0 while anything runs
		assert.Equal(t, 1.0, s.greenShare(0, 500))
	})
}

// TestEnergyMetricsSample compares what the two strategies answer for the same night. Every phase has the
// measured shape: a device that reports its counter in bursts, on a site whose mix changes in between.
func TestEnergyMetricsSample(t *testing.T) {
	// one phase: hold an environment for `minutes` of polls, then optionally let the counter move
	type phase struct {
		minutes     float64
		power       float64 // W, the loadpoint's own draw
		greenShare  float64
		moveCounter bool // the device's counter advances at the end of this phase
	}

	// run replays the phases against an EnergyMetrics. `sampled=false` reproduces the pre-2026-11-04
	// behaviour (the environment is a snapshot, no power weighted), so the two can be compared side by side.
	//
	// Sampling happens at the *end* of each minute and covers the minute before it, which is what a poll
	// does: the value read now is the value that was valid since the previous read.
	run := func(sampled bool, phases []phase, price bool) EnergyMetrics {
		var em EnergyMetrics
		start := time.Unix(1735689600, 0)

		// seed the window: the first sample only records its own instant
		em.Sample(0, start)

		for _, p := range phases {
			for m := 0.0; m < p.minutes; m++ {
				start = start.Add(time.Minute)
				if price {
					em.SetEnvironment(p.greenShare, effectivePriceAt(p.greenShare), nil)
				} else {
					em.SetEnvironment(p.greenShare, nil, nil)
				}
				if sampled {
					em.Sample(p.power, start)
				}
			}

			if p.moveCounter {
				kWh := p.power * p.minutes / 60 / 1e3
				em.Update(em.TotalWh()/1e3 + kWh)
			}
		}

		return em
	}

	t.Run("constant environment: sampling changes nothing", func(t *testing.T) {
		phases := []phase{
			{minutes: 30, power: 500, greenShare: 0.4},
			{minutes: 30, power: 500, greenShare: 0.4, moveCounter: true},
		}

		weighted, snapshot := run(true, phases, false), run(false, phases, false)

		assert.InDelta(t, snapshot.SolarPercentage(), weighted.SolarPercentage(), 1e-9)
		assert.InDelta(t, 40, weighted.SolarPercentage(), 1e-9)
	})

	t.Run("night: import first, battery discharge when the counter moves", func(t *testing.T) {
		// 30 min on grid import (share 0), then 30 min on battery (share 1). The pump draws 500 W
		// throughout and its counter moves only at the end — the measured shape.
		phases := []phase{
			{minutes: 30, power: 500, greenShare: 0},
			{minutes: 30, power: 500, greenShare: 1, moveCounter: true},
		}

		weighted, snapshot := run(true, phases, false), run(false, phases, false)

		assert.InDelta(t, 100, snapshot.SolarPercentage(), 1e-9, "the defect, reproduced")
		assert.InDelta(t, 50, weighted.SolarPercentage(), 1e-9, "the fix")
	})

	t.Run("a resting device whose counter catches up keeps the current share", func(t *testing.T) {
		// power 0 the whole time (evcc never saw it run — the burst arrived later), but the counter moves.
		// There is no window to weight, so the environment of the moment is the honest answer.
		var em EnergyMetrics
		start := time.Unix(1735689600, 0)

		em.SetEnvironment(0.3, effectivePriceAt(0.3), nil)
		for m := 0; m < 10; m++ {
			em.Sample(0, start.Add(time.Duration(m)*time.Minute))
		}
		em.Update(0.5)

		assert.InDelta(t, 30, em.SolarPercentage(), 1e-9)
		assert.InDelta(t, 0.5, em.TotalWh()/1e3, 1e-9)
	})

	t.Run("the price follows the same weighted share", func(t *testing.T) {
		phases := []phase{
			{minutes: 30, power: 500, greenShare: 0},
			{minutes: 30, power: 500, greenShare: 1, moveCounter: true},
		}

		weighted, snapshot := run(true, phases, true), run(false, phases, true)

		require.NotNil(t, weighted.PricePerKWh())
		assert.InDelta(t, (gridTariff+feedinTariff)/2, *weighted.PricePerKWh(), 1e-9, "half grid, half feed-in")

		require.NotNil(t, snapshot.PricePerKWh())
		assert.InDelta(t, feedinTariff, *snapshot.PricePerKWh(), 1e-9, "the defect: feed-in tariff for grid energy")
	})

	t.Run("successive counter steps only book their increment", func(t *testing.T) {
		var em EnergyMetrics
		start := time.Unix(1735689600, 0)

		em.SetEnvironment(0.5, nil, nil)
		for m := 0; m < 60; m++ {
			em.Sample(1000, start.Add(time.Duration(m)*time.Minute))
		}

		// a device whose counter reports three times while the window is still open (a catch-up after a
		// restart): the total must still be the counter's own value, not the sum of the steps
		em.Update(0.2)
		em.Update(0.5)
		em.Update(1.0)

		assert.InDelta(t, 1.0, em.TotalWh()/1e3, 1e-9)
		assert.InDelta(t, 0.5, em.solarKWh, 1e-9)
	})

	t.Run("a counter reset books nothing", func(t *testing.T) {
		var em EnergyMetrics
		start := time.Unix(1735689600, 0)

		em.SetEnvironment(1, nil, nil)
		em.Update(5.0)
		em.Sample(1000, start.Add(time.Minute))

		added, addedGreen := em.Update(0.1)
		assert.Zero(t, added)
		assert.Zero(t, addedGreen)
		assert.InDelta(t, 5.0, em.TotalWh()/1e3, 1e-9, "a lower reading is not a negative step")
		assert.InDelta(t, 5.0, em.solarKWh, 1e-9)
	})

	t.Run("reset starts a new window", func(t *testing.T) {
		var em EnergyMetrics
		start := time.Unix(1735689600, 0)

		em.SetEnvironment(1, nil, nil)
		em.Sample(1000, start)
		em.Sample(1000, start.Add(time.Minute))

		em.Reset()

		// the first sample after a reset only records its instant: no interval, no energy
		em.SetEnvironment(0, nil, nil)
		em.Sample(1000, start.Add(10*time.Minute))
		em.Update(0.1)

		assert.True(t, em.lastSample.Equal(start.Add(10*time.Minute)))
	})
}

// TestHeatingDaySolarShare runs one night through the real Site and the real EnergyMetrics in the order the
// loadpoint tick uses them. It is the end-to-end counterpart of TestEnergyMetricsSample: the share comes
// from `Site.greenShare`, not from a constant in the test.
func TestHeatingDaySolarShare(t *testing.T) {
	clk := clock.NewMock()
	clk.Set(time.Date(2026, 11, 4, 0, 0, 0, 0, time.Local))

	site := &Site{
		log:       util.NewLogger("foo"),
		siteState: siteState{gridPower: 1200, pvPower: 0},
	}

	var em EnergyMetrics

	// 1200 W purchased for the first half hour, then the battery took over. The pump draws 500 W the whole
	// time; the sample at each minute covers the minute before it.
	em.Sample(0, clk.Now()) // seed the window

	for minute := 1; minute <= 60; minute++ {
		clk.Add(time.Minute)

		if minute <= 30 {
			site.siteState = siteState{gridPower: 1200, pvPower: 0}
		} else {
			site.siteState = siteState{gridPower: 0, pvPower: 0, battery: types.BatteryState{Power: 1200}}
		}

		em.SetEnvironment(site.greenShare(0, 1200), nil, nil)
		em.Sample(500, clk.Now())
	}

	// the device's counter moves once, at the end of the hour
	em.Update(0.5)

	assert.InDelta(t, 50, em.SolarPercentage(), 1e-6,
		"half the energy on import, half on the battery — the weighted share, not the instant the counter moved")
	assert.InDelta(t, 0.5, em.TotalWh()/1e3, 1e-9)
}

// TestLoadpointSamplesChargePower is the wiring check for EnergyMetrics.Sample: the accumulator only ever
// sees a value if the loadpoint hands its charge power in on every tick.
func TestLoadpointSamplesChargePower(t *testing.T) {
	clk := clock.NewMock()
	clk.Set(time.Date(2026, 11, 4, 12, 0, 0, 0, time.Local))

	lp := &Loadpoint{
		log:   util.NewLogger("foo"),
		clock: clk,
	}

	lp.chargePower = 800
	lp.energyMetrics.SetEnvironment(1, nil, nil)
	lp.energyMetrics.Sample(lp.chargePower, lp.clock.Now())

	clk.Add(10 * time.Minute)
	lp.energyMetrics.Sample(lp.chargePower, lp.clock.Now())

	assert.InDelta(t, 0.8*10.0/60, lp.energyMetrics.sampledKWh, 1e-9, "10 min at 800 W")
}

// TestEnergyMetricsRestore is the restart scenario, measured on the running system on 2026-11-04 13:35: the
// "Brauchwasser" day row stood at 6,82 kWh / 62,0 % and read 7,33 kWh / 18,4 % a quarter of an hour later.
// The kWh grew by 0,51 while the green energy *fell* from 4,23 to 1,35 kWh — which no accumulator can do, so
// the metrics had been reset by the restart and the whole day re-booked at the moment's share.
func TestEnergyMetricsRestore(t *testing.T) {
	f := func(f float64) *float64 { return &f }

	const (
		bookedKWh = 6.82
		booked    = 62.0
		afterKWh  = 7.33 // the counter move after the restart
	)

	t.Run("a restart keeps the day's share", func(t *testing.T) {
		var em EnergyMetrics

		// what the row says when the process comes back
		em.Restore(bookedKWh, booked, f(0.175), nil)
		assert.InDelta(t, booked, em.SolarPercentage(), 1e-9, "the restore reproduces the row")

		// the environment after the restart is a grey afternoon
		em.SetEnvironment(0.18, f(0.18), nil)

		em.Update(afterKWh)

		// 4.23 kWh of green from before, plus 0.51 kWh at 18 %
		want := (bookedKWh*booked/100 + (afterKWh-bookedKWh)*0.18) / afterKWh * 100
		assert.InDelta(t, want, em.SolarPercentage(), 1e-9)
		assert.InDelta(t, 58.8, em.SolarPercentage(), 0.6, "the day keeps its history")
	})

	t.Run("without the restore the whole day is re-valued", func(t *testing.T) {
		var em EnergyMetrics

		em.SetEnvironment(0.18, f(0.18), nil)
		em.Update(afterKWh) // a fresh accumulator, given the cumulative day delta

		assert.InDelta(t, 18, em.SolarPercentage(), 1e-9, "the defect, reproduced")
	})

	t.Run("a row without a share is not restored", func(t *testing.T) {
		var em EnergyMetrics

		em.Restore(0, 0, nil, nil)
		assert.Zero(t, em.TotalWh())
	})
}
