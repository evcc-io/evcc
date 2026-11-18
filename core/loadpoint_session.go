package core

import (
	"errors"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/evcc-io/evcc/tariff"
)

func (lp *Loadpoint) chargeMeterTotal() float64 {
	m, ok := api.Cap[api.MeterEnergy](lp.chargeMeter)
	if !ok {
		return 0
	}

	f, err := m.TotalEnergy()
	if err != nil {
		if !errors.Is(err, api.ErrNotAvailable) {
			lp.log.ERROR.Printf("charge total import: %v", err)
		}
		return 0
	}

	lp.log.DEBUG.Printf("charge total import: %.3fkWh", f)

	return f
}

// heatingDevice reports a heating loadpoint that meters itself: the device's own counter is
// the truth, not the power evcc integrates while it has the device switched on
func (lp *Loadpoint) heatingDevice() bool {
	return lp.chargerHasFeature(api.Heating) && lp.chargerHasFeature(api.IntegratedDevice)
}

// sessionChargeEnergy returns the energy charged in the current session in kWh
func (lp *Loadpoint) sessionChargeEnergy() (float64, error) {
	if s := lp.session; lp.heatingDevice() && s != nil && s.MeterStart != nil {
		if f := lp.chargeMeterTotal(); f >= *s.MeterStart {
			return f - *s.MeterStart, nil
		}

		return 0, nil
	}

	f, err := lp.chargeRater.ChargedEnergy()
	if err != nil {
		return 0, err
	}

	return f - lp.chargedAtStartup, nil
}

// createSession creates a charging session. The created timestamp is empty until set by evChargeStartHandler.
// The session is not persisted yet. That will only happen when stopSession is called.
func (lp *Loadpoint) createSession() {
	// the guard has to run before the meter is read: a loadpoint without a charge meter would otherwise
	// dereference it (measured as a nil-pointer panic in TestDefaultVehicle)
	if lp.db == nil || lp.session != nil {
		return
	}

	lp.createSessionAt(lp.chargeMeterTotal())
}

// createSessionAt is createSession with an explicit start meter (see applyEnergyMetricsAt).
func (lp *Loadpoint) createSessionAt(meter float64) {
	// test guard
	if lp.db == nil || lp.session != nil {
		return
	}

	lp.session = lp.db.New(meter)

	if v := lp.GetVehicle(); v != nil {
		lp.session.Vehicle = v.GetTitle()
	} else if lp.chargerHasFeature(api.IntegratedDevice) {
		lp.session.Vehicle = lp.GetTitle()
	}

	if c, ok := api.Cap[api.Identifier](lp.charger); ok {
		if ids, err := c.Identify(); err == nil {
			if idx := slices.IndexFunc(ids, func(id string) bool { return id != "" }); idx >= 0 {
				lp.session.Identifier = ids[idx]
			}
		}
	}

	if lp.site != nil {
		lp.session.ReferencePricePerKWh = tariff.AverageRate(lp.site.GetTariff(api.TariffUsageGrid), 24*time.Hour)
		lp.session.ReferenceCo2PerKWh = tariff.AverageRate(lp.site.GetTariff(api.TariffUsageCo2), 24*time.Hour)
	}

	// energy
	lp.energyMetrics.Reset()
	if _, estimator := lp.getVehicleAndSocEstimator(); estimator != nil {
		estimator.Reset()
	}
	lp.energyMetrics.Publish("session", lp)
	lp.publish(keys.ChargedEnergy, lp.GetChargedEnergy())
}

// applyEnergyMetrics writes current energy metrics into the session and persists it.
func (lp *Loadpoint) applyEnergyMetrics(s *session.Session) {
	lp.applyEnergyMetricsAt(s, lp.chargeMeterTotal())
}

