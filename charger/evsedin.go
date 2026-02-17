package charger

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
)

// EvseDIN charger implementation
type EvseDIN struct {
	conn    *modbus.Connection
	current uint16
}

const (
	evseRegCurrent  = 1000
	evseRegStatus   = 1002
	evseRegFirmware = 1005
	evseRegConfig   = 2005
)

func init() {
	registry.AddCtx("evsedin", NewEvseDINFromConfig)
}

// https://www.evracing.cz/user/documents/upload/EVSE-WB-DIN_latest.pdf

// NewEvseDINFromConfig creates an EVSE DIN charger from generic config
func NewEvseDINFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		Baudrate: 9600,
		Comset:   "8N1",
		ID:       1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEvseDIN(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
}

//go:generate go tool decorate -f decorateEvseDIN -b *EvseDIN -r api.Charger -t "api.ChargerEx,MaxCurrentMillis,func(float64) error"

// NewEvseDIN creates EVSE DIN charger
func NewEvseDIN(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	log := util.NewLogger("evse")

	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	conn.Logger(log.TRACE)
	conn.Delay(200 * time.Millisecond)

	evse := &EvseDIN{
		conn:    conn,
		current: 6, // assume min current
	}

	var maxCurrentMillis func(float64) error

	// check firmware
	bFirmware, err := evse.conn.ReadHoldingRegisters(evseRegFirmware, 1)
	if err != nil {
		return nil, err
	}

	if encoding.Uint16(bFirmware) >= 17 {
		// check configuration
		bConfig, err := evse.conn.ReadHoldingRegisters(evseRegConfig, 1)
		if err != nil {
			return nil, err
		}

		config := encoding.Uint16(bConfig)

		// enable mA feature if not enabled yet
		if config&0x0080 == 0 {
			b := make([]byte, 2)
			config |= 0x0080 // set milliAmps config bit7
			binary.BigEndian.PutUint16(b, config)

			if _, err := evse.conn.WriteMultipleRegisters(evseRegConfig, 1, b); err != nil {
				return nil, err
			}
		}

		if config&0x0080 != 0 {
			maxCurrentMillis = evse.maxCurrentMillis
			evse.current = 600 // assume min current
		}
	}

	current, err := evse.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if current > 0 {
		evse.current = current
	}

	return decorateEvseDIN(evse, maxCurrentMillis), nil
}

// Status implements the api.Charger interface
func (evse *EvseDIN) Status() (api.ChargeStatus, error) {
	b, err := evse.conn.ReadHoldingRegisters(evseRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)
	switch s {
	case 1: // not connected
		return api.StatusA, nil
	case 2: // connected, not charging
		return api.StatusB, nil
	case 3, 4: // charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (evse *EvseDIN) Enabled() (bool, error) {
	b, err := evse.conn.ReadHoldingRegisters(evseRegCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (evse *EvseDIN) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, evse.current)
	}

	_, err := evse.conn.WriteMultipleRegisters(evseRegCurrent, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (evse *EvseDIN) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	if err := evse.setCurrent(uint16(current)); err != nil {
		return err
	}
	evse.current = uint16(current)

	return nil
}

// maxCurrentMillis implements the api.ChargerEx interface
func (evse *EvseDIN) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.2f", current)
	}

	u := uint16(current * 100) // 0.01A Steps
	if err := evse.setCurrent(u); err != nil {
		return err
	}
	evse.current = u

	return nil
}

func (evse *EvseDIN) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := evse.conn.WriteMultipleRegisters(evseRegCurrent, 1, b)

	return err
}

func (evse *EvseDIN) getCurrent() (uint16, error) {
	b, err := evse.conn.ReadHoldingRegisters(evseRegCurrent, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}
