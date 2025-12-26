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

// operationWithInversion holds an operation and whether its value should be inverted
type operationWithInversion struct {
	op     rs485.Operation
	invert bool
}

// rs485FindDeviceOp checks is RS485 device supports operation.
// If the name starts with '-', the value will be inverted.
func rs485FindDeviceOp(ops []rs485.Operation, name string) (operationWithInversion, error) {
	// Check for inversion prefix
	invert := false
	if strings.HasPrefix(name, "-") {
		invert = true
		name = strings.TrimPrefix(name, "-")
	}

	measurement, err := meters.MeasurementString(name)
	if err != nil {
		return operationWithInversion{}, fmt.Errorf("invalid measurement: %s", name)
	}

	for _, op := range ops {
		if op.IEC61850 == measurement {
			return operationWithInversion{op: op, invert: invert}, nil
		}
	}

	return operationWithInversion{}, fmt.Errorf("unsupported measurement: %s", measurement.String())
}
