package configure

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"gopkg.in/yaml.v3"
)

type DeviceTestResult string

const (
	DeviceTestResultValid             DeviceTestResult = "Valid"
	DeviceTestResultValidMissingMeter DeviceTestResult = "Valid_MissingMeter"
	DeviceTestResultInvalid           DeviceTestResult = "Invalid"
)

type DeviceTest struct {
	DeviceCategory DeviceCategory
	Template       templates.Template
	ConfigValues   map[string]interface{}
}

// Test returns:
// - DeviceTestResult: Valid, Valid_MissingMeter, Invalid
// - error: != nil if the device is invalid and can not be configured with the provided settings
func (d *DeviceTest) Test() (DeviceTestResult, error) {
	v, err := d.configure()
	if err != nil {
		return DeviceTestResultInvalid, err
	}

	switch DeviceCategories[d.DeviceCategory].class {
	case templates.Charger:
		return d.testCharger(v)

	case templates.Meter:
		return d.testMeter(d.DeviceCategory, v)

	case templates.Vehicle:
		return d.testVehicle(v)

	default:
		panic("invalid class for category: " + d.DeviceCategory)
	}
}

// configure creates a configured device from a template so we can test it
func (d *DeviceTest) configure() (interface{}, error) {
	b, _, err := d.Template.RenderResult(templates.TemplateRenderModeInstance, d.ConfigValues)
	if err != nil {
		return nil, err
	}

	var instance struct {
		Type  string
		Other map[string]interface{} `yaml:",inline"`
	}

	if err := yaml.Unmarshal(b, &instance); err != nil {
		return nil, err
	}

	var v interface{}

	switch DeviceCategories[d.DeviceCategory].class {
	case templates.Meter:
		v, err = meter.NewFromConfig(instance.Type, instance.Other)
	case templates.Charger:
		v, err = charger.NewFromConfig(instance.Type, instance.Other)
	case templates.Vehicle:
		v, err = vehicle.NewFromConfig(instance.Type, instance.Other)
	}

	return v, err
}

// testCharger tests a charger device
func (d *DeviceTest) testCharger(v interface{}) (DeviceTestResult, error) {
	if v, ok := v.(api.Charger); ok {
		if _, err := v.Status(); err != nil {
			return DeviceTestResultInvalid, err
		}
	} else {
		return DeviceTestResultInvalid, errors.New("selected device is not a wallbox")
	}

	if v, ok := v.(api.Meter); ok {
		if _, err := v.CurrentPower(); err != nil {
			return DeviceTestResultValidMissingMeter, nil
		}
	} else {
		return DeviceTestResultValidMissingMeter, nil
	}

	return DeviceTestResultValid, nil
}

// testMeter tests a meter device
func (d *DeviceTest) testMeter(deviceCategory DeviceCategory, v interface{}) (DeviceTestResult, error) {
	if v, ok := v.(api.Meter); ok {
		power, err := v.CurrentPower()
		if err != nil {
			return DeviceTestResultInvalid, err
		}
		// check if the grid meter reports power 0, which should be impossible
		// happens with Kostal Piko charger that do not have a grid meter attached
		// but we can't determine this
		if power == 0 && deviceCategory == DeviceCategoryGridMeter {
			return DeviceTestResultInvalid, errors.New("grid meter reports power 0")
		}

		if deviceCategory == DeviceCategoryBatteryMeter {
			if v, ok := v.(api.Battery); ok {
				_, err := v.SoC()

				for err != nil && errors.Is(err, api.ErrMustRetry) {
					time.Sleep(3 * time.Second)
					_, err = v.SoC()
				}

				if err != nil {
					return DeviceTestResultInvalid, err
				}
			} else {
				return DeviceTestResultInvalid, errors.New("selected device is not a battery meter")
			}
		}
	} else {
		return DeviceTestResultInvalid, errors.New("selected device is not a meter")
	}

	return DeviceTestResultValid, nil
}

// testVehicle tests a vehicle device
func (d *DeviceTest) testVehicle(v interface{}) (DeviceTestResult, error) {
	if _, ok := v.(api.Vehicle); ok {
		if v, ok := v.(api.Battery); ok {
			if _, err := v.SoC(); err != nil {
				return DeviceTestResultInvalid, err
			}
		}
	} else {
		return DeviceTestResultInvalid, errors.New("selected device is not a vehicle")
	}

	return DeviceTestResultValid, nil
}