// applyEnergyMetricsAt is applyEnergyMetrics with an explicit meter reading, so a session that has to end
// on a boundary no reading falls on (midnight) books the interpolated value.
func (lp *Loadpoint) applyEnergyMetricsAt(s *session.Session, meterStop float64) {
	if meterStop > 0 {
		s.MeterStop = &meterStop
	}

	// a heating device meters itself: book the counter delta, not the windows evcc switched it on
	if lp.heatingDevice() && s.MeterStart != nil && meterStop >= *s.MeterStart {
		lp.energyMetrics.Update(meterStop - *s.MeterStart)
	}

	if soc := lp.vehicleSoc; soc > 0 && !lp.chargerHasFeature(api.Heating) {
		s.SocEnd = &soc
	}

	// added range by linear projection from current range and soc delta
	if rng := float64(lp.vehicleRange); rng > 0 && s.SocStart != nil && s.SocEnd != nil && *s.SocEnd > *s.SocStart {
		added := rng * (*s.SocEnd - *s.SocStart) / *s.SocEnd
		s.AddedRange = &added
	}

	s.SolarPercentage = new(lp.energyMetrics.SolarPercentage())
	s.Price = lp.energyMetrics.Price()
	s.PricePerKWh = lp.energyMetrics.PricePerKWh()
	s.Co2PerKWh = lp.energyMetrics.Co2PerKWh()
	s.ChargedEnergy = lp.energyMetrics.TotalWh() / 1e3

	lp.db.Persist(s)
}

// persistSessionOnShutdown writes the last meter reading when evcc stops. A self-metering heater needs
// this and nothing else: its session is the day, and closing it at shutdown made the next start open a
// second row for the same day (measured 2026-11-02: every restart closed the day row at 08:54:52 and
// opened another at 08:55:03). Patch 0008.
func (lp *Loadpoint) persistSessionOnShutdown() {
	if lp.heatingDevice() {
		if s := lp.session; s != nil && s.MeterStart != nil {
			lp.applyEnergyMetricsAt(s, lp.chargeMeterTotal())
		}

		return
	}

	// upstream's shutdown flush: the rater's energy is read one last time before the session ends
	lp.finalizeSessionEnergy()
	lp.stopSession()
}

// stopSession ends a charging session segment and persists the session.
func (lp *Loadpoint) stopSession() {
	if lp.db == nil || lp.session == nil {
		return
	}

	lp.stopSessionAt(lp.clock.Now(), lp.chargeMeterTotal())
}

// stopSessionAt ends the session at the given time with the given meter reading. A heating session ends
// on the day boundary (23:59:59), which is not a tick: the meter reading for that instant is interpolated.
func (lp *Loadpoint) stopSessionAt(ts time.Time, meterStop float64) {
	s := lp.session

	// test guard
	if lp.db == nil || s == nil {
		return
	}

	// abort the session if charging has never started
	if s.Created.IsZero() {
		return
	}

	s.Finished = ts
	s.ChargeDuration = new(lp.chargeDuration.Abs())

	lp.applyEnergyMetricsAt(s, meterStop)
}

type sessionOption func(*session.Session)

// updateSession updates any parameter of a charging session and persists the session.
func (lp *Loadpoint) updateSession(opts ...sessionOption) {
	// test guard
	if lp.db == nil || lp.session == nil {
		return
	}

	for _, opt := range opts {
		opt(lp.session)
	}

	if !lp.session.Created.IsZero() {
		lp.db.Persist(lp.session)
	}
}

// clearSession clears the charging session without persisting it.
func (lp *Loadpoint) clearSession() {
	// test guard
	if lp.db == nil {
		return
	}

	lp.session = nil
}

