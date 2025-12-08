package homewizard

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

// HomeWizardKWH is a wrapper for kWh meters using the common HomeWizardMeter base
type HomeWizardKWH struct {
	*HomeWizardMeter
	device *device.KWHMeterDevice
	usage  string // "pv" or "charge"
}

func NewHomeWizardKWHFromConfig(common Config, other map[string]any) (api.Meter, error) {
	// Parse kWh-specific configuration
	cc := struct {
		Phases int // 1 or 3
	}{
		Phases: 3,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Validate phases
	if cc.Phases != 1 && cc.Phases != 3 {
		return nil, fmt.Errorf("invalid phases value %d: must be 1 or 3", cc.Phases)
	}

	log := util.NewLogger("homewizard-kwh")
	kwhDevice := device.NewKWHMeterDevice(common.Host, common.Token, common.Timeout)

	// Start device connection and wait for it to succeed
	if err := kwhDevice.StartAndWait(common.Timeout); err != nil {
		return nil, err
	}

	log.INFO.Printf("configured kWh meter at %s (%d-phase, usage=%s)", common.Host, cc.Phases, common.Usage)

	m := &HomeWizardKWH{
		HomeWizardMeter: &HomeWizardMeter{
			log:    log,
			phases: cc.Phases,
		},
		device: kwhDevice,
		usage:  common.Usage,
	}

	return m, nil
}

var _ api.Meter = (*HomeWizardKWH)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardKWH) CurrentPower() (float64, error) {
	return m.device.GetPower(m.usage == "pv")
}

var _ api.MeterEnergy = (*HomeWizardKWH)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardKWH) TotalEnergy() (float64, error) {
	return m.device.GetTotalEnergy(m.usage == "pv")
}

var _ api.PhaseCurrents = (*HomeWizardKWH)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *HomeWizardKWH) Currents() (float64, float64, float64, error) {
	return m.device.GetPhaseCurrents(m.phases)
}

var _ api.PhaseVoltages = (*HomeWizardKWH)(nil)

// Voltages implements the api.PhaseVoltages interface
func (m *HomeWizardKWH) Voltages() (float64, float64, float64, error) {
	return m.device.GetPhaseVoltages(m.phases)
}

var _ api.PhasePowers = (*HomeWizardKWH)(nil)

// Powers implements the api.PhasePowers interface
func (m *HomeWizardKWH) Powers() (float64, float64, float64, error) {
	return m.device.GetPhasePowers(m.phases, m.usage == "pv")
}
