package meter

import (
	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/util"
	"github.com/volkszaehler/mbmd/meters"
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
}

// CurrentPower implements the Meter.CurrentPower interface
func (m *Modbus) CurrentPower() (float64, error) {
	return 0, nil
}

// TotalEnergy implements the Meter.TotalEnergy interface
func (m *Modbus) TotalEnergy() (float64, error) {
	return 0, nil
}
