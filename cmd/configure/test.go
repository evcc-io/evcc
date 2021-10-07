package configure

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
)

func (c *CmdConfigure) testDevice(deviceCategory string, v interface{}) error {
	switch DeviceCategories[deviceCategory].class {
	case DeviceClassCharger:
		return c.testCharger(deviceCategory, v)
	case DeviceClassMeter:
		return c.testMeter(deviceCategory, v)
	case DeviceClassVehicle:
		return c.testVehicle(deviceCategory, v)
	}

	return errors.New("testDevice not implemented for this device class")
}

func (c *CmdConfigure) testCharger(deviceCategory string, v interface{}) error {
	if v, ok := v.(api.Charger); ok {
		if _, err := v.Status(); err != nil {
			return err
		}
	} else {
		return errors.New("Selected device is not a charger!")
	}

	return nil
}

func (c *CmdConfigure) testMeter(deviceCategory string, v interface{}) error {
	if v, ok := v.(api.Meter); ok {
		if _, err := v.CurrentPower(); err != nil {
			return err
		}

		if deviceCategory == DeviceCategoryBatteryMeter {
			if v, ok := v.(api.Battery); ok {
				_, err := v.SoC()

				for err != nil && errors.Is(err, api.ErrMustRetry) {
					time.Sleep(3 * time.Second)
					_, err = v.SoC()
				}

				if err != nil {
					return err
				}
			} else {
				return errors.New("Selected device is not a battery meter!")
			}
		}
	} else {
		return errors.New("Selected device is not a meter!")
	}

	return nil
}

func (c *CmdConfigure) testVehicle(deviceCategory string, v interface{}) error {
	if _, ok := v.(api.Vehicle); !ok {
		return errors.New("Selected device is not a charger!")
	}

	return nil
}
