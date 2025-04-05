package charger

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// Kathrein charger implementation
type Kathrein struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	// Device related registers
	kathreinRegHeader       = 0x0000 // uint16 Current Version = 0x0001
	kathreinRegDeviceNumber = 0x0001 // String
	kathreinRegDeviceType   = 0x0009 // String
	kathreinRegDeviceSerial = 0x0011 // String

	// Device Info (uint16)
	//   Bits 0-1 : Power-Class
	//     0x0001 : 11kW (3 x 16A)
	//     0x0002 : 22kW (3 x 32A)
	//   Bit 4 : Cable / Plug
	//     0x0010 : "0" = Cable, "1" = Plug
	//   Bit 7 : Eichrecht
	//     0x0080 : "0" = Standard, "1" = Eichrecht
	//   Bits 8-15 : Relais-Capability
	//     0x0100 : L1 only (1 Line)
	//     0x0200 : L2 only (1 Line)
	//     0x0400 : L3 only (1 Line)
	//     0x1000 : L1 and L2 (2 Lines)
	//     0x2000 : L1 and L3 (2 Lines)
	//     0x4000 : L2 and L3 (2 Lines)
	//     0x8000 : L1 and L2 and L3 (3 Lines)
	kathreinRegDeviceInfo = 0x0019

	// Device Line Information (uint16)
	//   0 : L1-L2-L3 (default)
	//   1 : L1-L3-L2 (invalid phase rotation)
	//   2 : L2-L3-L1
	//   3 : L2-L1-L3 (invalid phase rotation)
	//   4 : L3-L1-L2
	//   5 : L3-L2-L1 (invalid phase rotation)
	kathreinRegDeviceLineMapping = 0x001A
	kathreinRegDeviceTimestamp   = 0x001B // uint64 Local Date & Time according to "UNIX-Format". Seconds since 01.01.1970 00:00:00

	// Meter
	kathreinRegL1Voltage          = 0x0030 // float32 Line 1 to Neutral Volts (V)
	kathreinRegL2Voltage          = 0x0032 // float32 Line 2 to Neutral Volts (V)
	kathreinRegL3Voltage          = 0x0034 // float32 Line 3 to Neutral Volts (V)
	kathreinRegL1Current          = 0x0036 // float32 Line 1 Current (A)
	kathreinRegL2Current          = 0x0038 // float32 Line 2 Current (A)
	kathreinRegL3Current          = 0x003A // float32 Line 3 Current (A)
	kathreinRegL1Power            = 0x003C // float32 Line 1 Power (W)
	kathreinRegL2Power            = 0x003E // float32 Line 2 Power (W)
	kathreinRegL3Power            = 0x0040 // float32 Line 3 Power (W)
	kathreinRegL1ApparentPower    = 0x0042 // float32 Line 1 Apparent power (VA)
	kathreinRegL2ApparentPower    = 0x0044 // float32 Line 2 Apparent power (VA)
	kathreinRegL3ApparentPower    = 0x0046 // float32 Line 3 Apparent power (VA)
	kathreinRegL1ReactivePower    = 0x0048 // float32 Line 1 Reactive power (VAr)
	kathreinRegL2ReactivePower    = 0x004A // float32 Line 2 Reactive power (VAr)
	kathreinRegL3ReactivePower    = 0x004C // float32 Line 3 Reactive power (VAr)
	kathreinRegL1PowerFactor      = 0x004E // float32 Line 1 Power factor (cos φ, 0.0 … 1.0)
	kathreinRegL2PowerFactor      = 0x0050 // float32 Line 2 Power factor (cos φ, 0.0 … 1.0)
	kathreinRegL3PowerFactor      = 0x0052 // float32 Line 3 Power factor (cos φ, 0.0 … 1.0)
	kathreinRegTotalActivePower   = 0x0054 // float32 Total active power (W)
	kathreinRegTotalApparentPower = 0x0056 // float32 Total apparent power (VA)
	kathreinRegTotalReactivePower = 0x0058 // float32 Total reactive power (VAr)
	kathreinRegTotalPowerFactor   = 0x005A // float32 Total power factor (cos φ, 0.0 … 1.0)
	kathreinRegTotalEnergy        = 0x005C // float32 Total Energy (since production) (kWh)
	kathreinRegFrequencyLine      = 0x005E // float32 Frequency line (Hz)

	// EVSE - Charging state (uint16)
	//   0 : Idle
	//   1 : EV Connected
	//   2 : Authentication Waiting
	//   3 : Authentication Confirmed
	//   4 : Charging Active
	//   5 : Charging Paused
	//   6 : Charging Completed
	//   7 : RFID-Pairing
	//   0xFFFF : Error
	kathreinRegChargingState = 0x0060

	// EVSE - Error state (uint16)
	//   0x0000 : No Error
	//   0x0001 : Relais welded
	//   0x0002 : Residual DC-Current detected (RCD)
	//   0x0004 : Socket Lock-Detection Error
	//   0x0008 : Charging Overcurrent
	//   0x0010 : CP-D: Ventilation not available
	//   0x0020 : CP-E: Short-Circuit (CP-PE)
	//   0x0040 : CP-F: Loop broken (CP-PE)
	//   0x0080 : PP-Error (Short-Circuit)
	//   0x8000 : Internal Error
	kathreinRegErrorState = 0x0061

	// EVSE - PP-State (uint16)
	//  Plug-Variant:
	//    0 : Cable not connected
	//    13 : 13 A (1500 Ω)
	//    20 : 20 A (680 Ω)
	//    32 : 32 A (220 Ω)
	//    63 : 63 A (100 Ω)
	//    0xFFFF : Error (invalid PP-Resistor)
	//  Cable-Variant:
	//    0 : fix Cable mounted
	kathreinRegPPState = 0x0062

	// EVSE - CP-State (uint16)
	//   0 : A (EV not detected, standby)
	//   1 : B (EV detected, ready to charge)
	//   2 : C (EV charging)
	//   3 : D (EV charging with fan)
	//   4 : E (CP Short-Circuit)
	//   5 : F (EVSE not available, CP = -12VDC)
	//   all other values : Undefined / Error
	kathreinRegCPState = 0x0063

	// EVSE - Relais State (uint16)
	//   0x0000 : Relais OFF
	//   0x0001 : Relais L1 activated
	//   0x0002 : Relais L2 activated
	//   0x0004 : Relais L3 activated
	kathreinRegRelaisState = 0x0064

	kathreinRegGrantedCurrent   = 0x0065 // uint16 Granted charging current per Line (related to CP-Signal) [0, [6000 … 32000]] (mA)
	kathreinRegGrantedPower     = 0x0066 // uint16 Granted charging power [1380 … 22080] @230VAC (W)
	kathreinRegChargingDuration = 0x0067 // uint32 Duration Charging (s)
	kathreinRegChargingEnergy   = 0x0069 // uint32 Energy Charging Energy (per charging session) (Wh)
	kathreinRegTariffInfo       = 0x006B // uint16 Charging Tariff Info & Currency (0,001€)
	kathreinRegCurrentTariff    = 0x006C // uint16 Charging Tariff for active charging session (0,001€)

	// Control (R/W)
	kathreinRegNextTariff = 0x006D // uint16 Charging Tariff for next charging session (0,001€)

	// EMS-Control - Control register (uint16)
	//   0x8000 : Enable EMS-Control
	//   Default = 0x0000 (EMS-Control disabled)
	kathreinRegControlRegister = 0x00A0

	// EMS-Control - Setpoint Relais-Matrix (uint16)
	//   0x0001 : Line 1
	//   0x0002 : Line 2 (reserved for future)
	//   0x0004 : Line 3 (reserved for future)
	//   Default = 0x0007 (3 Lines)
	kathreinRegSetpointRelais = 0x00A1

	// EMS-Control - Setpoint Charging Current (mA) (uint16)
	//   0 : Charging Paused
	//   6000 … 32000 : Charging
	//   0xFFFF : Charging Cancel
	//   Default = max. Current according to Power-Class
	kathreinRegSetpointChargingCurrent = 0x00A2

	// EMS-Control - Timeout period (s) (uint16)
	//   0 : Timeout deactivated (default)
	//   >0 : Timeout activated (each Setpoint-Write-Cycle resets the Timer)
	kathreinRegTimeoutPeriod = 0x00A3

	// EMS-Control - Timeout fallback pattern (uint16)
	//   0x0001 : Line 1
	//   0x0002 : Line 2
	//   0x0004 : Line 3
	//   Default = 0x0007 (3 Lines)
	kathreinTimeOutFallbackPattern = 0x00A4

	// EMS-Control - Timeout fallback current (mA) (uint16)
	//   0, 6000 … 32000 mA
	//   Default = 6000  mA
	kathreinTimeOutFallbackCurrent = 0x00A5
)

