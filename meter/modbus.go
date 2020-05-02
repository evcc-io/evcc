package meter

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/provider/modbus"
	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
	"github.com/volkszaehler/mbmd/meters/rs485"
	"github.com/volkszaehler/mbmd/meters/sunspec"
)

// Modbus is an api.Meter implementation with configurable getters and setters.
type Modbus struct {
	log      *util.Logger
	conn     meters.Connection
	device   meters.Device
	slaveID  uint8
	opPower  rs485.Operation
	opEnergy rs485.Operation
}

// NewModbusFromConfig creates api.Meter from config
func NewModbusFromConfig(log *util.Logger, other map[string]interface{}) api.Meter {
	cc := struct {
		provider.ModbusSettings `mapstructure:",squash"`
		Power, Energy           string
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

		// silence Kostal implementation errors
		if _, partial := err.(meters.SunSpecPartiallyInitialized); partial {
			err = nil
		}
	}
	if err != nil {
		log.FATAL.Fatal(err)
	}

	m := &Modbus{
		log:     log,
		conn:    conn,
		device:  device,
		slaveID: cc.ID,
	}

	// power reading
	if cc.Power == "" {
		cc.Power = "Power"
	}

	powerM, err := meters.MeasurementString(cc.Power)
	if err != nil {
		log.FATAL.Fatalf("invalid measurement for power: %s", cc.Power)
	}

	// for RS485 check if producer supports the measurement
	m.opPower = rs485.Operation{IEC61850: powerM}
	if dev, ok := device.(*rs485.RS485); ok {
		m.opPower = modbus.RS485FindDeviceOp(dev, powerM)

		if m.opPower.IEC61850 == 0 {
			log.FATAL.Fatalf("unsupported measurement for power: %s", cc.Power)
		}
	}

	// decorate energy reading
	if cc.Energy != "" {
		energyM, err := meters.MeasurementString(cc.Energy)
		if err != nil {
			log.FATAL.Fatalf("invalid measurement for energy: %s", cc.Energy)
		}

		// for RS485 check if producer supports the measurement
		m.opEnergy = rs485.Operation{IEC61850: energyM}
		if dev, ok := device.(*rs485.RS485); ok {
			m.opEnergy = modbus.RS485FindDeviceOp(dev, energyM)

			if m.opEnergy.IEC61850 == 0 {
				log.FATAL.Fatalf("unsupported measurement for energy: %s", cc.Energy)
			}
		}

		return &ModbusEnergy{m}
	}

	return m
}

// floatGetter executes configured modbus read operation and implements provider.FloatGetter
func (m *Modbus) floatGetter(op rs485.Operation) (float64, error) {
	m.conn.Slave(m.slaveID)

	var res meters.MeasurementResult
	var err error

	if dev, ok := m.device.(*rs485.RS485); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), op)
	}

	if dev, ok := m.device.(*sunspec.SunSpec); ok {
		res, err = dev.QueryOp(m.conn.ModbusClient(), op.IEC61850)
	}

	if err == nil {
		m.log.TRACE.Printf("%+v", res)
	}

	return res.Value, err
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Modbus) CurrentPower() (float64, error) {
	return m.floatGetter(m.opPower)
}

// ModbusEnergy decorates Modbus with api.MeterEnergy interface
type ModbusEnergy struct {
	*Modbus
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *ModbusEnergy) TotalEnergy() (float64, error) {
	return m.floatGetter(m.opEnergy)
}
