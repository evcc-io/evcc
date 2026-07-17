package meter

import (
	"context"
	"encoding/binary"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Marstek is a native modbus implementation for the Marstek Venus battery family.
//
// It replaces the template-based chargePower/dischargePower setter sequences with
// Go code so upstream changes to the shared Marstek meter templates can no longer
// break the watt-level fast-loop control. Reads use generation-specific registers;
// control (mode + watt-level charge/discharge) is generation-independent.

// control registers (same across generations)
const (
	marstekRegRS485     = 42000 // RS485 control mode
	marstekRegForce     = 42010 // force charge/discharge direction
	marstekRegChargeW   = 42020 // forcible charge power (W)
	marstekRegDischW    = 42021 // forcible discharge power (W)
	marstekRegWorkMode  = 43000 // user work mode
	marstekRegMaxCharge = 44002 // max charge power (W)

	marstekRS485On  = 21930
	marstekRS485Off = 21947

	marstekForceStop      = 0
	marstekForceCharge    = 1
	marstekForceDischarge = 2
)

// Marstek implements watt-level Marstek Venus control over modbus.
type Marstek struct {
	conn                        *modbus.Connection
	generation                  int
	workModeNormal              uint16
	maxChargePower              uint16
	minSoc, maxSoc              float64
	capacity                    float64
	chargeLimit, dischargeLimit float64
}

func init() {
	registry.AddCtx("marstek", NewMarstekFromConfig)
}

// NewMarstekFromConfig creates a Marstek meter from generic config
func NewMarstekFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		modbus.Settings   `mapstructure:",squash"`
		Generation        int
		WorkModeNormal    uint16
		MaxChargePower    uint16
		MaxDischargePower float64
		MinSoc, MaxSoc    float64
		Capacity          float64
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
		Generation:        3,
		WorkModeNormal:    1,
		MaxChargePower:    2500,
		MaxDischargePower: 800,
		MinSoc:            11,
		MaxSoc:            100,
		Capacity:          5.12,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	conn, err := modbus.NewConnection(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Settings.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("marstek")
	conn.Logger(log.TRACE)

	m := &Marstek{
		conn:           conn,
		generation:     cc.Generation,
		workModeNormal: cc.WorkModeNormal,
		maxChargePower: cc.MaxChargePower,
		minSoc:         cc.MinSoc,
		maxSoc:         cc.MaxSoc,
		capacity:       cc.Capacity,
		chargeLimit:    float64(cc.MaxChargePower),
		dischargeLimit: cc.MaxDischargePower,
	}

	return m, nil
}

// CurrentPower implements the api.Meter interface
func (m *Marstek) CurrentPower() (float64, error) {
	if m.generation >= 3 {
		b, err := m.conn.ReadHoldingRegisters(30006, 1)
		if err != nil {
			return 0, err
		}
		return float64(int16(binary.BigEndian.Uint16(b))), nil
	}

	b, err := m.conn.ReadHoldingRegisters(32202, 2)
	if err != nil {
		return 0, err
	}
	return float64(int32(binary.BigEndian.Uint32(b))), nil
}

var _ api.MeterEnergy = (*Marstek)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (m *Marstek) TotalEnergy() (float64, error) {
	b, err := m.conn.ReadHoldingRegisters(33002, 2)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint32(b)) * 0.01, nil
}

var _ api.Battery = (*Marstek)(nil)

// Soc implements the api.Battery interface
func (m *Marstek) Soc() (float64, error) {
	if m.generation >= 3 {
		b, err := m.conn.ReadHoldingRegisters(34002, 1)
		if err != nil {
			return 0, err
		}
		return float64(binary.BigEndian.Uint16(b)) * 0.1, nil
	}

	b, err := m.conn.ReadHoldingRegisters(32104, 1)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.BatteryCapacity = (*Marstek)(nil)

// Capacity implements the api.BatteryCapacity interface
func (m *Marstek) Capacity() float64 {
	return m.capacity
}

var _ api.BatterySocLimiter = (*Marstek)(nil)

// GetSocLimits implements the api.BatterySocLimiter interface
func (m *Marstek) GetSocLimits() (min, max float64) {
	return m.minSoc, m.maxSoc
}

var _ api.BatteryPowerLimiter = (*Marstek)(nil)

// GetPowerLimits implements the api.BatteryPowerLimiter interface
func (m *Marstek) GetPowerLimits() (charge, discharge float64) {
	return m.chargeLimit, m.dischargeLimit
}

var _ api.BatteryController = (*Marstek)(nil)

// SetBatteryMode implements the api.BatteryController interface.
//
// Hold and Charge keep RS485 control enabled so the fast loop retains ownership;
// Normal and HoldCharge hand control back to the device's work mode. HoldCharge
// blocks charging via the max-charge-power register while leaving discharge available.
func (m *Marstek) SetBatteryMode(mode api.BatteryMode) error {
	switch mode {
	case api.BatteryNormal:
		return m.write(
			reg{marstekRegMaxCharge, m.maxChargePower}, // restore charge limit
			reg{marstekRegRS485, marstekRS485On},
			reg{marstekRegWorkMode, m.workModeNormal},
			reg{marstekRegRS485, marstekRS485Off},
		)

	case api.BatteryHold:
		return m.write(
			reg{marstekRegRS485, marstekRS485On},
			reg{marstekRegForce, marstekForceStop},
		)

	case api.BatteryCharge:
		return m.write(
			reg{marstekRegRS485, marstekRS485On},
			reg{marstekRegChargeW, m.maxChargePower},
			reg{marstekRegForce, marstekForceCharge},
		)

	case api.BatteryHoldCharge:
		return m.write(
			reg{marstekRegMaxCharge, 0}, // block charging
			reg{marstekRegRS485, marstekRS485On},
			reg{marstekRegWorkMode, m.workModeNormal},
			reg{marstekRegRS485, marstekRS485Off},
		)

	default:
		return api.ErrNotAvailable
	}
}

var _ api.BatteryPowerController = (*Marstek)(nil)

// SetBatteryChargePower implements the api.BatteryPowerController interface
func (m *Marstek) SetBatteryChargePower(watts float64) error {
	if watts <= 0 {
		return m.stop()
	}

	return m.write(
		reg{marstekRegRS485, marstekRS485On},
		reg{marstekRegChargeW, uint16(watts)}, // power before direction
		reg{marstekRegForce, marstekForceCharge},
	)
}

// SetBatteryDischargePower implements the api.BatteryPowerController interface
func (m *Marstek) SetBatteryDischargePower(watts float64) error {
	if watts <= 0 {
		return m.stop()
	}

	return m.write(
		reg{marstekRegRS485, marstekRS485On},
		reg{marstekRegDischW, uint16(watts)}, // power before direction
		reg{marstekRegForce, marstekForceDischarge},
	)
}

// stop sets direction to Stop and zeroes the charge power register
func (m *Marstek) stop() error {
	return m.write(
		reg{marstekRegRS485, marstekRS485On},
		reg{marstekRegForce, marstekForceStop},
		reg{marstekRegChargeW, 0},
	)
}

// reg is a single register write
type reg struct {
	addr, val uint16
}

// write applies the register writes in order, aborting on the first error
func (m *Marstek) write(regs ...reg) error {
	for _, r := range regs {
		if _, err := m.conn.WriteSingleRegister(r.addr, r.val); err != nil {
			return err
		}
	}
	return nil
}
