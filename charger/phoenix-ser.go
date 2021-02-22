package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	phxSERRegEnable     = 20000 // Coil
	phxSERRegMaxCurrent = 22000 // Holding
	phxSERRegStatus     = 24000 // Input
)

// PhoenixSer is an api.ChargeController implementation for Phoenix Contact EV Charge Control controllers.
// It uses serial Modbus RTU (RS485) to communicate with the controller at a configurable modbus client id.
type PhoenixSer struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-ser", NewPhoenixSerFromConfig)
}

// NewPhoenixSerFromConfig creates a Phoenix charger from generic config
func NewPhoenixSerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixSer(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewPhoenixSer creates a Phoenix charger
func NewPhoenixSer(uri, device, comset string, baudrate int, id uint8) (*PhoenixSer, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, true, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("phoenix-ser")
	conn.Logger(log.TRACE)

	wb := &PhoenixSer{
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixSer) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxSERRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[0])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixSer) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxSERRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixSer) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(phxSERRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixSer) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxSERRegMaxCurrent, uint16(current))

	return err
}