func (lp *Loadpoint) finalizeSessionEnergy() {
	s := lp.session
	if lp.db == nil || s == nil || s.Created.IsZero() {
		return
	}

	f, err := lp.chargeRater.ChargedEnergy()
	if err != nil {
		lp.log.ERROR.Printf("session energy: %v", err)
		return
	}

	chargedKWh := f - lp.chargedAtStartup
	if chargedKWh <= s.ChargedEnergy {
		return
	}

	lp.log.DEBUG.Printf("session energy: %.3f -> %.3fkWh", s.ChargedEnergy, chargedKWh)

	lp.energyMetrics.Update(chargedKWh)

	lp.applyEnergyMetrics(s)
}

// updateDaySession runs on every loadpoint tick and keeps each session inside one day: a session still open
// when the day turns is cut at 23:59:59 and continued at 00:00:00, so the kWh consumed on a day is booked to
// that day — whether it comes from a device counter, a charge meter or evcc own integration of charge power.
// Nothing here is keyed to a device type; the two branches differ only in what the day means for the session.
func (lp *Loadpoint) updateDaySession() {
	nowT := lp.clock.Now()
	// The boundary is midnight **of the clock's own location**, built explicitly. `now.With(t).BeginningOfDay()`
	// keeps whatever location the time carries, so a process with the wrong TZ writes a plausible day row for the
	// wrong day — measured 2026-11-01: a development run in +08 wrote 13 rows dated `2026-11-02 00:00:00+08:00`,
	// i.e. 17:00 the previous afternoon, and they read as "yesterday landed on today" in the UI.
	dayStart := time.Date(nowT.Year(), nowT.Month(), nowT.Day(), 0, 0, 0, 0, nowT.Location())

	if lp.heatingDevice() {
		lp.updateHeatingDaySession(nowT, dayStart)
		return
	}

	lp.splitSessionAtDayBoundary(nowT, dayStart)
}

// splitSessionAtDayBoundary cuts an open session at the day boundary: the part before midnight is closed
// with the reading of that instant and persisted, the rest continues as a session of its own day. A
// loadpoint that is not delivering anything at the boundary simply ends its session there.
func (lp *Loadpoint) splitSessionAtDayBoundary(nowT, dayStart time.Time) {
	meterNow := lp.chargeMeterTotal()
	kWhNow := lp.energyMetrics.TotalWh() / 1e3

	if s := lp.session; lp.db != nil && s != nil && !s.Created.IsZero() && dayStart.After(s.Created) {
		meterAt, kWhAt := lp.readingsAt(dayStart, meterNow, kWhNow, nowT)

		// the closing part carries what was consumed before the boundary, and only that
		lp.energyMetrics = lp.energyMetricsBefore(kWhAt)
		lp.stopSessionAt(dayStart.Add(-time.Second), meterAt)
		lp.clearSession()

		// still delivering? then this day continues the session at the reading of the boundary
		if lp.charging() {
			if f, err := lp.chargeRater.ChargedEnergy(); err == nil {
				lp.chargedAtStartup = f
			}
			lp.energyMetrics.Reset()
			lp.createSessionAt(meterAt)

			if s := lp.session; s != nil {
				s.Created = dayStart
				lp.db.Persist(s)
			}
		} else {
			lp.energyMetrics.Reset()
		}

		lp.energyMetrics.Publish("session", lp)
	}

	lp.rememberReading(nowT, meterNow, kWhNow)
}

