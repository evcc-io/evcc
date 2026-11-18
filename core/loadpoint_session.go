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
	"github.com/jinzhu/now"
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
	// test guard
	if lp.db == nil || lp.session != nil {
		return
	}

	lp.session = lp.db.New(lp.chargeMeterTotal())

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
	meterStop := lp.chargeMeterTotal()
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
	s.ChargeDuration = new(lp.chargeDuration.Abs())

	lp.applyEnergyMetrics(s)
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

func (lp *Loadpoint) resetHeatingSession() {
	if !lp.heatingDevice() {
		return
	}

	if lp.session != nil && now.With(lp.clock.Now()).BeginningOfDay().After(lp.session.Created) {
		lp.stopSession()
		lp.clearSession()

		if cr, ok := lp.chargeRater.(wrapper.ChargeResetter); ok {
			cr.ResetCharge()
		}
		if ct, ok := lp.chargeTimer.(wrapper.ChargeResetter); ok {
			ct.ResetCharge()
		}
	}

	// the day is the session: start it at the day boundary, so the device's counter is booked
	// from midnight even when evcc never switches the device on
	if lp.session == nil {
		lp.createSession()
		if lp.session != nil && lp.session.Created.IsZero() {
			lp.session.Created = lp.clock.Now()
		}
	}
}
