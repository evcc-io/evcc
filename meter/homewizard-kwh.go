package meter

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

func init() {
	registry.Add("homewizard-kwh", NewHomeWizardKWHFromConfig)
}

// HomeWizardKWH implements the api.Meter interface for kWh meters
type HomeWizardKWH struct {
	log    *util.Logger
	device *device.KWHDevice
	usage  string // "pv" or "grid"
}

func NewHomeWizardKWHFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		Host    string
		Token   string
		Usage   string
		Timeout time.Duration
	}{
		Usage:   "pv",
		Timeout: device.DefaultTimeout,
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
		device: device.NewKWHDevice(cc.Host, cc.Token, cc.Timeout),
		usage:  cc.Usage,
	}

	// Start device connection and wait for it to succeed
	if err := m.device.StartAndWait(cc.Timeout); err != nil {
		return nil, err
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

	// Invert power for PV (production shows as negative)
	// Don't invert for grid (import = positive, export = negative)
	if m.usage == "pv" {
		return -1 * measurement.PowerW, nil
	}
	return measurement.PowerW, nil
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
