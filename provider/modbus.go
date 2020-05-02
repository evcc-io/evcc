package provider

import (
	"math"

	"github.com/andig/evcc/provider/modbus"
	"github.com/andig/evcc/util"
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
		Value          string
	}{}
	util.DecodeOther(log, other, &cc)

	// assume RTU if not set and this is a known RS485 meter model
	if cc.RTU == nil {
		b := modbus.IsRS485(cc.Model)
		cc.RTU = &b
	}

	conn := modbus.NewConnection(log, cc.URI, cc.Device, cc.Comset, cc.Baudrate, *cc.RTU)
	device, err := modbus.NewDevice(log, cc.Model, *cc.RTU)

	log = util.NewLogger("modb")
	conn.Logger(log.TRACE)

	// prepare device
	if err == nil {
		conn.Slave(cc.ID)
		err = device.Initialize(conn.ModbusClient())

		// silence KOSTAL implementation errors
		if _, partial := err.(meters.SunSpecPartiallyInitialized); partial {
			err = nil
		}
	}
	if err != nil {
		log.FATAL.Fatal(err)
	}

	measurement, err := meters.MeasurementString(cc.Value)
	if err != nil {
		log.FATAL.Fatalf("invalid measurement %s", cc.Value)
	}

	// for RS485 check if producer supports the measurement
	op := rs485.Operation{IEC61850: measurement}
	if dev, ok := device.(*rs485.RS485); ok {
		op = modbus.RS485FindDeviceOp(dev, measurement)

		if op.IEC61850 == 0 {
			log.FATAL.Fatalf("unsupported measurement value: %s", measurement)
		}
	}

	return &Modbus{
		log:     log,
		conn:    conn,
		device:  device,
		op:      op,
		slaveID: cc.ID,
	}
}

// FloatGetter executes configured modbus read operation and implements provider.FloatGetter
func (m *Modbus) FloatGetter() (float64, error) {
	m.conn.Slave(m.slaveID)

	var res meters.MeasurementResult
	var err error

	if dev, ok := m.device.(*rs485.RS485); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), m.op)
	}

	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), m.op.IEC61850)
	}

	if err == nil {
		m.log.TRACE.Printf("%+v", res)
	}

	return res.Value, err
}

// IntGetter executes configured modbus read operation and implements provider.IntGetter
func (m *Modbus) IntGetter() (int64, error) {
	res, err := m.FloatGetter()
	return int64(math.Round(res)), err
}