// updateHeatingDaySession is the day session of a device that meters itself and runs whether or not evcc
// switches it on: its own counter is the truth, so the day is the session, and the session exists from the
// first tick after midnight even while the device rests.
func (lp *Loadpoint) updateHeatingDaySession(nowT, dayStart time.Time) {
	meterNow := lp.chargeMeterTotal()

	// **Der Tag ist die Sitzung — und er endet an seinem letzten Tagessekunde**, nicht beim ersten Tick
	// danach. Vorher schloss der Tick nach Mitternacht die Sitzung: die Zeile hieß dann "00:02", trug aber
	// das Geld des Vortags bis zu diesem Tick (gemessen 2026-11-01: Brauchwasser 00:02:20, FBH 00:02:50,
	// Gewölbe 00:03:20; Bediener: "we shall strive to finalize the sessions at end of the previous day (at
	// 23:59 or even 23:59:59) so that we do not effectively shift the accounting by one day"). Die Sitzung
	// schließt jetzt auf 23:59:59 mit dem *interpolierten* Zählerstand dieser Sekunde, und die neue Sitzung
	// beginnt auf 00:00:00 mit demselben Stand — der Zähler wird also genau an der Tagesgrenze geteilt.
	var meterAtMidnight float64
	closed := false
	if s := lp.session; s != nil && dayStart.After(s.Created) {
		meterAtMidnight = lp.meterAt(dayStart, meterNow, nowT)

		// the closed part carries its own share of the cost and CO2, not the whole day's
		if m0 := s.MeterStart; m0 != nil && meterNow > *m0 {
			lp.energyMetrics = lp.energyMetricsShare((meterAtMidnight - *m0) / (meterNow - *m0))
		}

		lp.stopSessionAt(dayStart.Add(-time.Second), meterAtMidnight)
		lp.clearSession()
		closed = true

		if cr, ok := lp.chargeRater.(wrapper.ChargeResetter); ok {
			cr.ResetCharge()
		}
		if ct, ok := lp.chargeTimer.(wrapper.ChargeResetter); ok {
			ct.ResetCharge()
		}
	}

	// After a restart the day's row is already in the database. Adopt it instead of opening a second row
	// backdated to midnight: every restart used to split one day into another zero-energy row and left the
	// counter movement between shutdown and first tick unbooked (measured 2026-11-01: FBH 194/203/212, all
	// three dated 00:00, and 0.43 kWh missing between them).
	if lp.session == nil && !closed {
		if open := lp.db.LastOpen(); open != nil && sameDay(open.Created, dayStart) {
			lp.session = open
		} else if open != nil {
			// an open row from an earlier day is a leftover (evcc stopped before the boundary): close it
			// where it stands so it cannot shadow today's session forever
			f := open.Created.Add(24 * time.Hour).Add(-time.Second)
			if f.After(open.Created) {
				open.Finished = f
				lp.db.Persist(open)
			}
		}
	}

	// the day is the session: start it at the day boundary, so the device's counter is booked
	// from midnight even when evcc never switches the device on
	if lp.session == nil {
		if closed {
			lp.createSessionAt(meterAtMidnight)
		} else {
			lp.createSessionAt(meterNow)
		}
		if lp.session != nil && lp.session.Created.IsZero() {
			if closed {
				lp.session.Created = dayStart
			} else {
				lp.session.Created = nowT
			}
		}
	}

	// The session store only records a start meter above zero (DB.New: `if meter > 0`). A heater that
	// begins its counter at 0.0 kWh -- or a session restored after a restart, where the row was written
	// without one -- therefore keeps MeterStart nil, and every reader of this session short-circuits:
	// no counter delta, no energy, no money (measured 2026-10-30: both Stiebel sockets with
	// meter_start_kwh NULL and charged_kwh 0.0 while todayEnergy showed 22 Wh). Take the start from the
	// meter itself. publishChargeProgress runs before this in the same tick and may already have written
	// a stop meter, so reconcile with it rather than requiring it to be empty.
	if s := lp.session; s != nil && s.MeterStart == nil && !s.Created.IsZero() {
		if m, ok := api.Cap[api.MeterEnergy](lp.chargeMeter); ok {
			if f, err := m.TotalEnergy(); err == nil {
				if s.MeterStop == nil || *s.MeterStop < f {
					s.MeterStop = &f
				}
				s.MeterStart = &f
				lp.db.Persist(s)
			}
		}
	}

	// A resting heating device draws no power, so publishChargeProgress -- the only other writer -- never
	// runs and the day's session would stay in memory, invisible in the sessions list and absent from the
	// database until midnight (measured 2026-10-30: 0.28 kWh on the counter, no row). Show the session from
	// the first tick and keep it current, but write only when something changed: one write per counter step.
	if s := lp.session; s != nil && s.MeterStart != nil && !s.Created.IsZero() {
		switch {
		case s.MeterStop == nil:
			s.MeterStop = s.MeterStart
			lp.db.Persist(s)
		case meterNow > *s.MeterStop:
			lp.applyEnergyMetricsAt(s, meterNow)
		}
	}

	// keep the readings for the interpolation at the next day boundary
	lp.rememberReading(nowT, meterNow, lp.energyMetrics.TotalWh()/1e3)
}

