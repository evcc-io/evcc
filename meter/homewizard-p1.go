package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	v2 "github.com/evcc-io/evcc/meter/homewizard-v2"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("homewizard-p1", NewHomeWizardP1FromConfig)
}

// HomeWizardP1 implements the api.Meter interface for P1 meters
type HomeWizardP1 struct {
	log    *util.Logger
	device *v2.P1Device
}

func NewHomeWizardP1FromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		Host    string
		Token   string
		Timeout time.Duration
	}{
		Timeout: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Validate required parameters
	if cc.Host == "" || cc.Token == "" {
		return nil, fmt.Errorf("missing host or token - run 'evcc token homewizard-v2'")
	}

	m := &HomeWizardP1{
		log:    util.NewLogger("homewizard-p1"),
		device: v2.NewP1Device(cc.Host, cc.Token, cc.Timeout),
	}

	// Start device connection
	errC := make(chan error, 1)
	m.device.Start(errC)

	// Wait for connection or timeout
	select {
	case err := <-errC:
		if err != nil {
			m.device.Stop()
			return nil, fmt.Errorf("connecting to device: %w", err)
		}
	case <-time.After(cc.Timeout):
		m.device.Stop()
		return nil, fmt.Errorf("connection timeout")
	}

	m.log.INFO.Printf("configured P1 meter at %s", cc.Host)

	return m, nil
}

var _ api.Meter = (*HomeWizardP1)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardP1) CurrentPower() (float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	return measurement.PowerW, nil
}

var _ api.MeterEnergy = (*HomeWizardP1)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardP1) TotalEnergy() (float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	return measurement.EnergyImportT1kWh + measurement.EnergyImportT2kWh, nil
}

var _ api.PhaseCurrents = (*HomeWizardP1)(nil)

// Currents implements the api.PhaseCurrents interface
func (m *HomeWizardP1) Currents() (float64, float64, float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, 0, 0, err
	}
	return measurement.CurrentL1A, measurement.CurrentL2A, measurement.CurrentL3A, nil
}

var _ api.PhaseVoltages = (*HomeWizardP1)(nil)

// Voltages implements the api.PhaseVoltages interface
func (m *HomeWizardP1) Voltages() (float64, float64, float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, 0, 0, err
	}
	return measurement.VoltageL1V, measurement.VoltageL2V, measurement.VoltageL3V, nil
}
