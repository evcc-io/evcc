package core

import (
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core/wrapper"
)

// LoadPointAPI is the external loadpoint API
type LoadPointAPI interface {
	Name() string
	HasChargeMeter() bool

	// settings
	GetMode() api.ChargeMode
	SetMode(api.ChargeMode)
	GetTargetSoC() int
	SetTargetSoC(int) error
	GetMinSoC() int
	SetMinSoC(int) error
	SetTargetCharge(time.Time, int)
	RemoteControl(string, RemoteDemand)

	// energy
	GetMinCurrent() int64
	GetMaxCurrent() int64
	GetMinPower() int64
	GetMaxPower() int64
}

// GetMode returns loadpoint charge mode
func (lp *LoadPoint) GetMode() api.ChargeMode {
	lp.Lock()
	defer lp.Unlock()
	return lp.Mode
}

// SetMode sets loadpoint charge mode
func (lp *LoadPoint) SetMode(mode api.ChargeMode) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Printf("set charge mode: %s", string(mode))

	// apply immediately
	if lp.Mode != mode {
		lp.Mode = mode
		lp.publish("mode", mode)
		lp.requestUpdate()
	}
}

// GetTargetSoC returns loadpoint charge target soc
func (lp *LoadPoint) GetTargetSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Target
}

// SetTargetSoC sets loadpoint charge target soc
func (lp *LoadPoint) SetTargetSoC(soc int) error {
	if lp.vehicle == nil {
		return api.ErrNotAvailable
	}

	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Println("set target soc:", soc)

	// apply immediately
	if lp.SoC.Target != soc {
		lp.SoC.Target = soc
		lp.publish("targetSoC", soc)
		lp.requestUpdate()
	}

	return nil
}

// GetMinSoC returns loadpoint charge minimum soc
func (lp *LoadPoint) GetMinSoC() int {
	lp.Lock()
	defer lp.Unlock()
	return lp.SoC.Min
}

// SetMinSoC sets loadpoint charge minimum soc
func (lp *LoadPoint) SetMinSoC(soc int) error {
	if lp.vehicle == nil {
		return api.ErrNotAvailable
	}

	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Println("set min soc:", soc)

	// apply immediately
	if lp.SoC.Min != soc {
		lp.SoC.Min = soc
		lp.publish("minSoC", soc)
		lp.requestUpdate()
	}

	return nil
}

// SetTargetCharge sets loadpoint charge targetSoC
func (lp *LoadPoint) SetTargetCharge(finishAt time.Time, targetSoC int) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Printf("set target charge: %d @ %v", targetSoC, finishAt)

	// apply immediately
	// TODO check reset of targetSoC
	lp.publish("targetTime", finishAt)
	lp.publish("targetSoC", targetSoC)

	lp.requestUpdate()
}

// RemoteControl sets remote status demand
func (lp *LoadPoint) RemoteControl(source string, demand RemoteDemand) {
	lp.Lock()
	defer lp.Unlock()

	lp.log.INFO.Println("remote demand:", demand)

	// apply immediately
	if lp.remoteDemand != demand {
		lp.remoteDemand = demand

		lp.publish("remoteDisabled", demand)
		lp.publish("remoteDisabledSource", source)

		lp.requestUpdate()
	}
}

// HasChargeMeter determines if a physical charge meter is attached
func (lp *LoadPoint) HasChargeMeter() bool {
	_, isWrapped := lp.chargeMeter.(*wrapper.ChargeMeter)
	return lp.chargeMeter != nil && !isWrapped
}

// GetMinCurrent returns the minimal loadpoint current
func (lp *LoadPoint) GetMinCurrent() int64 {
	return lp.MinCurrent
}

// GetMaxCurrent returns the minimal loadpoint current
func (lp *LoadPoint) GetMaxCurrent() int64 {
	return lp.MaxCurrent
}

// GetMinPower returns the minimal loadpoint power for a single phase
func (lp *LoadPoint) GetMinPower() int64 {
	return int64(Voltage) * lp.MinCurrent
}

// GetMaxPower returns the minimal loadpoint power taking active phases into account
func (lp *LoadPoint) GetMaxPower() int64 {
	return int64(Voltage) * lp.Phases * lp.MaxCurrent
}
