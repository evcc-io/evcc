package meter

import (
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	v2 "github.com/evcc-io/evcc/meter/homewizard-v2"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

func init() {
	registry.Add("homewizard-battery", NewHomeWizardBatteryFromConfig)
}

// HomeWizardBattery implements the api.Meter interface for battery devices
type HomeWizardBattery struct {
	log            *util.Logger
	device         *v2.BatteryDevice
	controllerName string
	controller     *v2.P1Device
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
		Timeout:      30 * time.Second,
		MaxCharge:    800, // Default 800W charge limit for HWE-BAT
		MaxDischarge: 800, // Default 800W discharge limit for HWE-BAT
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// Validate required parameters
	if cc.Host == "" || cc.Token == "" {
		return nil, fmt.Errorf("missing host or token - run 'evcc token homewizard-v2'")
	}

	if cc.Controller == "" {
		return nil, fmt.Errorf("battery requires controller parameter (P1 meter name)")
	}

	m := &HomeWizardBattery{
		log:            util.NewLogger("homewizard-battery"),
		device:         v2.NewBatteryDevice(cc.Host, cc.Token, cc.Timeout),
		controllerName: cc.Controller,
		capacity:       cc.Capacity,
		maxCharge:      cc.MaxCharge,
		maxDischarge:   cc.MaxDischarge,
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

	m.log.INFO.Printf("configured battery at %s with controller: %s", cc.Host, cc.Controller)

	return m, nil
}

// getController resolves the controller P1 meter (lazy initialization to avoid timing issues)
func (m *HomeWizardBattery) getController() (*v2.P1Device, error) {
	var err error

	m.controllerOnce.Do(func() {
		// Debug: List all available meters
		allMeters := config.Meters().Devices()
		m.log.DEBUG.Printf("looking for controller '%s', available meters: %d", m.controllerName, len(allMeters))
		for _, dev := range allMeters {
			m.log.DEBUG.Printf("  - meter: %s (type: %T)", dev.Config().Name, dev.Instance())
		}

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

	// Default HWE-BAT capacity
	const batteryCapacity = 2.47 // kWh - HWE-BAT capacity
	return batteryCapacity
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
