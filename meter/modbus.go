package meter

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// NewModbusFromConfig creates api.Meter from config
func NewModbusFromConfig(log *util.Logger, other map[string]interface{}) api.Meter {
	cc := struct {
		provider.ModbusSettings `mapstructure:",squash"`
	}{}
	util.DecodeOther(log, other, &cc)

	conn, device, err := provider.NewDeviceConnection(log, cc.ModbusSettings)

	log = util.NewLogger("modb")
	conn.Logger(log.TRACE)

	// prepare device
	if err == nil {
		conn.Slave(cc.ID)
		err = device.Initialize(conn.ModbusClient())

		// silence Kostal implementation errors
		if _, partial := err.(meters.SunSpecPartiallyInitialized); partial {
			err = nil
		}
	}
	if err != nil {
		log.FATAL.Fatal(err)
	}

	return &Modbus{
		log:     log,
		conn:    conn,
		device:  device,
		slaveID: cc.ID,
	}
}

// Modbus is an api.Meter implementation with configurable getters and setters.
type Modbus struct {
	log     *util.Logger
	conn    meters.Connection
	device  meters.Device
	slaveID uint8
	op      rs485.Operation
}

// floatGetter executes configured modbus read operation and implements provider.FloatGetter
func (m *Modbus) floatGetter(measurement meters.Measurement) (float64, error) {
	m.conn.Slave(m.slaveID)

	var res meters.MeasurementResult
	var err error

	if dev, ok := m.device.(*rs485.RS485); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), m.op)
	}

	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), measurement)
	}

	if err == nil {
		m.log.TRACE.Printf("%+v", res)
	}

	return res.Value, err
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Modbus) CurrentPower() (float64, error) {
	return m.floatGetter(meters.Power)
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *Modbus) TotalEnergy() (float64, error) {
	return m.floatGetter(meters.Sum)
}
