package charger

import (
	"fmt"

	"github.com/mark-sch/evcc/api"
	"github.com/mark-sch/evcc/util"
	"github.com/mark-sch/evcc/util/modbus"
)

const (
	phxEVSerRegEnable     = 20000 // Coil
	phxEVSerRegMaxCurrent = 22000 // Holding
	phxEVSerRegStatus     = 24000 // Input
)

// PhoenixEVSer is an api.ChargeController implementation for Phoenix EV-CC-AC1-M wallboxes.
// It uses Modbus RTU to communicate with the wallbox at configurable modbus client.
type PhoenixEVSer struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-ev-ser", NewPhoenixEVSerFromConfig)
}

// NewPhoenixEVSerFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVSerFromConfig(other map[string]interface{}) (api.Charger, error) {
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

	return NewPhoenixEVSer(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewPhoenixEVSer creates a Phoenix charger
func NewPhoenixEVSer(uri, device, comset string, baudrate int, id uint8) (*PhoenixEVSer, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, true, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ev-ser")
	conn.Logger(log.TRACE)

	wb := &PhoenixEVSer{
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixEVSer) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVSerRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[0])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixEVSer) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxEVSerRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixEVSer) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0xFF00
	}

	_, err := wb.conn.WriteSingleCoil(phxEVSerRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixEVSer) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxEVSerRegMaxCurrent, uint16(current))

	return err
}
