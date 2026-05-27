package charger

import (
	"context"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	phxEVSerRegEnable     = 20000 // Coil
	phxEVSerRegMaxCurrent = 22000 // Holding
	phxEVSerRegStatus     = 24000 // Input
)

// PhoenixEVSer is an api.Charger implementation for Phoenix EV-CC-AC1-M wallboxes.
// It uses Modbus RTU to communicate with the wallbox at configurable modbus client.
type PhoenixEVSer struct {
	conn *modbus.Connection
}

func init() {
	registry.AddCtx("phoenix-ev-ser", NewPhoenixEVSerFromConfig)
}

// NewPhoenixEVSerFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVSerFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixEVSer(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
}

// NewPhoenixEVSer creates a Phoenix charger
func NewPhoenixEVSer(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, id uint8) (*PhoenixEVSer, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, id)
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

// Status implements the api.Charger interface
func (wb *PhoenixEVSer) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxEVSerRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string(b[0]))
}

// Enabled implements the api.Charger interface
func (wb *PhoenixEVSer) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxEVSerRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the api.Charger interface
func (wb *PhoenixEVSer) Enable(enable bool) error {
	var u uint16
	if enable {
		u = modbus.CoilOn
	}

	_, err := wb.conn.WriteSingleCoil(phxEVSerRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PhoenixEVSer) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phxEVSerRegMaxCurrent, uint16(current))

	return err
}
