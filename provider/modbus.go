package provider

import (
	"strings"

	"github.com/andig/evcc/api"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

type Modbus struct {
	log     *api.Logger
	conn    meters.Connection
	device  meters.Device
	slaveID uint8
	op      rs485.Operation
}

var connections map[string]meters.Connection

func NewModbusFromConfig(log *api.Logger, typ string, other map[string]interface{}) *Modbus {
	cc := struct {
		URI, Device, Comset string
		Meter, Value        string
		Baudrate            int
		ID                  uint8
	}{}
	api.DecodeOther(log, other, &cc)

	if connections == nil {
		connections = make(map[string]meters.Connection)
	}

	var conn meters.Connection
	var device meters.Device
	var op rs485.Operation
	var err error

	switch strings.ToLower(typ) {
	case "modbus-rtu":
		conn = connections[cc.Device]
		if conn == nil {
			conn = meters.NewRTU(cc.Device, cc.Baudrate, cc.Comset)
			connections[cc.Device] = conn
		}
		device, err = rs485.NewDevice(strings.ToUpper(cc.Meter))
	case "modbus-rtuovertcp", "modbus-tcprtu", "modbus-rtutcp":
		conn = connections[cc.URI]
		if conn == nil {
			conn = meters.NewRTUOverTCP(cc.URI)
			connections[cc.URI] = conn
		}
		device, err = rs485.NewDevice(strings.ToUpper(cc.Meter))
	case "modbus-tcp":
		conn = connections[cc.URI]
		if conn == nil {
			conn = meters.NewTCP(cc.URI)
			connections[cc.URI] = conn
		}
		device = sunspec.NewDevice(strings.ToUpper(cc.Meter))
	default:
		log.FATAL.Fatalf("invalid provider type %s", typ)
	}

	log = api.NewLogger("modb")
	conn.Logger(log.TRACE)

	if err == nil {
		conn.Slave(cc.ID)
		err = device.Initialize(conn.ModbusClient())
	}
	if err != nil {
		log.FATAL.Fatal(err)
	}

	measurement := meters.UnStringMeasurement(cc.Value)
	if measurement == 0 {
		log.FATAL.Fatalf("invalid measurement %s", cc.Value)
	}

	// for RS485 check if producer supports the measurement
	if dev, ok := device.(*rs485.RS485); ok {
		op = rs485FindOp(dev, measurement)

		if op.IEC61850 == 0 {
			log.FATAL.Fatalf("invalid value %s", measurement)
		}
	} else {
		op.IEC61850 = measurement
	}

	log.FATAL.Printf("%+v", op)

	return &Modbus{
		log:     log,
		conn:    conn,
		device:  device,
		op:      op,
		slaveID: cc.ID,
	}
}

func rs485FindOp(device *rs485.RS485, measurement meters.Measurement) (op rs485.Operation) {
	ops := device.Producer().Produce()

	for _, o := range ops {
		if o.IEC61850 == measurement {
			op = o
			break
		}
	}

	return op
}

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

	log.TRACE.Printf("%+v", res)

	return res.Value, err
}

// IntGetter executes configured modbus read operation
func (m *Modbus) IntGetter() (int64, error) {
	res, err := m.FloatGetter()
	return int64(res), err
}
