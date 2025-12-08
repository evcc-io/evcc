package homewizard

import (
	"fmt"
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/mluiten/evcc-homewizard-v2/device"
)

// HomeWizardBattery implements the api.Meter interface for battery devices
type HomeWizardBattery struct {
	log            *util.Logger
	device         *device.BatteryDevice
	controller     *device.P1MeterDevice
	controllerOnce sync.Once
	capacity       float64
	maxCharge      float64 // Maximum charge power in W
	maxDischarge   float64 // Maximum discharge power in W
}

func NewHomeWizardBatteryFromConfig(common Config, other map[string]any) (api.Meter, error) {
	cc := struct {
		Capacity     float64
		MaxCharge    float64
		MaxDischarge float64
	}{
		MaxCharge:    device.DefaultMaxCharge,
		MaxDischarge: device.DefaultMaxDischarge,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	m := &HomeWizardBattery{
		log:          util.NewLogger("homewizard-battery"),
		device:       device.NewBatteryDevice(common.Host, common.Token, common.Timeout),
		capacity:     cc.Capacity,
		maxCharge:    cc.MaxCharge,
		maxDischarge: cc.MaxDischarge,
	}

	// Start device connection and wait for it to succeed
	if err := m.device.StartAndWait(common.Timeout); err != nil {
		return nil, err
	}

	m.log.INFO.Printf("configured battery at %s", common.Host)

	return m, nil
}

// getController resolves the controller P1 meter (lazy initialization to avoid timing issues)
func (m *HomeWizardBattery) getController() (*device.P1MeterDevice, error) {
	var err error

	m.controllerOnce.Do(func() {
		// Look up controller meter from registry
		dev, lookupErr := findP1Device(config.Meters().Devices())
		if lookupErr != nil {
			err = fmt.Errorf("controller meter not found: %w", lookupErr)
			return
		}

		controllerMeter := dev.Instance()

		// Controller must be a HomeWizardP1 meter
		controllerP1, ok := controllerMeter.(*HomeWizardP1)
		if !ok {
			err = fmt.Errorf("expected meter '%s' to be a homewizard-v2 P1 meter (got %T)", dev.Config().Name, controllerMeter)
			return
		}

		// Store P1MeterDevice for battery control
		m.controller = controllerP1.device
		m.log.DEBUG.Printf("resolved controller: %s", m.controller.Host())
	})

	if err != nil {
		return nil, err
	}

	if m.controller == nil {
		return nil, fmt.Errorf("controller not resolved")
	}

	return m.controller, nil
}

func findP1Device[T any](in []config.Device[T]) (config.Device[T], error) {
	for _, d := range in {
		if d.Config().Type == "homewizard-v2" {
			// Check if instance is a P1 meter by converting to interface{} first
			instance := d.Instance()
			if _, ok := any(instance).(*HomeWizardP1); ok {
				return d, nil
			}
		}
	}

	return nil, fmt.Errorf("cannot find any HomeWizard P1 devices; configure one before adding the battery")
}

var _ api.Meter = (*HomeWizardBattery)(nil)

// CurrentPower implements the api.Meter interface
func (m *HomeWizardBattery) CurrentPower() (float64, error) {
	return m.device.GetPower()
}

var _ api.MeterEnergy = (*HomeWizardBattery)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *HomeWizardBattery) TotalEnergy() (float64, error) {
	return m.device.GetTotalEnergy()
}

var _ api.Battery = (*HomeWizardBattery)(nil)

// Soc implements the api.Battery interface
func (m *HomeWizardBattery) Soc() (float64, error) {
	return m.device.GetSoc()
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

	m.log.INFO.Printf("converted to HomeWizard mode: %s", hwMode)

	// Set battery mode via controller P1 meter's wrapper method
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
