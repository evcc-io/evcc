package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/db"
)

func (lp *LoadPoint) chargeMeterTotal() float64 {
	m, ok := lp.chargeMeter.(api.MeterEnergy)
	if !ok {
		return 0
	}

	f, err := m.TotalEnergy()
	if err != nil {
		lp.log.ERROR.Printf("meter energy: %v", err)
		return 0
	}

	return f
}

// createSession creates a charging session. The created timestamp is empty until set by evChargeStartHandler.
// The session is not persisted yet. That will only happen when stopSession is called.
func (lp *LoadPoint) createSession() {
	// test guard
	if lp.db == nil || lp.session != nil {
		return
	}

	lp.session = lp.db.Session(lp.chargeMeterTotal())

	if lp.vehicle != nil {
		lp.session.Vehicle = lp.vehicle.Title()
	}

	if c, ok := lp.charger.(api.Identifier); ok {
		if id, err := c.Identify(); err == nil {
			lp.session.Identifier = id
		}
	}
}

// stopSession ends a charging session segment and persists the session.
func (lp *LoadPoint) stopSession() {
	// test guard
	if lp.db == nil || lp.session == nil {
		return
	}

	// abort the session if charging has never started
	if lp.session.Created.IsZero() {
		return
	}

	lp.session.Finished = time.Now()
	lp.session.MeterStop = lp.chargeMeterTotal()

	if chargedEnergy := lp.getChargedEnergy() / 1e3; chargedEnergy > lp.session.ChargedEnergy {
		lp.session.ChargedEnergy = chargedEnergy
	}

	lp.db.Persist(lp.session)
}

type sessionOption func(*db.Session)

// updateSession updates any parameter of a charging session and persists the session.
func (lp *LoadPoint) updateSession(opts ...sessionOption) {
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
func (lp *LoadPoint) clearSession() {
	// test guard
	if lp.db == nil {
		return
	}

	lp.session = nil
}
