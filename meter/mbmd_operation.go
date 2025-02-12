package meter

import (
	"fmt"
	"strings"

	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// isRS485 determines if model is a known MBMD rs485 device model
func isRS485(model string) bool {
	for k := range rs485.Producers {
		if strings.EqualFold(model, k) {
			return true
		}
	}
	return false
}

// rs485FindDeviceOp checks is RS485 device supports operation
func rs485FindDeviceOp(device *rs485.RS485, name string) (op rs485.Operation, err error) {
	measurement, err := meters.MeasurementString(name)
	if err != nil {
		return rs485.Operation{}, fmt.Errorf("invalid measurement: %s", name)
	}

	ops := device.Producer().Produce()

	for _, op := range ops {
		if op.IEC61850 == measurement {
			return op, nil
		}
	}

	return op, fmt.Errorf("unsupported measurement: %s", measurement.String())
}