var (
	kathreinRegVoltages = []uint16{kathreinRegL1Voltage, kathreinRegL2Voltage, kathreinRegL3Voltage} // uint16 V
	kathreinRegCurrents = []uint16{kathreinRegL1Current, kathreinRegL1Current, kathreinRegL1Current} // uint16 A
	kathreinRegPowers   = []uint16{kathreinRegL1Power, kathreinRegL1Power, kathreinRegL2Power}       // uint16 W
)

func init() {
	registry.AddCtx("kathrein", NewKathreinFromConfig)
}

// NewKathreinromConfig creates a Kathrein charger from generic config
func NewKathreinFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 248,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewKathrein(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
}

// NewKathrein creates Kathrein charger
func NewKathrein(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("kathrein")
	conn.Logger(log.TRACE)

	wb := &Kathrein{
		log:  log,
		conn: conn,
		curr: 60,
	}

	return wb, err
}

// getPhaseValues returns 3 non-sequential register values
func (wb *Kathrein) getPhaseValues(regs []uint16, divider float64) (float64, float64, float64, error) {
	var res [3]float64
	for i, reg := range regs {
		b, err := wb.conn.ReadInputRegisters(reg, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = rs485.RTUUint16ToFloat64(b) / divider
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Kathrein) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegCPState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 0: // A (EV not detected, standby)
		return api.StatusA, nil
	case 1: // B (EV detected, ready to charge)
		return api.StatusB, nil
	case 2: // C (EV charging)
		return api.StatusC, nil
	case 3: // D (EV charging with fan)
		return api.StatusD, nil
	case 4: // E (CP Short-Circuit)
		return api.StatusE, nil
	case 5: // F (EVSE not available, CP = -12VDC)
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Kathrein) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegControlRegister, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 0, nil
}

