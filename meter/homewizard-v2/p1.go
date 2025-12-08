package homewizard

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

// HomeWizardP1 is a wrapper for P1 meters using the common HomeWizardMeter base
type HomeWizardP1 struct {
	*HomeWizardMeter
	device *device.P1MeterDevice // Keep reference for battery control
}

func NewHomeWizardP1FromConfig(common Config, other map[string]any) (api.Meter, error) {
	// Parse P1-specific configuration
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

	log := util.NewLogger("homewizard-p1")

	// Create P1MeterDevice (includes battery control)
	p1MeterDevice := device.NewP1MeterDevice(common.Host, common.Token, common.Timeout)

	// Start device connection and wait for it to succeed
	if err := p1MeterDevice.StartAndWait(common.Timeout); err != nil {
		return nil, err
	}

	log.INFO.Printf("configured P1 meter at %s (%d-phase)", common.Host, cc.Phases)

	m := &HomeWizardP1{
		HomeWizardMeter: &HomeWizardMeter{
			log:    log,
			phases: cc.Phases,
		},
		device: p1MeterDevice,
	}

	return m, nil
}

var _ api.Meter = (*HomeWizardP1)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardP1) CurrentPower() (float64, error) {
	return m.device.GetPower()
}

var _ api.MeterEnergy = (*HomeWizardP1)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardP1) TotalEnergy() (float64, error) {
	return m.device.GetTotalEnergy()
}

var _ api.PhaseCurrents = (*HomeWizardP1)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *HomeWizardP1) Currents() (float64, float64, float64, error) {
	return m.device.GetPhaseCurrents(m.phases)
}

var _ api.PhaseVoltages = (*HomeWizardP1)(nil)

// Voltages implements the api.PhaseVoltages interface
func (m *HomeWizardP1) Voltages() (float64, float64, float64, error) {
	return m.device.GetPhaseVoltages(m.phases)
}

var _ api.PhasePowers = (*HomeWizardP1)(nil)

// Powers implements the api.PhasePowers interface
func (m *HomeWizardP1) Powers() (float64, float64, float64, error) {
	return m.device.GetPhasePowers(m.phases)
}

// SetBatteryMode sets battery mode via P1 meter (for battery controller)
func (m *HomeWizardP1) SetBatteryMode(mode string) error {
	return m.device.SetBatteryMode(mode)
}
