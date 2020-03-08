package core

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
	"github.com/andig/evcc/push"
)

const (
	evcc            = "evcc"          // notification sender
	evChargeCurrent = "ChargeCurrent" // update fakeChargeMeter
	evChargePower   = "ChargePower"   // update chargeRater
	evStartCharge   = "StartCharge"   // update chargeTimer
	evStopCharge    = "StopCharge"    // update chargeTimer
)

// evChargeStartHandler sends external start event
func (lp *LoadPoint) evChargeStartHandler() {
	lp.notificationChan <- push.Event{
		EventId: push.ChargeStart,
		Sender:  evcc,
		Attributes: map[string]interface{}{
			"lp":   lp.Name,
			"mode": lp.GetMode(),
		},
	}
}

// evChargeStartHandler sends external stop event
func (lp *LoadPoint) evChargeStopHandler() {
	energy, err := lp.ChargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s %v", lp.Name, err)
	}

	lp.notificationChan <- push.Event{
		EventId: push.ChargeStop,
		Sender:  evcc,
		Attributes: map[string]interface{}{
			"lp":     lp.Name,
			"energy": energy,
		},
	}
}

// evChargeCurrentHandler updates proxy charge meter's charge current.
// If physical charge meter is present this handler is not used.
func (lp *LoadPoint) evChargeCurrentHandler(m *wrapper.ChargeMeter) func(para ...interface{}) {
	return func(para ...interface{}) {
		current := para[0].(int64)
		if current > 0 && lp.status != api.StatusC {
			current = 0
		}
		m.SetChargeCurrent(current)
	}
}

// Prepare loadpoint configuration by adding missing helper elements
func (lp *LoadPoint) Prepare(uiChan chan<- Param, notificationChan chan<- push.Event) {
	lp.notificationChan = notificationChan
	lp.uiChan = uiChan

	if lp.PVMeter == nil && lp.GridMeter == nil {
		log.FATAL.Fatal("missing either PV or Grid meter - aborting")
	}

	// ensure charge meter exists
	if lp.ChargeMeter == nil {
		if mt, ok := lp.Charger.(api.Meter); ok {
			lp.ChargeMeter = mt
		} else {
			mt := &wrapper.ChargeMeter{
				Phases:  lp.Phases,
				Voltage: lp.Voltage,
			}
			_ = lp.bus.Subscribe(evChargeCurrent, lp.evChargeCurrentHandler(mt))
			_ = lp.bus.Subscribe(evStopCharge, func() {
				mt.SetChargeCurrent(0)
			})
			lp.ChargeMeter = mt
		}
	}

	// ensure charge rater exists
	if rt, ok := lp.Charger.(api.ChargeRater); ok {
		lp.ChargeRater = rt
	} else {
		rt := wrapper.NewChargeRater(lp.Name, lp.ChargeMeter)
		_ = lp.bus.Subscribe(evChargePower, rt.SetChargePower)
		_ = lp.bus.Subscribe(evStartCharge, rt.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, rt.StopCharge)
		lp.ChargeRater = rt
	}

	// ensure charge timer exists
	if ct, ok := lp.Charger.(api.ChargeTimer); ok {
		lp.ChargeTimer = ct
	} else {
		ct := wrapper.NewChargeTimer()
		_ = lp.bus.Subscribe(evStartCharge, ct.StartCharge)
		_ = lp.bus.Subscribe(evStopCharge, ct.StopCharge)
		lp.ChargeTimer = ct
	}

	// event handlers
	_ = lp.bus.Subscribe(evStartCharge, lp.evChargeStartHandler)
	_ = lp.bus.Subscribe(evStopCharge, lp.evChargeStopHandler)
}
