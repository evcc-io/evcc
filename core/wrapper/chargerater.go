package wrapper

import (
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// ChargeRater is responsible for providing charged energy amount
// by implementing api.ChargeRater. It uses the charge meter's ImportEnergy or
// keeps track of consumed energy by regularly updating consumed power.
type ChargeRater struct {
	sync.Mutex
	log           *util.Logger
	clck          clock.Clock
	meter         api.Meter
	charging      bool
	start         time.Time
	startEnergy   *float64 // nil until baseline successfully read from meter
	chargedEnergy float64
}

// ChargeResetter resets the charging session
type ChargeResetter interface {
	ResetCharge()
}

// NewChargeRater creates charge rater and initializes realtime clock
func NewChargeRater(log *util.Logger, meter api.Meter) *ChargeRater {
	return &ChargeRater{
		log:   log,
		clck:  clock.New(),
		meter: meter,
	}
}

// StartCharge records meter start energy. If meter does not supply ImportEnergy,
// start time is recorded and  charged energy set to zero.
func (cr *ChargeRater) StartCharge(continued bool) {
	cr.Lock()
	defer cr.Unlock()

	// time is needed if MeterImport is not supported
	cr.start = cr.clck.Now()
	cr.startEnergy = nil

	// get end energy amount
	if m, ok := api.Cap[api.MeterImport](cr.meter); ok {
		if f, err := m.ImportEnergy(); err == nil {
			cr.startEnergy = &f
			cr.log.DEBUG.Printf("charge start energy: %.3fkWh", f)
		} else if !loadpoint.AcceptableError(err) {
			cr.log.ERROR.Printf("charge total import: %v", err)
		}
	}

	if continued {
		cr.charging = true
	} else {
		cr.chargedEnergy = 0
	}
}

// StopCharge records meter stop energy. If meter does not supply ImportEnergy,
// stop time is recorded and accumulating energy though SetChargePower stopped.
func (cr *ChargeRater) StopCharge() {
	cr.Lock()
	defer cr.Unlock()

	cr.charging = false

	// get end energy amount
	if m, ok := api.Cap[api.MeterImport](cr.meter); ok {
		if f, err := m.ImportEnergy(); err == nil {
			if cr.startEnergy != nil {
				cr.chargedEnergy += f - *cr.startEnergy
			}
			cr.log.DEBUG.Printf("charge final energy: %.3fkWh", cr.chargedEnergy)
		} else if !loadpoint.AcceptableError(err) {
			cr.log.ERROR.Printf("charge total import: %v", err)
		}
	}
}

var _ ChargeResetter = (*ChargeRater)(nil)

// ChargeResetter resets the charging session
func (cr *ChargeRater) ResetCharge() {
	cr.Lock()
	defer cr.Unlock()

	// get end energy amount
	if m, ok := api.Cap[api.MeterImport](cr.meter); ok {
		if f, err := m.ImportEnergy(); err == nil {
			if cr.startEnergy != nil {
				cr.chargedEnergy += f - *cr.startEnergy
				cr.log.DEBUG.Printf("charge final energy: %.3fkWh", cr.chargedEnergy)
			}

			cr.startEnergy = &f
		} else if !loadpoint.AcceptableError(err) {
			cr.log.ERROR.Printf("charge total import: %v", err)
		}
	}

	cr.chargedEnergy = 0
	cr.start = cr.clck.Now()
}

// SetChargePower increments consumed energy by amount in kWh since last update
func (cr *ChargeRater) SetChargePower(power float64) {
	cr.Lock()
	defer cr.Unlock()

	if !cr.charging {
		return
	}

	// update energy amount if not provided by meter
	if !api.HasCap[api.MeterImport](cr.meter) {
		// convert power to energy in kWh
		cr.chargedEnergy += power / 1e3 * float64(cr.clck.Since(cr.start)) / float64(time.Hour)
		// move timestamp
		cr.start = cr.clck.Now()
	}
}

// ChargedEnergy implements the ChargeRater interface.
// It returns energy consumption since charge start in kWh.
func (cr *ChargeRater) ChargedEnergy() (float64, error) {
	cr.Lock()
	defer cr.Unlock()

	// return previously charged energy
	if !cr.charging {
		return cr.chargedEnergy, nil
	}

	// get current energy amount
	if m, ok := api.Cap[api.MeterImport](cr.meter); ok {
		f, err := m.ImportEnergy()
		if err != nil {
			return 0, fmt.Errorf("charge total import: %v", err)
		}

		// late-latch baseline if StartCharge could not read ImportEnergy
		// (e.g. OCPP transaction recovery before first MeterValues frame)
		if cr.startEnergy == nil {
			cr.startEnergy = &f
			cr.log.DEBUG.Printf("charge start energy: %.3fkWh", f)
		}

		return cr.chargedEnergy + f - *cr.startEnergy, nil
	}

	// return charged energy sofar if meter is not used
	return cr.chargedEnergy, nil
}
