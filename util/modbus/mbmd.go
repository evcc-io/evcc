package modbus

import (
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// Operation is a register-based or sunspec modbus operation
type Operation struct {
	MBMD    rs485.Operation
	SunSpec SunSpecOperation
}

// ParseOperation parses an MBMD measurement or SunsSpec point definition into a modbus operation
func ParseOperation(dev meters.Device, measurement string) (Operation, error) {
	var (
		op  Operation
		err error
	)

	// if measurement cannot be parsed it could be SunSpec model/block/point
	op.MBMD.IEC61850, err = meters.MeasurementString(measurement)
	if err != nil {
		if op.SunSpec, err = ParsePoint(measurement); err != nil {
			return op, err
		}
	}

	// for RS485 check if producer supports the measurement
	if dev, ok := dev.(*rs485.RS485); ok {
		op.MBMD, err = RS485FindDeviceOp(dev, op.MBMD.IEC61850)
	}

	return op, err
}
