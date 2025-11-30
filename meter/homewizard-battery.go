package meter

import (
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

func init() {
	registry.Add("homewizard-battery", NewHomeWizardBatteryFromConfig)
}

// HomeWizardBattery implements the api.Meter interface for battery devices
type HomeWizardBattery struct {
	log            *util.Logger
	device         *device.BatteryDevice
	controllerName string
	controller     *device.P1Device
	controllerOnce sync.Once
	capacity       float64
	maxCharge      float64 // Maximum charge power in W
	maxDischarge   float64 // Maximum discharge power in W
}

func NewHomeWizardBatteryFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		Host         string
		Token        string
		Controller   string
		Capacity     float64
		MaxCharge    float64
		MaxDischarge float64
		Timeout      time.Duration
	}{
		Timeout:      device.DefaultTimeout,
		MaxCharge:    device.DefaultMaxCharge,
		MaxDischarge: device.DefaultMaxDischarge,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Validate required parameters
	if cc.Host == "" || cc.Token == "" {
		return nil, fmt.Errorf("missing host or token - run 'evcc token homewizard'")
	}

	if cc.Controller == "" {
		return nil, fmt.Errorf("battery requires controller parameter (P1 meter name)")
	}

	m := &HomeWizardBattery{
		log:            util.NewLogger("homewizard-battery"),
		device:         device.NewBatteryDevice(cc.Host, cc.Token, cc.Timeout),
		controllerName: cc.Controller,
		capacity:       cc.Capacity,
		maxCharge:      cc.MaxCharge,
		maxDischarge:   cc.MaxDischarge,
	}

	// Start device connection and wait for it to succeed
	if err := m.device.StartAndWait(cc.Timeout); err != nil {
		return nil, err
	}

	m.log.INFO.Printf("configured battery at %s with controller: %s", cc.Host, cc.Controller)

	return m, nil
}

// getController resolves the controller P1 meter (lazy initialization to avoid timing issues)
func (m *HomeWizardBattery) getController() (*device.P1Device, error) {
	var err error

	m.controllerOnce.Do(func() {
		// Look up controller meter from registry
		dev, lookupErr := config.Meters().ByName(m.controllerName)
		if lookupErr != nil {
			err = fmt.Errorf("controller meter '%s' not found: %w", m.controllerName, lookupErr)
			return
		}

		controllerMeter := dev.Instance()

		// Controller must be a HomeWizardP1 meter
		controllerP1, ok := controllerMeter.(*HomeWizardP1)
		if !ok {
			err = fmt.Errorf("controller '%s' must be a homewizard-p1 meter (got %T)", m.controllerName, controllerMeter)
			return
		}

		m.controller = controllerP1.device
		m.log.DEBUG.Printf("resolved controller: %s (%s)", m.controllerName, m.controller.Host())
	})

	if err != nil {
		return nil, err
	}

	if m.controller == nil {
		return nil, fmt.Errorf("controller not resolved")
	}

	return m.controller, nil
}

var _ api.Meter = (*HomeWizardBattery)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardBattery) CurrentPower() (float64, error) {
	// Get power directly from battery device
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	// Invert the battery power, because HW reports negative = discharging and positive = charging
	return -1 * measurement.PowerW, nil
}

var _ api.MeterEnergy = (*HomeWizardBattery)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardBattery) TotalEnergy() (float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	return measurement.EnergyImportkWh, nil
}

var _ api.Battery = (*HomeWizardBattery)(nil)

// Soc implements the api.Battery interface
func (m *HomeWizardBattery) Soc() (float64, error) {
	measurement, err := m.device.GetMeasurement()
	if err != nil {
		return 0, err
	}
	return measurement.StateOfChargePct, nil
}

var _ api.BatteryCapacity = (*HomeWizardBattery)(nil)

// Capacity implements the api.BatteryCapacity interface
func (m *HomeWizardBattery) Capacity() float64 {
	// If user provided capacity, use that
	if m.capacity > 0 {
		return m.capacity
	}

	// Use default HWE-BAT capacity from device
	return m.device.DefaultCapacity()
}

var _ api.BatteryController = (*HomeWizardBattery)(nil)

// SetBatteryMode implements the api.BatteryController interface
func (m *HomeWizardBattery) SetBatteryMode(mode api.BatteryMode) error {
	m.log.INFO.Printf("setting battery mode to: %v", mode)

	// Get controller P1 meter (lazy resolution)
	controller, err := m.getController()
	if err != nil {
		m.log.ERROR.Printf("failed to get controller: %v", err)
		return err
	}

	// Convert evcc mode to HomeWizard mode
	var hwMode string
	switch mode {
	case api.BatteryNormal:
		hwMode = "zero"
	case api.BatteryCharge:
		hwMode = "to_full"
	case api.BatteryHold:
		hwMode = "standby"
	default:
		return fmt.Errorf("unsupported battery mode: %v", mode)
	}

	m.log.INFO.Printf("converted to HomeWizard mode: %s (controller: %s)", hwMode, m.controllerName)

	// Set battery mode via controller P1 meter
	if err := controller.SetBatteryMode(hwMode); err != nil {
		m.log.ERROR.Printf("failed to set battery mode: %v", err)
		return err
	}

	m.log.INFO.Printf("battery mode set successfully to: %s", hwMode)
	return nil
}

var _ api.BatteryPowerLimiter = (*HomeWizardBattery)(nil)

// GetPowerLimits implements the api.BatteryPowerLimiter interface
func (m *HomeWizardBattery) GetPowerLimits() (float64, float64) {
	return m.maxCharge, m.maxDischarge
}
