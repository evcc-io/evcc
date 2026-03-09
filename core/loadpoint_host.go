package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/chargercontroller"
)

// Verify Loadpoint implements chargercontroller.Host
var _ chargercontroller.Host = (*Loadpoint)(nil)

// Note: GetVehicle() is already defined in loadpoint_api.go

// WakeUpVehicle implements chargercontroller.Host
func (lp *Loadpoint) WakeUpVehicle() error {
	v := lp.GetVehicle()
	if v == nil {
		return nil
	}
	if r, ok := v.(api.Resurrector); ok {
		return r.WakeUp()
	}
	return nil
}

// Charging implements chargercontroller.Host
func (lp *Loadpoint) Charging() bool {
	return lp.charging()
}

// StartWakeUpTimer implements chargercontroller.Host
func (lp *Loadpoint) StartWakeUpTimer() {
	lp.startWakeUpTimer()
}

// StopWakeUpTimer implements chargercontroller.Host
func (lp *Loadpoint) StopWakeUpTimer() {
	lp.stopWakeUpTimer()
}
