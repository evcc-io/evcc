package charger

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/evcc-io/evcc/util/modbus"
)

// Policharger charger implementation for the Ulartec Policharger 8.1 series.
//
// The charger runs in normally locked mode as documented by the vendor: reg 80 is kept
// at 16 A or more while idle, reg 44 caps the current and reg 88 starts and stops
// charging. Starting through reg 80 alone takes about 205 s, stopping about 55 s.
type Policharger struct {
	conn    *modbus.Connection
	phases  int
	current uint16
	limit   uint16 // last value written to reg 44
}

const (
	polRegThreePhase = 25
	polRegLockType   = 30 // 1 = normally locked
	polRegCurrent    = 37
	polRegConnected  = 39 // 0 while locked
	polRegCharging   = 41
	polRegLimit      = 44
	polRegSetpoint   = 80
	polRegPinPlugged = 86 // valid in normally locked mode
	polRegLock       = 88 // 0 charging allowed, 1 locked

	polArmCurrent = 16 // lowest reg 80 value that lets the charger start a session
)

func init() {
	registry.AddCtx("policharger", NewPolichargerFromConfig)
}

// NewPolichargerFromConfig creates a Policharger charger from generic config
func NewPolichargerFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID:       1,
		Baudrate: 9600,
		Comset:   "8N1",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPolicharger(ctx, cc)
}

// NewPolicharger creates a Policharger charger
func NewPolicharger(ctx context.Context, settings modbus.Settings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("policharger")
	conn.Logger(log.TRACE)

	wb := &Policharger{
		conn:   conn,
		phases: 1,
	}

	if v, err := wb.get(polRegThreePhase); err != nil {
		return nil, err
	} else if v != 0 {
		wb.phases = 3
	}

	if v, err := wb.get(polRegLockType); err != nil {
		return nil, err
	} else if v != 1 {
		return nil, errors.New("lock type must be set to 1 (normally locked) in the charger menu")
	}

	if wb.limit, err = wb.get(polRegLimit); err != nil {
		return nil, err
	}

	// an armed and unlocked charger would start without evcc enabling it
	charging, err := wb.charging()
	if err != nil {
		return nil, err
	}

	if enabled, err := wb.Enabled(); err != nil {
		return nil, err
	} else if enabled && !charging {
		if err := wb.set(polRegLock, 1); err != nil {
			return nil, err
		}
	}

	return wb, nil
}

func (wb *Policharger) get(reg uint16) (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

func (wb *Policharger) set(reg, val uint16) error {
	_, err := wb.conn.WriteSingleRegister(reg, val)
	return err
}

func (wb *Policharger) charging() (bool, error) {
	v, err := wb.get(polRegCharging)
	return v != 0, err
}

// arm caps the current and keeps the setpoint high enough for the next start.
// Reg 80 is only written on change, writing it restarts the start delay.
func (wb *Policharger) arm(amps uint16) error {
	if wb.limit != amps {
		if err := wb.set(polRegLimit, amps); err != nil {
			return err
		}
		wb.limit = amps
	}

	setpoint := max(amps, polArmCurrent)

	v, err := wb.get(polRegSetpoint)
	if err != nil || v == setpoint {
		return err
	}

	return wb.set(polRegSetpoint, setpoint)
}

// Status implements the api.Charger interface
func (wb *Policharger) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(polRegConnected, 3)
	if err != nil {
		return api.StatusNone, err
	}

	if binary.BigEndian.Uint16(b[4:]) != 0 {
		return api.StatusC, nil
	}

	plugged, err := wb.get(polRegPinPlugged)
	if err != nil {
		return api.StatusNone, err
	}

	if plugged != 0 || binary.BigEndian.Uint16(b) != 0 {
		return api.StatusB, nil
	}

	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (wb *Policharger) Enabled() (bool, error) {
	v, err := wb.get(polRegLock)
	return v == 0, err
}

// Enable implements the api.Charger interface
func (wb *Policharger) Enable(enable bool) error {
	if !enable {
		if err := wb.set(polRegLock, 1); err != nil {
			return err
		}
	}

	// arm only between sessions, never while one is starting or stopping
	charging, err := wb.charging()
	if err == nil && !charging {
		err = wb.arm(max(wb.current, 6))
	}

	// stopped already, arming is repeated on the next enable
	if !enable {
		return nil
	}

	if err != nil {
		return err
	}

	return wb.set(polRegLock, 0)
}

// MaxCurrent implements the api.Charger interface
func (wb *Policharger) MaxCurrent(current int64) error {
	if current < 6 || current > 32 {
		return fmt.Errorf("invalid current %d", current)
	}

	amps := uint16(current)
	wb.current = amps

	charging, err := wb.charging()
	if err != nil {
		return err
	}

	if !charging {
		return wb.arm(amps)
	}

	if err := wb.set(polRegSetpoint, amps); err != nil {
		return err
	}

	// the cap is only raised while charging to keep its writes rare
	if amps > wb.limit {
		if err := wb.set(polRegLimit, amps); err != nil {
			return err
		}
		wb.limit = amps
	}

	return nil
}

// chargeCurrent returns the measured current, the sensor may show a residual value while idle
func (wb *Policharger) chargeCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(polRegCurrent, 5)
	if err != nil || binary.BigEndian.Uint16(b[8:]) == 0 {
		return 0, err
	}
	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.Meter = (*Policharger)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Policharger) CurrentPower() (float64, error) {
	i, err := wb.chargeCurrent()
	return grid.CurrentToPower(i, wb.phases), err
}

var _ api.PhaseCurrents = (*Policharger)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Policharger) Currents() (float64, float64, float64, error) {
	i, err := wb.chargeCurrent()
	if wb.phases == 3 {
		return i, i, i, err
	}
	return i, 0, 0, err
}
