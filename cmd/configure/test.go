package configure

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
)

var ErrChargerHasNoMeter = errors.New("charger has no meter")

func (c *CmdConfigure) testDevice(deviceCategory DeviceCategory, v interface{}) (bool, error) {
	switch DeviceCategories[deviceCategory].class {
	case DeviceClassCharger:
		return c.testCharger(v)
	case DeviceClassMeter:
		return c.testMeter(deviceCategory, v)
	case DeviceClassVehicle:
		return c.testVehicle(v)
	}

	return false, errors.New("testDevice not implemented for this device class")
}

func (c *CmdConfigure) testCharger(v interface{}) (bool, error) {
	if v, ok := v.(api.Charger); ok {
		if _, err := v.Status(); err != nil {
			return false, err
		}
	} else {
		return false, errors.New("Selected device is not a wallbox!")
	}

	if v, ok := v.(api.Meter); ok {
		if _, err := v.CurrentPower(); err != nil {
			return true, ErrChargerHasNoMeter
		}
	}

	return true, nil
}

func (c *CmdConfigure) testMeter(deviceCategory DeviceCategory, v interface{}) (bool, error) {
	if v, ok := v.(api.Meter); ok {
		if _, err := v.CurrentPower(); err != nil {
			return false, err
		}

		if deviceCategory == DeviceCategoryBatteryMeter {
			if v, ok := v.(api.Battery); ok {
				_, err := v.SoC()

				for err != nil && errors.Is(err, api.ErrMustRetry) {
					time.Sleep(3 * time.Second)
					_, err = v.SoC()
				}

				if err != nil {
					return false, err
				}
			} else {
				return false, errors.New("Selected device is not a battery meter!")
			}
		}
	} else {
		return false, errors.New("Selected device is not a meter!")
	}

	return true, nil
}

func (c *CmdConfigure) testVehicle(v interface{}) (bool, error) {
	if _, ok := v.(api.Vehicle); ok {
		if v, ok := v.(api.Battery); ok {
			if _, err := v.SoC(); err != nil {
				return false, err
			}
		}
	} else {
		return false, errors.New("Selected device is not a vehicle!")
	}

	return true, nil
}
