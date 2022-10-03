package core

import (
	"github.com/evcc-io/evcc/api"
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

func (lp *LoadPoint) startTxn() {
	// test guard
	if lp.db == nil {
		return
	}

	if lp.txn == nil {
		lp.txn = lp.db.Txn(lp.chargeMeterTotal())

		if lp.vehicle != nil {
			lp.txn.Vehicle = lp.vehicle.Title()
		}

		if c, ok := lp.charger.(api.Identifier); ok {
			if id, err := c.Identify(); err == nil {
				lp.txn.Identifier = id
			}
		}

		lp.db.Persist(lp.txn)
	}
}

func (lp *LoadPoint) stopTxn() {
	// test guard
	if lp.db == nil || lp.txn == nil {
		return
	}

	lp.txn.Stop(lp.chargedEnergy, lp.chargeMeterTotal())

	lp.db.Persist(lp.txn)
}

func (lp *LoadPoint) updateTxn() {
	// test guard
	if lp.db == nil || lp.txn == nil {
		return
	}

	var title string
	if lp.vehicle != nil {
		title = lp.vehicle.Title()
	}

	if lp.txn.Vehicle != title {
		lp.txn.Vehicle = title
		lp.db.Persist(lp.txn)
	}
}

func (lp *LoadPoint) finalizeTxn() {
	// test guard
	if lp.db == nil {
		return
	}

	lp.txn = nil
}
