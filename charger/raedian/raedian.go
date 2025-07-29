package raedian

import (
	"context"
	"encoding/binary"
	"fmt"
	
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Modbus Register Addresses
const (
	[span_0](start_span)[span_1](start_span)raedianRegSerial                = 0x8000 // 4 regs, Read Only[span_0](end_span)[span_1](end_span)
	[span_2](start_span)[span_3](start_span)raedianRegFirmwareVersion       = 0x8004 // 2 regs, Read Only[span_2](end_span)[span_3](end_span)
	[span_4](start_span)[span_5](start_span)raedianRegMaxRatedCurrent       = 0x8006 // 2 regs, Read Only[span_4](end_span)[span_5](end_span)
	[span_6](start_span)raedianRegErrorCode             = 0x8008 // 2 regs, Read Only[span_6](end_span)
	[span_7](start_span)[span_8](start_span)raedianRegSocketLockState       = 0x800A // 2 regs, Read Only[span_7](end_span)[span_8](end_span)
	[span_9](start_span)[span_10](start_span)raedianRegChargingState         = 0x800C // 2 regs, Read Only[span_9](end_span)[span_10](end_span)
	[span_11](start_span)[span_12](start_span)raedianRegCurrentChargingLimit  = 0x800E // 2 regs, Read Only[span_11](end_span)[span_12](end_span)
	[span_13](start_span)[span_14](start_span)raedianRegChargingCurrentL1     = 0x8010 // 2 regs, Read Only[span_13](end_span)[span_14](end_span)
	[span_15](start_span)[span_16](start_span)raedianRegChargingCurrentL2     = 0x8012 // 2 regs, Read Only[span_15](end_span)[span_16](end_span)
	[span_17](start_span)[span_18](start_span)raedianRegChargingCurrentL3     = 0x8014 // 2 regs, Read Only[span_17](end_span)[span_18](end_span)
	[span_19](start_span)[span_20](start_span)raedianRegVoltageL1             = 0x8016 // 2 regs, Read Only[span_19](end_span)[span_20](end_span)
	[span_21](start_span)[span_22](start_span)raedianRegVoltageL2             = 0x8018 // 2 regs, Read Only[span_21](end_span)[span_22](end_span)
	[span_23](start_span)[span_24](start_span)raedianRegVoltageL3             = 0x801A // 2 regs, Read Only[span_23](end_span)[span_24](end_span)
	[span_25](start_span)[span_26](start_span)raedianRegActivePower           = 0x801C // 2 regs, Read Only[span_25](end_span)[span_26](end_span)
	[span_27](start_span)[span_28](start_span)raedianRegEnergyDelivered       = 0x801E // 2 regs, Read Only[span_27](end_span)[span_28](end_span)
	[span_29](start_span)[span_30](start_span)raedianRegSetChargingCurrent    = 0x8100 // 2 regs, Write Only[span_29](end_span)[span_30](end_span)
	[span_31](start_span)[span_32](start_span)raedianRegSetChargingPhase      = 0x8102 // 1 reg, Write Only (in roadmap)[span_31](end_span)[span_32](end_span)
	[span_33](start_span)[span_34](start_span)raedianRegStartStopSession      = 0x8105 // 1 reg, Write Only[span_33](end_span)[span_34](end_span)
)

// Charger is an api.Charger implementation
type Charger struct {
	conn *modbus.Connection
	log  *util.Logger
	id   uint8
}

// NewRaedianFromConfig creates a new Raedian charger
// This function would typically be called by the `evcc` configuration loader.
func NewRaedianFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}
	return NewRaedian(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
}

