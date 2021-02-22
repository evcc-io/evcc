package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	phSERRegEnable     = 20000 // Coil
	phSERRegMaxCurrent = 22000 // Holding
	phSERRegStatus     = 24000 // Input
)

// PhoenixSER is an api.ChargeController implementation for Phoenix Contact EV Charge Control Basic controllers.
// It uses serial Modbus RTU (RS485) to communicate with the controller at a configurable modbus client id.
type PhoenixSER struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-ser", NewPhoenixSERFromConfig)
}

// NewPhoenixSERFromConfig creates a Phoenix charger from generic config
func NewPhoenixSERFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixSER(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewPhoenixSER creates a Phoenix charger
func NewPhoenixSER(uri, device, comset string, baudrate int, id uint8) (*PhoenixSER, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, true, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("phoenix-ser")
	conn.Logger(log.TRACE)

	wb := &PhoenixSER{
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixSER) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phSERRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[0])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixSER) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phSERRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixSER) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(phSERRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixSER) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phSERRegMaxCurrent, uint16(current))

	return err
}
