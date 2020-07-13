package provider

import (
	"fmt"
	"math"

	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
	"github.com/pkg/errors"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// Modbus implements modbus RTU and TCP access
type Modbus struct {
	log     *util.Logger
	conn    meters.Connection
	device  meters.Device
	slaveID uint8
	op      modbus.Operation
	scale   float64
}

// NewModbusFromConfig creates Modbus plugin
func NewModbusFromConfig(log *util.Logger, other map[string]interface{}) *Modbus {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Register        modbus.Register
		Value           string
		Scale           float64
	}{}
	util.DecodeOther(log, other, &cc)

	// assume RTU if not set and this is a known RS485 meter model
	if cc.RTU == nil {
		b := modbus.IsRS485(cc.Model)
		cc.RTU = &b
	}

	log = util.NewLogger("modb")
	conn := modbus.NewConnection(log, cc.URI, cc.Device, cc.Comset, cc.Baudrate, *cc.RTU)
	conn.Logger(log.TRACE)

	var err error
	var device meters.Device
	var op modbus.Operation

	if cc.Value != "" && cc.Register.Decode != "" {
		log.FATAL.Fatalf("config: modbus cannot have value and register both")
	}

	// model + value configured
	if cc.Value != "" {
		if err := modbus.ParseOperation(device, cc.Value, &op); err != nil {
			log.FATAL.Fatalf("config: invalid value %s", cc.Value)
		}

		// if sunspec reading configured make sure model is defined or device won't be initalized
		if op.SunSpec.Point != "" && cc.Model == "" {
			cc.Model = "SunSpec"
		}
	}

	// register configured
	if cc.Register.Decode != "" {
		if op.MBMD, err = modbus.RegisterOperation(cc.Register); err != nil {
			log.TRACE.Fatal(err)
		}
	}

	// model configured
	if cc.Model != "" {
		device, err = modbus.NewDevice(log, cc.Model, *cc.RTU)

		// prepare device
		if err == nil {
			conn.Slave(cc.ID)
			err = device.Initialize(conn.ModbusClient())

			// silence KOSTAL implementation errors
			if _, partial := err.(meters.SunSpecPartiallyInitialized); partial {
				err = nil
			}
		}
	}
	if err != nil {
		log.FATAL.Fatal(err)
	}

	if cc.Scale == 0 {
		cc.Scale = 1
	}

	return &Modbus{
		log:     log,
		conn:    conn,
		device:  device,
		op:      op,
		scale:   cc.Scale,
		slaveID: cc.ID,
	}
}

// FloatGetter executes configured modbus read operation and implements func() (float64, error)
func (m *Modbus) FloatGetter() (float64, error) {
	m.conn.Slave(m.slaveID)

	var res meters.MeasurementResult
	var err error

	// if funccode is configured, execute the read directly
	if op := m.op.MBMD; op.FuncCode != 0 {
		client := m.conn.ModbusClient()

		var bytes []byte
		switch op.FuncCode {
		case rs485.ReadHoldingReg:
			bytes, err = client.ReadHoldingRegisters(op.OpCode, op.ReadLen)
		case rs485.ReadInputReg:
			bytes, err = client.ReadInputRegisters(op.OpCode, op.ReadLen)
		default:
			return 0, fmt.Errorf("unknown function code %d", op.FuncCode)
		}

		if err != nil {
			return 0, errors.Wrap(err, "read failed")
		}

		return m.scale * op.Transform(bytes), nil
	}

	// if funccode is not configured, try find the reading on sunspec
	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		if m.op.MBMD.IEC61850 != 0 {
			res, err = dev.QueryOp(m.conn.ModbusClient(), m.op.MBMD.IEC61850)
		} else {
			res, err = dev.QueryPoint(
				m.conn.ModbusClient(),
				m.op.SunSpec.Model,
				m.op.SunSpec.Block,
				m.op.SunSpec.Point,
			)
		}
	}

	if err == nil {
		m.log.TRACE.Printf("%+v", res)
	}

	return m.scale * res.Value, err
}

// IntGetter executes configured modbus read operation and implements provider.IntGetter
func (m *Modbus) IntGetter() (int64, error) {
	res, err := m.FloatGetter()
	return int64(math.Round(res)), err
}
