package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
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
		lp.session.Vehicle = vehicle.GetTitle()
	} else if lp.chargerHasFeature(api.IntegratedDevice) {
		lp.session.Vehicle = lp.GetTitle()
	}

	if c, ok := lp.charger.(api.Identifier); ok {
		if id, err := c.Identify(); err == nil {
			lp.session.Identifier = id
		}
	}

	// energy
	lp.energyMetrics.Reset()
	lp.energyMetrics.Publish("session", lp)
	lp.publish(keys.ChargedEnergy, lp.GetChargedEnergy())
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

	s.SolarPercentage = lo.ToPtr(lp.energyMetrics.SolarPercentage())
	s.Price = lp.energyMetrics.Price()
	s.PricePerKWh = lp.energyMetrics.PricePerKWh()
	s.Co2PerKWh = lp.energyMetrics.Co2PerKWh()
	s.ChargedEnergy = lp.energyMetrics.TotalWh() / 1e3
	s.ChargeDuration = lo.ToPtr(lp.chargeDuration.Abs())

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

func (lp *Loadpoint) resetHeatingSession() {
	if lp.session == nil || !lp.chargerHasFeature(api.Heating) || !lp.chargerHasFeature(api.IntegratedDevice) {
		return
	}

	if !now.With(lp.clock.Now()).BeginningOfDay().After(lp.session.Created) {
		return
	}

	lp.stopSession()
	lp.clearSession()

	if cr, ok := lp.chargeRater.(wrapper.ChargeResetter); ok {
		cr.ResetCharge()
	}
	if ct, ok := lp.chargeRater.(wrapper.ChargeResetter); ok {
		ct.ResetCharge()
	}

	lp.createSession()
}
