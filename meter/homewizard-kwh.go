package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	v2 "github.com/evcc-io/evcc/meter/homewizard-v2"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("homewizard-kwh", NewHomeWizardKWHFromConfig)
}

// HomeWizardKWH implements the api.Meter interface for kWh meters
type HomeWizardKWH struct {
	log    *util.Logger
	device *v2.KWHDevice
}

func NewHomeWizardKWHFromConfig(other map[string]any) (api.Meter, error) {
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
		return nil, fmt.Errorf("missing host or token - run 'evcc token homewizard'")
	}

	m := &HomeWizardKWH{
		log:    util.NewLogger("homewizard-kwh"),
		device: v2.NewKWHDevice(cc.Host, cc.Token, cc.Timeout),
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

	m.log.INFO.Printf("configured kWh meter at %s", cc.Host)

	return m, nil
}

var _ api.Meter = (*HomeWizardKWH)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardKWH) CurrentPower() (float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	// Invert sign for PV (same logic as existing homewizard meter)
	return -1 * measurement.PowerW, nil
}

var _ api.MeterEnergy = (*HomeWizardKWH)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardKWH) TotalEnergy() (float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	return measurement.EnergyImportkWh, nil
}