// sameDay reports whether both instants fall on the same calendar day, compared in a's location.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()

	return ay == by && am == bm && ad == bd
}

// meterAt returns the counter reading at `at`, interpolated between the previous tick and this one (see
// ramp). Falls back to the current reading when there is no usable pair: first tick, counter reset.
func (lp *Loadpoint) meterAt(at time.Time, meterNow float64, nowT time.Time) float64 {
	return ramp(lp.dayReading.meter, meterNow, lp.dayReading.at, nowT, at)
}

// readingsAt returns both readings at `at`: the charge meter counter and evcc own session accumulator.
func (lp *Loadpoint) readingsAt(at time.Time, meterNow, kWhNow float64, nowT time.Time) (float64, float64) {
	return ramp(lp.dayReading.meter, meterNow, lp.dayReading.at, nowT, at),
		ramp(lp.dayReading.kWh, kWhNow, lp.dayReading.at, nowT, at)
}

// ramp interpolates v1 -> v0 across from..to at the instant at. A day boundary is not a tick, and booking a
// whole tick to one side of it would move up to one poll of energy into the wrong day. Anything that is not
// a clean rising pair (no sample, clock stepped back, counter reset, at outside the interval) falls back to
// the current reading — a wrong guess is worse than the value the tick itself measured.
func ramp(v1, v0 float64, from, to, at time.Time) float64 {
	if from.IsZero() || !to.After(from) || v0 <= v1 || at.Before(from) || at.After(to) {
		return v0
	}

	v := v1 + (v0-v1)*float64(at.Sub(from))/float64(to.Sub(from))
	if v < v1 {
		return v1
	}
	if v > v0 {
		return v0
	}

	return v
}

// rememberReading keeps this tick readings for the next day boundary.
func (lp *Loadpoint) rememberReading(nowT time.Time, meter, kWh float64) {
	lp.dayReading = energyReading{at: nowT, meter: meter, kWh: kWh}
}

// energyMetricsBefore returns the accumulator as it stood at the day boundary: the energy interpolated to
// that instant, with price, CO2 and solar energy scaled to the same share.
func (lp *Loadpoint) energyMetricsBefore(kWhAt float64) EnergyMetrics {
	if em := lp.energyMetrics; em.totalKWh > 0 {
		return lp.energyMetricsShare(kWhAt / em.totalKWh)
	}

	return lp.energyMetrics
}

// energyMetricsShare reduces the accumulator to the given share of its energy, price, CO2 and solar energy.
// The cost is linear in energy, so a part closed at the day boundary carries its own money instead of the
// whole session's — whichever way the session is booked.
func (lp *Loadpoint) energyMetricsShare(r float64) EnergyMetrics {
	em := lp.energyMetrics

	if r < 0 {
		r = 0
	}
	if r >= 1 {
		return em
	}

	em.totalKWh *= r
	em.solarKWh *= r

	if em.price != nil {
		v := *em.price * r
		em.price = &v
	}
	if em.co2 != nil {
		v := *em.co2 * r
		em.co2 = &v
	}

	return em
}

// energyReading is a pair of readings a session can be booked from at one instant.
type energyReading struct {
	at    time.Time
	meter float64 // charge meter counter in kWh, zero without one
	kWh   float64 // evcc own session accumulator in kWh
}
