package raedian

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Raedian charger implementation
type Charger struct {
	conn *modbus.Connection
	log  *util.Logger
	id   uint8
}

// NewCharger creates a new Raedian charger
func NewCharger(
	uri string,
	slaveID uint8,
	baudrate int,
	comset string,
	delay time.Duration,
) (*Charger, error) {
	conn, err := modbus.NewConnection(uri, slaveID, baudrate, comset, delay)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("raedian")
	conn.Logger(log)

	c := &Charger{
		conn: conn,
		log:  log,
		id:   slaveID,
	}

	return c, nil
}

// Status implements the api.Charger interface
func (c *Charger) Status() (api.ChargeStatus, error) {
	// Read register 0x800C (decimal 32780) for charging state.
	[span_0](start_span)// The register size is 2 (2 bytes)[span_0](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x800C, 1)
	if err != nil {
		return api.StatusA, fmt.Errorf("could not read status register: %w", err)
	}

	// Status values from the document:
	// 0x00: State A: Idle
	// 0x01: State B1: EV Plug in, pending authorization
	// 0x02: State B2: EV Plug in, EVSE ready for charging
	// 0x04: State C2: Charging Contact closed, energy delivering
	status := b[1]

	switch status {
	case 0x00:
		return api.StatusA, nil
	case 0x01, 0x02:
		return api.StatusB, nil
	case 0x03, 0x04:
		return api.StatusC, nil
	case 0x05:
		return api.StatusD, nil // or StatusB, depending on interpretation of 'Other'
	default:
		return api.StatusE, fmt.Errorf("unknown status: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (c *Charger) Enabled() (bool, error) {
	[span_1](start_span)// A charger is considered "enabled" if the charging current limit is above the minimum charging current[span_1](end_span).
	[span_2](start_span)// We read register 0x800E (decimal 32782) which contains the 'Current charging current limit'[span_2](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x800E, 1)
	if err != nil {
		return false, fmt.Errorf("could not read current limit: %w", err)
	}

	currentLimit := uint16(b[0])<<8 + uint16(b[1])
	[span_3](start_span)// A value between 0 to 5999 will set the charger to pause charging[span_3](end_span).
	// Therefore, any value >= 6000mA means it's enabled.
	return currentLimit >= 6000, nil
}

// Enable implements the api.Charger interface
func (c *Charger) Enable(enable bool) error {
	[span_4](start_span)// The document mentions writing to register 0x8105 (decimal 33029) to start/stop the session[span_4](end_span).
	[span_5](start_span)// 0x00: Start, 0x01: Stop[span_5](end_span).
	var value uint16
	if !enable {
		value = 0x01
	} else {
		value = 0x00
	}

	_, err := c.conn.WriteSingleRegister(0x8105, value)
	if err != nil {
		return fmt.Errorf("could not enable/disable: %w", err)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *Charger) MaxCurrent(current float64) error {
	[span_6](start_span)// The document specifies that charging current can be set from 6A to 32A[span_6](end_span).
	if current < 6 {
		[span_7](start_span)current = 0 // Setting the current limit to < 6A will pause charging[span_7](end_span).
	}

	[span_8](start_span)// Register 0x8100 (decimal 33024) sets the charging current limit[span_8](end_span).
	[span_9](start_span)// The value should be in mA[span_9](end_span).
	[span_10](start_span)// The wallbox ignores the decimal part[span_10](end_span).
	value := uint16(current * 1000)

	_, err := c.conn.WriteSingleRegister(0x8100, value)
	if err != nil {
		return fmt.Errorf("could not set current: %w", err)
	}

	return nil
}

// CurrentPower implements the api.Meter interface
func (c *Charger) CurrentPower() (float64, error) {
	[span_11](start_span)// Read register 0x801C (decimal 32796) for Active Power in Watts[span_11](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x801C, 1)
	if err != nil {
		return 0, fmt.Errorf("could not read power register: %w", err)
	}

	[span_12](start_span)// The value is an UNSIGNED 16-bit integer[span_12](end_span).
	val := uint16(b[0])<<8 + uint16(b[1])

	return float64(val), nil
}

// TotalEnergy implements the api.Meter interface
func (c *Charger) TotalEnergy() (float64, error) {
	[span_13](start_span)// Read register 0x801E (decimal 32798) for Energy delivered in charging session in Wh[span_13](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x801E, 1)
	if err != nil {
		return 0, fmt.Errorf("could not read energy register: %w", err)
	}

	[span_14](start_span)// The value is an UNSIGNED 16-bit integer[span_14](end_span).
	val := uint16(b[0])<<8 + uint16(b[1])

	// Return the value in Wh, evcc expects kWh so we must convert.
	return float64(val) / 1000, nil
}

// Currents implements the api.PhaseCurrents interface
func (c *Charger) Currents() (float64, float64, float64, error) {
	[span_15](start_span)// Read registers for Charging current L1 (0x8010), L2 (0x8012), L3 (0x8014)[span_15](end_span).
	[span_16](start_span)[span_17](start_span)// Each is a 16-bit UNSIGNED value in mA[span_16](end_span)[span_17](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x8010, 6)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("could not read currents registers: %w", err)
	}

	[span_18](start_span)// Extract values and convert from mA to A (scale 0.001)[span_18](end_span).
	l1 := float64(uint16(b[0])<<8+uint16(b[1])) / 1000
	l2 := float64(uint16(b[2])<<8+uint16(b[3])) / 1000
	l3 := float64(uint16(b[4])<<8+uint16(b[5])) / 1000

	return l1, l2, l3, nil
}

// Lock implements the api.Lock interface
func (c *Charger) Lock() (api.LockState, error) {
	[span_19](start_span)// Read register 0x800A (decimal 32778) for Socket lock state[span_19](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x800A, 1)
	if err != nil {
		return api.LockUnknown, fmt.Errorf("could not read socket lock state: %w", err)
	}

	val := uint16(b[0])<<8 + uint16(b[1])

	[span_20](start_span)// Supported values from the document[span_20](end_span):
	// 0x0000: No cable is plugged.
	// 0x0001: Cable is connected to the charging station unlocked
	// 0x0011: Cable is connected to the charging station locked
	// 0x0101: Cable is connected to the charging station and the electric vehicle, locked in charging station
	// 0x0111: Cable is connected to the charging station and the electric vehicle, unlocked in charging station
	switch val {
	case 0x0000, 0x0001:
		return api.LockUnlocked, nil
	case 0x0011, 0x0101:
		return api.LockLocked, nil
	case 0x0111:
		return api.LockUnlocked, nil
	default:
		return api.LockUnknown, fmt.Errorf("unknown lock state: %d", val)
	}
}

// CurrentLimit implements the api.Charger interface
func (c *Charger) CurrentLimit() (float64, error) {
	[span_21](start_span)// Read the `Current charging current limit` from register 0x800E[span_21](end_span).
	b, err := c.conn.ReadHoldingRegisters(0x800E, 1)
	if err != nil {
		return 0, fmt.Errorf("could not read current limit: %w", err)
	}

	val := uint16(b[0])<<8 + uint16(b[1])

	[span_22](start_span)// The value is in mA, so we convert to A[span_22](end_span).
	return float64(val) / 1000, nil
}

