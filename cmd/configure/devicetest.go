package configure

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/vehicle"
	"gopkg.in/yaml.v3"
)

type DeviceTestResult string

const (
	DeviceTestResult_Valid              DeviceTestResult = "Valid"
	DeviceTestResult_Valid_MissingMeter DeviceTestResult = "Valid_MissingMeter"
	DeviceTestResult_Invalid            DeviceTestResult = "Invalid"
)

type DeviceTest struct {
	DeviceCategory DeviceCategory
	Template       templates.Template
	ConfigValues   map[string]interface{}
}

// returns
// DeviceTestResult: Valid, Valid_MissingMeter, Invalid
// error: != nil if the device is invalid and can not be configured with the provided settings
func (d *DeviceTest) Test() (DeviceTestResult, error) {
	v, err := d.configure()
	if err != nil {
		return DeviceTestResult_Invalid, err
	}

	switch DeviceCategories[d.DeviceCategory].class {
	case DeviceClassCharger:
		return d.testCharger(v)
	case DeviceClassMeter:
		return d.testMeter(d.DeviceCategory, v)
	case DeviceClassVehicle:
		return d.testVehicle(v)
	}

	return DeviceTestResult_Invalid, errors.New("testDevice not implemented for this device class")
}

// create a configured device from a template so we can test it
func (d *DeviceTest) configure() (interface{}, error) {
	b, _, err := d.Template.RenderResult(false, d.ConfigValues)
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
	case DeviceClassMeter:
		v, err = meter.NewFromConfig(instance.Type, instance.Other)
	case DeviceClassCharger:
		v, err = charger.NewFromConfig(instance.Type, instance.Other)
	case DeviceClassVehicle:
		v, err = vehicle.NewFromConfig(instance.Type, instance.Other)
	}

	return v, err
}

func (d *DeviceTest) testCharger(v interface{}) (DeviceTestResult, error) {
	if v, ok := v.(api.Charger); ok {
		if _, err := v.Status(); err != nil {
			return DeviceTestResult_Invalid, err
		}
	} else {
		return DeviceTestResult_Invalid, errors.New("Selected device is not a wallbox!")
	}

	if v, ok := v.(api.Meter); ok {
		if _, err := v.CurrentPower(); err != nil {
			return DeviceTestResult_Valid_MissingMeter, nil
		}
	}

	return DeviceTestResult_Valid, nil
}

func (d *DeviceTest) testMeter(deviceCategory DeviceCategory, v interface{}) (DeviceTestResult, error) {
	if v, ok := v.(api.Meter); ok {
		if _, err := v.CurrentPower(); err != nil {
			return DeviceTestResult_Invalid, err
		}

		if deviceCategory == DeviceCategoryBatteryMeter {
			if v, ok := v.(api.Battery); ok {
				_, err := v.SoC()

				for err != nil && errors.Is(err, api.ErrMustRetry) {
					time.Sleep(3 * time.Second)
					_, err = v.SoC()
				}

				if err != nil {
					return DeviceTestResult_Invalid, err
				}
			} else {
				return DeviceTestResult_Invalid, errors.New("Selected device is not a battery meter!")
			}
		}
	} else {
		return DeviceTestResult_Invalid, errors.New("Selected device is not a meter!")
	}

	return DeviceTestResult_Valid, nil
}

func (d *DeviceTest) testVehicle(v interface{}) (DeviceTestResult, error) {
	if _, ok := v.(api.Vehicle); ok {
		if v, ok := v.(api.Battery); ok {
			if _, err := v.SoC(); err != nil {
				return DeviceTestResult_Invalid, err
			}
		}
	} else {
		return DeviceTestResult_Invalid, errors.New("Selected device is not a vehicle!")
	}

	return DeviceTestResult_Valid, nil
}