// NewRaedian creates a new charger
func NewRaedian(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}
	log := util.NewLogger("raedian")
	conn.Logger(log.TRACE)

	wb := &Charger{
		conn: conn,
		log:  log,
		id:   slaveID,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (c *Charger) Status() (api.ChargeStatus, error) {
	b, err := c.conn.ReadHoldingRegisters(raedianRegChargingState, 1)
	if err != nil {
		return api.StatusNone, fmt.Errorf("could not read status register: %w", err)
	}
	status := binary.BigEndian.Uint16(b)

	switch status {
	case 0x0000:
		[span_35](start_span)return api.StatusA, nil // Idle[span_35](end_span)
	case 0x0001, 0x0002:
		[span_36](start_span)return api.StatusB, nil // EV plugged in, waiting or ready[span_36](end_span)
	case 0x0003, 0x0004:
		[span_37](start_span)return api.StatusC, nil // Charging[span_37](end_span)
	default:
		// We'll treat any other status as "none" or unknown.
		return api.StatusNone, fmt.Errorf("unknown status: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (c *Charger) Enabled() (bool, error) {
	b, err := c.conn.ReadHoldingRegisters(raedianRegCurrentChargingLimit, 1)
	if err != nil {
		return false, fmt.Errorf("could not read current limit: %w", err)
	}
	currentLimit := binary.BigEndian.Uint16(b)
	[span_38](start_span)// A limit of 0-5999 mA will pause charging[span_38](end_span)
	return currentLimit >= 6000, nil
}

// Enable implements the api.Charger interface
func (c *Charger) Enable(enable bool) error {
	var value uint16
	if !enable {
		[span_39](start_span)value = 0x01 // Stop charging[span_39](end_span)
	} else {
		[span_40](start_span)value = 0x00 // Start charging[span_40](end_span)
	}
	_, err := c.conn.WriteSingleRegister(raedianRegStartStopSession, value)
	if err != nil {
		return fmt.Errorf("could not enable/disable: %w", err)
	}
	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *Charger) MaxCurrent(current int64) error {
	if current < 6 {
		[span_41](start_span)current = 0 // Setting the current to < 6A pauses charging[span_41](end_span)
	}
	[span_42](start_span)[span_43](start_span)value := uint16(current * 1000) // The value should be in mA[span_42](end_span)[span_43](end_span)
	_, err := c.conn.WriteSingleRegister(raedianRegSetChargingCurrent, value)
	if err != nil {
		return fmt.Errorf("could not set current: %w", err)
	}
	return nil
}

var _ api.Meter = (*Charger)(nil)

// CurrentPower implements the api.Meter interface
func (c *Charger) CurrentPower() (float64, error) {
	b, err := c.conn.ReadHoldingRegisters(raedianRegActivePower, 1)
	if err != nil {
		return 0, fmt.Errorf("could not read power register: %w", err)
	}
	val := binary.BigEndian.Uint16(b)
	return float64(val), nil
}

var _ api.MeterEnergy = (*Charger)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Charger) TotalEnergy() (float64, error) {
	b, err := c.conn.ReadHoldingRegisters(raedianRegEnergyDelivered, 1)
	if err != nil {
		return 0, fmt.Errorf("could not read energy register: %w", err)
	}
	val := binary.BigEndian.Uint16(b)
	[span_44](start_span)// The value is in Wh, we convert to kWh[span_44](end_span)
	return float64(val) / 1000, nil
}

var _ api.PhaseCurrents = (*Charger)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Charger) Currents() (float64, float64, float64, error) {
	b, err := c.conn.ReadHoldingRegisters(raedianRegChargingCurrentL1, 3)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("could not read currents registers: %w", err)
	}
	[span_45](start_span)l1 := float64(binary.BigEndian.Uint16(b[0:2])) / 1000[span_45](end_span)
	[span_46](start_span)l2 := float64(binary.BigEndian.Uint16(b[2:4])) / 1000[span_46](end_span)
	[span_47](start_span)l3 := float64(binary.BigEndian.Uint16(b[4:6])) / 1000[span_47](end_span)
	return l1, l2, l3, nil
}

var _ api.PhaseVoltages = (*Charger)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *Charger) Voltages() (float64, float64, float64, error) {
	b, err := c.conn.ReadHoldingRegisters(raedianRegVoltageL1, 3)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("could not read voltage registers: %w", err)
	}
	[span_48](start_span)l1 := float64(binary.BigEndian.Uint16(b[0:2])) / 10 // The value is in 0.1V[span_48](end_span)
	[span_49](start_span)l2 := float64(binary.BigEndian.Uint16(b[2:4])) / 10[span_49](end_span)
	[span_50](start_span)l3 := float64(binary.BigEndian.Uint16(b[4:6])) / 10[span_50](end_span)
	return l1, l2, l3, nil
}

var _ api.PhaseSwitcher = (*Charger)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *Charger) Phases1p3p(phases int) error {
	[span_51](start_span)c.log.WARN.Println("Phase switching is not yet supported for Raedian Wallbox, as per the documentation it is a feature 'in roadmap'.")[span_51](end_span)
	[span_52](start_span)[span_53](start_span)// The register 0x8102 is marked "in roadmap"[span_52](end_span)[span_53](end_span)
	return fmt.Errorf("phase switching not supported")
}

var _ api.Diagnosis = (*Charger)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *Charger) Diagnose() {
	c.log.INFO.Println("--- Raedian Wallbox Diagnosis ---")
	
	if b, err := c.conn.ReadHoldingRegisters(raedianRegSerial, 2); err == nil {
		[span_54](start_span)fmt.Printf("\tSerial: %d\n", binary.BigEndian.Uint32(b))[span_54](end_span)
	} else {
		fmt.Printf("\tSerial: ERROR - %v\n", err)
	}

	if b, err := c.conn.ReadHoldingRegisters(raedianRegFirmwareVersion, 1); err == nil {
		[span_55](start_span)fmt.Printf("\tFirmware Version: %d\n", binary.BigEndian.Uint16(b))[span_55](end_span)
	} else {
		fmt.Printf("\tFirmware Version: ERROR - %v\n", err)
	}

	if b, err := c.conn.ReadHoldingRegisters(raedianRegMaxRatedCurrent, 1); err == nil {
		[span_56](start_span)fmt.Printf("\tMax Rated Current: %.3f A\n", float64(binary.BigEndian.Uint16(b))/1000)[span_56](end_span)
	} else {
		fmt.Printf("\tMax Rated Current: ERROR - %v\n", err)
	}

	if b, err := c.conn.ReadHoldingRegisters(raedianRegErrorCode, 1); err == nil {
		[span_57](start_span)fmt.Printf("\tError Code: %d\n", binary.BigEndian.Uint16(b))[span_57](end_span)
	} else {
		fmt.Printf("\tError Code: ERROR - %v\n", err)
	}
	
	if b, err := c.conn.ReadHoldingRegisters(raedianRegSocketLockState, 1); err == nil {
		[span_58](start_span)fmt.Printf("\tSocket Lock State: %d\n", binary.BigEndian.Uint16(b))[span_58](end_span)
	} else {
		fmt.Printf("\tSocket Lock State: ERROR - %v\n", err)
	}

	c.log.INFO.Println("--------------------------------")
}
