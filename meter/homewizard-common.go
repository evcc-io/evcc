package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// meterDevice interface abstracts P1MeterDevice and KWHMeterDevice
type meterDevice interface {
	GetPower(invertForPV bool) (float64, error)
	GetPhasePowers(phases int, invertForPV bool) (float64, float64, float64, error)
	GetPhaseCurrents(phases int) (float64, float64, float64, error)
	GetPhaseVoltages(phases int) (float64, float64, float64, error)
	GetTotalEnergy(usePVExport bool) (float64, error)
}

// HomeWizardMeter provides common functionality for all HomeWizard meters
type HomeWizardMeter struct {
	log    *util.Logger
	device meterDevice
	usage  string // "pv", "grid", etc.
	phases int    // 1 or 3
}

var _ api.Meter = (*HomeWizardMeter)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardMeter) CurrentPower() (float64, error) {
	return m.device.GetPower(m.usage == "pv")
}

var _ api.MeterEnergy = (*HomeWizardMeter)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardMeter) TotalEnergy() (float64, error) {
	// For PV meters, return export energy (production)
	// For grid meters, return import energy (consumption)
	return m.device.GetTotalEnergy(m.usage == "pv")
}

var _ api.PhaseCurrents = (*HomeWizardMeter)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *HomeWizardMeter) Currents() (float64, float64, float64, error) {
	return m.device.GetPhaseCurrents(m.phases)
}

var _ api.PhaseVoltages = (*HomeWizardMeter)(nil)

// Voltages implements the api.PhaseVoltages interface
func (m *HomeWizardMeter) Voltages() (float64, float64, float64, error) {
	return m.device.GetPhaseVoltages(m.phases)
}

var _ api.PhasePowers = (*HomeWizardMeter)(nil)

// Powers implements the api.PhasePowers interface
func (m *HomeWizardMeter) Powers() (float64, float64, float64, error) {
	return m.device.GetPhasePowers(m.phases, m.usage == "pv")
}

var _ api.PhaseGetter = (*HomeWizardMeter)(nil)

// GetPhases implements the api.PhaseGetter interface
func (m *HomeWizardMeter) GetPhases() (int, error) {
	return m.phases, nil
}
