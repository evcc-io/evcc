package provider

import (
	"fmt"
	"math"

	"github.com/andig/evcc/provider/modbus"
	"github.com/andig/evcc/util"
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
	op      rs485.Operation
	scale   float64
}

// ModbusSettings combine physical modbus configuration and MBMD model
type ModbusSettings struct {
	Model             string
	modbus.Connection `mapstructure:",squash"`
}

// NewModbusFromConfig creates Modbus plugin
func NewModbusFromConfig(log *util.Logger, typ string, other map[string]interface{}) *Modbus {
	cc := struct {
		ModbusSettings `mapstructure:",squash"`
		Register       modbus.Register
		Value          string
		Scale          float64
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
	var op rs485.Operation

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

	// model + value configured
	if cc.Value != "" {
		measurement, err := meters.MeasurementString(cc.Value)
		if err != nil {
			log.FATAL.Fatalf("invalid measurement %s", cc.Value)
		}

		// for RS485 check if producer supports the measurement
		op.IEC61850 = measurement
		if dev, ok := device.(*rs485.RS485); ok {
			op = modbus.RS485FindDeviceOp(dev, measurement)

			if op.IEC61850 == 0 {
				log.FATAL.Fatalf("unsupported measurement value: %s", measurement)
			}
		}
	}

	// register configured
	if cc.Register.Decode != "" {
		if op, err = modbus.RegisterOperation(cc.Register); err != nil {
			log.TRACE.Fatal(err)
		}
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

// FloatGetter executes configured modbus read operation and implements provider.FloatGetter
func (m *Modbus) FloatGetter() (float64, error) {
	m.conn.Slave(m.slaveID)

	var res meters.MeasurementResult
	var err error

	// if funccode is configured, execute the read directly
	if m.op.FuncCode != 0 {
		client := m.conn.ModbusClient()

		var bytes []byte
		switch m.op.FuncCode {
		case rs485.ReadHoldingReg:
			bytes, err = client.ReadHoldingRegisters(m.op.OpCode, m.op.ReadLen)
		case rs485.ReadInputReg:
			bytes, err = client.ReadInputRegisters(m.op.OpCode, m.op.ReadLen)
		default:
			return 0, fmt.Errorf("unknown function code %d", m.op.FuncCode)
		}

		if err != nil {
			return 0, errors.Wrap(err, "read failed")
		}

		return m.scale * m.op.Transform(bytes), nil
	}

	// if funccode is not configured, try find the reading on sunspec
	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), m.op.IEC61850)
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
