package core

import (
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

func (lp *LoadPoint) startSession() {
	// test guard
	if lp.db == nil {
		return
	}

	if lp.session == nil {
		lp.session = lp.db.Session(lp.chargeMeterTotal())

		if lp.vehicle != nil {
			lp.session.Vehicle = lp.vehicle.Title()
		}

		if c, ok := lp.charger.(api.Identifier); ok {
			if id, err := c.Identify(); err == nil {
				lp.session.Identifier = id
			}
		}

		// TODO remove
		lp.log.DEBUG.Println("session started")

		lp.db.Persist(lp.session)
	}
}

func (lp *LoadPoint) stopSession() {
	// test guard
	if lp.db == nil || lp.session == nil {
		return
	}

	lp.session.Stop(lp.getChargedEnergy(), lp.chargeMeterTotal())

	// TODO remove
	lp.log.DEBUG.Println("session stopped")

	lp.db.Persist(lp.session)
}

type sessionOption func(*db.Session)

func (lp *LoadPoint) updateSession(opts ...sessionOption) {
	// test guard
	if lp.db == nil || lp.session == nil {
		return
	}

	for _, opt := range opts {
		opt(lp.session)
	}

	// TODO remove
	lp.log.DEBUG.Println("session updated")

	lp.db.Persist(lp.session)
}

func (lp *LoadPoint) finalizeSession() {
	// test guard
	if lp.db == nil {
		return
	}

	lp.session = nil
}