// Enable implements the api.Charger interface
func (wb *Kathrein) Enable(enable bool) error {
	var u uint16 = 0x8000 // Enable EMS-Control
	if !enable {
		u = 0x0000 // Disable EMS-Control
	}

	_, err := wb.conn.WriteSingleRegister(kathreinRegControlRegister, u)

	if err == nil && enable {
		_, err = wb.conn.WriteSingleRegister(kathreinRegSetpointChargingCurrent, wb.curr)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Kathrein) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Kathrein)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Kathrein) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	curr := uint16(current)

	_, err := wb.conn.WriteSingleRegister(kathreinRegSetpointChargingCurrent, curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*Kathrein)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Kathrein) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegTotalActivePower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b), err
}

var _ api.PhaseCurrents = (*Kathrein)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Kathrein) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(kathreinRegCurrents, 1)
}

var _ api.PhaseVoltages = (*Kathrein)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Kathrein) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(kathreinRegVoltages, 1)
}

var _ api.PhasePowers = (*Kathrein)(nil)

// Powers implements the api.PhasePowers interface
func (wb *Kathrein) Powers() (float64, float64, float64, error) {
	return wb.getPhaseValues(kathreinRegPowers, 1)
}

var _ api.ChargeTimer = (*Kathrein)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Kathrein) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegChargingDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint16(b)) * time.Second, nil
}

var _ api.ChargeRater = (*Kathrein)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Kathrein) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegChargingEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, err
}

var _ api.MeterEnergy = (*Kathrein)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Kathrein) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, err
}

var _ api.PhaseSwitcher = (*Kathrein)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Kathrein) Phases1p3p(phases int) error {
	var u uint16 = 0x0007 // Three phase charging

	if phases == 1 {
		u = 0x0001 // One phase charging
	}

	enabled, err := wb.Enabled()
	if err == nil && enabled {
		if err = wb.Enable(false); err != nil {
			return err
		}
	}

	// Switch phases
	_, err = wb.conn.WriteSingleRegister(kathreinRegSetpointRelais, u)

	// Re-enable charging if it was previously enabled
	if err == nil && enabled {
		err = wb.Enable(true)
	}

	return err
}

var _ api.PhaseGetter = (*Kathrein)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *Kathrein) GetPhases() (int, error) {
	b, err := wb.conn.ReadInputRegisters(kathreinRegSetpointRelais, 1)
	binary.BigEndian.Uint16(b)
	if err != nil {
		return 0, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 0x0001,
		0x0002,
		0x0003: // Single phase
		return 1, nil
	case 0x0007:
		return 3, nil
	default:
		return 0, err
	}
}

var _ api.Diagnosis = (*Kathrein)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Kathrein) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(sgRegSetOutI, 1); err == nil {
		fmt.Printf("\tSetOutI:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(sgRegPhaseSwitch, 1); err == nil {
		fmt.Printf("\tPhaseSwitch:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(sgRegUnavailable, 1); err == nil {
		fmt.Printf("\tUnavailable:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(sgRegRemoteControl, 1); err == nil {
		fmt.Printf("\tRemoteControl:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegWorkMode, 1); err == nil {
		fmt.Printf("\tWorkMode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPhase, 1); err == nil {
		fmt.Printf("\tPhase:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPhaseSwitchStatus, 1); err == nil {
		fmt.Printf("\tPhasesState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegStartMode, 1); err == nil {
		fmt.Printf("\tStartMode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegState, 1); err == nil {
		fmt.Printf("\tState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegRemCtrlStatus, 1); err == nil {
		fmt.Printf("\tRemCtrlStatus:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPowerRequest, 1); err == nil {
		fmt.Printf("\tPowerRequest:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPowerFlag, 1); err == nil {
		fmt.Printf("\tPowerFlag:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
