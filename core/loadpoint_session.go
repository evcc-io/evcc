package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/session"
)

func (lp *Loadpoint) chargeMeterTotal() float64 {
	m, ok := lp.chargeMeter.(api.MeterEnergy)
	if !ok {
		return 0
	}

	f, err := m.TotalEnergy()
	if err != nil {
		lp.log.ERROR.Printf("charge total import: %v", err)
		return 0
	}

	lp.log.DEBUG.Printf("charge total import: %.3fkWh", f)

	return f
}

// createSession creates a charging session. The created timestamp is empty until set by evChargeStartHandler.
// The session is not persisted yet. That will only happen when stopSession is called.
func (lp *Loadpoint) createSession() {
	// test guard
	if lp.db == nil || lp.session != nil {
		return
	}

	lp.session = lp.db.New(lp.chargeMeterTotal())

	if vehicle := lp.GetVehicle(); vehicle != nil {
		lp.session.Vehicle = vehicle.Title()
	}

	if c, ok := lp.charger.(api.Identifier); ok {
		if id, err := c.Identify(); err == nil {
			lp.session.Identifier = id
		}
	}
}

// stopSession ends a charging session segment and persists the session.
func (lp *Loadpoint) stopSession() {
	s := lp.session

	// test guard
	if lp.db == nil || s == nil {
		return
	}

	// abort the session if charging has never started
	if s.Created.IsZero() {
		return
	}

	s.Finished = lp.clock.Now()
	if meterStop := lp.chargeMeterTotal(); meterStop > 0 {
		s.MeterStop = &meterStop
	}

	if chargedEnergy := lp.getChargedEnergy() / 1e3; chargedEnergy > s.ChargedEnergy {
		{ // TODO remove
			lp.log.DEBUG.Printf("!! session: chargedEnergy=%.1f > chargedEnergy=%.1f", chargedEnergy, s.ChargedEnergy)
		}
		lp.sessionEnergy.Update(chargedEnergy)
	}

	solarPerc := lp.sessionEnergy.SolarPercentage()
	s.SolarPercentage = &solarPerc
	s.Price = lp.sessionEnergy.Price()
	s.PricePerKWh = lp.sessionEnergy.PricePerKWh()
	s.Co2PerKWh = lp.sessionEnergy.Co2PerKWh()
	s.ChargedEnergy = lp.sessionEnergy.TotalWh() / 1e3
	s.ChargeDuration = &lp.chargeDuration

	{ // TODO remove
		var meterStart, meterStop float64
		if s.MeterStart != nil {
			meterStart = *s.MeterStart
		}
		if s.MeterStop != nil {
			meterStop = *s.MeterStop
		}
		lp.log.DEBUG.Printf("!! session: start=%.3f stop=%.3f chargedEnergy=%.3f", meterStart, meterStop, s.ChargedEnergy)
	}

	lp.db.Persist(s)
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
