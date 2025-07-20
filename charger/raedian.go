package charger

// LICENSE

// Copyright (c) 2025 Mobility Pulse

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Raedian is an api.Charger implementation for Raedian NEO and NEX AC Wallbox chargers.
type Raedian struct {
	*embed
	log          *util.Logger
	conn         *modbus.Connection
	current      uint16
	energyFactor float64 // Factor for energy conversion
}

const (
	raedianRegSerialNumber             = 0x8000 // Size 4, RO
	raedianRegFirmwareVersion          = 0x8004 // Size 2, RO
	raedianRegMaxRatedSettableCurrent  = 0x8006 // Size 2, RO (0.001 A)
	raedianRegErrorCode                = 0x8008 // Size 2, RO
	raedianRegSocketLockState          = 0x800A // Size 2, RO
	raedianRegChargingState            = 0x800C // Size 2, RO
	raedianRegCurrentChargingCurrentLimit = 0x800E // Size 2, RO (0.001 A)
	raedianRegChargingCurrentL1        = 0x8010 // Size 2, RO (0.001 A)
	raedianRegChargingCurrentL2        = 0x8012 // Size 2, RO (0.001 A)
	raedianRegChargingCurrentL3        = 0x8014 // Size 2, RO (0.001 A)
	raedianRegVoltagePhase1            = 0x8016 // Size 2, RO (0.1 V)
	raedianRegVoltagePhase2            = 0x8018 // Size 2, RO (0.1 V)
	raedianRegVoltagePhase3            = 0x801A // Size 2, RO (0.1 V)
	raedianRegActivePower              = 0x801C // Size 2, RO (1 W)
	raedianRegEnergyDeliveredSession   = 0x801E // Size 2, RO (1 Wh)
	raedianRegSetChargingCurrentLimit  = 0x8100 // Size 2, WO (0.001 A)
	raedianRegSetChargingPhase         = 0x8102 // Size 1, WO
	raedianRegStartStopChargingSession = 0x8105 // Size 1, WO
)

func init() {
	registry.AddCtx("raedian", NewRaedianFromConfig)
}

//go:generate go tool decorate -f decorateRaedian -b *Raedian -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.StatusReasoner,StatusReason,func() (api.Reason, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewRaedianFromConfig creates a new Raedian charger from generic config
func NewRaedianFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed              `mapstructure:",squash"`
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1, // Default Modbus Slave ID for Raedian is 1
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewRaedian(ctx, cc.embed, cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	// optional features
	var (
		currentPower, totalEnergy             func() (float64, error)
		currents                              func() (float64, float64, float64, error)
		voltages                              func() (float64, float64, float64, error)
		identify                              func() (string, error)
		reason                                func() (api.Reason, error)
		phasesS                               func(int) error
		phasesG                               func() (int, error)
	)

	// All readable data (0x8000-0x801E) is available on both NEO and NEX.
	// Assume meter capabilities are present for monitoring.
	currentPower = wb.currentPower
	totalEnergy = wb.totalEnergy
	currents = wb.currents
	voltages = wb.voltages
	identify = wb.identify // Serial Number can act as identifier

	// Raedian protocol defines specific states for charging status.
	reason = wb.statusReason

	// Phase switching is available for NEX AC Wallbox
	// The document mentions "Set charging phase (in roadmap)" for 0x8102, so it might not be fully implemented or available on all devices.
	// For now, assuming it might be available based on charger type or future firmware.
	phasesS = wb.phases1p3p
	phasesG = wb.getPhases


	return decorateRaedian(wb, currentPower, totalEnergy, currents, voltages, identify, reason, phasesS, phasesG), nil
}

// NewRaedian creates a new Raedian charger
func NewRaedian(ctx context.Context, embed embed, uri string, slaveID uint8) (*Raedian, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("raedian")
	conn.Logger(log.TRACE)

	wb := &Raedian{
		embed:        &embed,
		log:          log,
		conn:         conn,
		energyFactor: 1.0, // Default to 1.0, adjust if resolution changes (e.g., 0.1 Wh vs 1 Wh)
	}

	// Check firmware for specific energy factor adjustments if needed
	// The document mentions "In software versions below 1.2.1 the registers 1502 and 1036
	// falsely report the value in “Wh” instead of “0.1 Wh” for KEBA P40.
	// Raedian document specifies 1 Wh for 0x801E, so no factor adjust for now unless found in future revisions.

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Raedian) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegChargingState, 2) // Size 2 for 0x800C
	if err != nil {
		return api.StatusNone, err
	}

	state := binary.BigEndian.Uint32(b)

	switch state {
	case 0x00: // State A: Idle
		return api.StatusA, nil
	case 0x01: // State B1: EV Plug in, pending authorization
		return api.StatusB, nil
	case 0x02: // State B2: EV Plug in, EVSE ready for charging (PWM)
		return api.StatusB, nil
	case 0x03: // State C1: EV Ready for charge, S2 closed (no PWM)
		return api.StatusC, nil
	case 0x04: // State C2: Charging Contact closed, energy delivering.
		return api.StatusC, nil
	case 0x05: // Other
		return api.StatusB, nil // Treat as B for now, indicates EV is connected but not charging
	default:
		return api.StatusNone, fmt.Errorf("invalid charging state: %d", state)
	}
}

// statusReason implements the api.StatusReasoner interface
func (wb *Raedian) statusReason() (api.Reason, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegChargingState, 2)
	if err != nil {
		return api.ReasonUnknown, err
	}

	state := binary.BigEndian.Uint32(b)

	// Referencing the Raedian protocol for states and their meanings
	switch state {
	case 0x01: // State B1: EV Plug in, pending authorization
		return api.ReasonWaitingForAuthorization, nil
	case 0x05: // Other states where charging is not active
		// Check for error code if available, otherwise general pause reason
		errorCode, err := wb.getErrorCode()
		if err != nil {
			return api.ReasonUnknown, err
		}
		if errorCode != 0 {
			return api.ReasonChargerError, nil
		}
		// If no specific error, it could be paused manually or by limit setting.
		// The protocol mentions "Setting charging current limit between 0 to 5999 will set the charger to pause charging session"
		currentLimit, err := wb.getCurrentChargingCurrentLimit()
		if err != nil {
			return api.ReasonUnknown, err
		}
		if currentLimit >= 0 && currentLimit < 6000 { // 6000 is 6A minimum
			return api.ReasonTooLittlePower, nil // Or api.ReasonChargePaused, depending on interpretation
		}
		return api.ReasonChargePaused, nil
	default:
		return api.ReasonUnknown, nil
	}
}

// getErrorCode reads the error code from the charger
func (wb *Raedian) getErrorCode() (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegErrorCode, 2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

// getCurrentChargingCurrentLimit reads the current charging current limit
func (wb *Raedian) getCurrentChargingCurrentLimit() (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegCurrentChargingCurrentLimit, 2)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

// Enabled implements the api.Charger interface
func (wb *Raedian) Enabled() (bool, error) {
	// The protocol document implies that setting current limit to 0-5999mA pauses charging.
	// So, we can check the current set limit to infer if it's "enabled" (allowing charge).
	b, err := wb.conn.ReadHoldingRegisters(raedianRegCurrentChargingCurrentLimit, 2)
	if err != nil {
		return false, err
	}

	currentLimit := binary.BigEndian.Uint32(b)
	return currentLimit >= 6000, nil // 6000 = 6A, minimum valid charging current
}

// Enable implements the api.Charger interface
func (wb *Raedian) Enable(enable bool) error {
	var val uint16
	if enable {
		if wb.current == 0 {
			// If no specific current was set before, default to a safe value (e.g., 16A)
			val = 16000
		} else {
			val = wb.current
		}
	} else {
		// Setting current limit between 0 to 5999 will set the charger to pause charging session.
		val = 0 // Setting to 0 to pause
	}

	_, err := wb.conn.WriteSingleRegister(raedianRegSetChargingCurrentLimit, val)
	if err == nil {
		wb.current = val // Store the last set current
	}
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Raedian) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Raedian)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Raedian) MaxCurrentMillis(current float64) error {
	// Raedian expects current in mA (multiplied by 1000) for registers 0x8006, 0x800E, 0x8100.
	// But note in 0x8100 example says "Number after decimal point will be ignored" when setting.
	// It also states valid range 6A to 32A (6000-32000 decimal).
	if current < 6.0 || current > 32.0 {
		return fmt.Errorf("invalid current %.1fA. Valid range is 6A to 32A.", current)
	}

	currMa := uint16(current * 1000)

	_, err := wb.conn.WriteSingleRegister(raedianRegSetChargingCurrentLimit, currMa)
	if err == nil {
		wb.current = currMa // Store the last set current
	}
	return err
}

// currentPower implements the api.Meter interface
func (wb *Raedian) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegActivePower, 2) // Active power in W
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint32(b)), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Raedian) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegEnergyDeliveredSession, 2) // Energy delivered in Wh
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint32(b)) / 1000.0, nil // Convert Wh to kWh
}

// currents implements the api.PhaseCurrents interface
func (wb *Raedian) currents() (float64, float64, float64, error) {
	// Currents are in 0.001A (mA)
	bL1, err1 := wb.conn.ReadHoldingRegisters(raedianRegChargingCurrentL1, 2)
	bL2, err2 := wb.conn.ReadHoldingRegisters(raedianRegChargingCurrentL2, 2)
	bL3, err3 := wb.conn.ReadHoldingRegisters(raedianRegChargingCurrentL3, 2)

	if err1 != nil { return 0, 0, 0, err1 }
	if err2 != nil { return 0, 0, 0, err2 }
	if err3 != nil { return 0, 0, 0, err3 }

	l1 := float64(binary.BigEndian.Uint32(bL1)) / 1000.0 // Convert mA to A
	l2 := float64(binary.BigEndian.Uint32(bL2)) / 1000.0 // Convert mA to A
	l3 := float64(binary.BigEndian.Uint32(bL3)) / 1000.0 // Convert mA to A

	return l1, l2, l3, nil
}

// voltages implements the api.PhaseVoltages interface
func (wb *Raedian) voltages() (float64, float64, float64, error) {
	// Voltages are in 0.1V
	bL1, err1 := wb.conn.ReadHoldingRegisters(raedianRegVoltagePhase1, 2)
	bL2, err2 := wb.conn.ReadHoldingRegisters(raedianRegVoltagePhase2, 2)
	bL3, err3 := wb.conn.ReadHoldingRegisters(raedianRegVoltagePhase3, 2)

	if err1 != nil { return 0, 0, 0, err1 }
	if err2 != nil { return 0, 0, 0, err2 }
	if err3 != nil { return 0, 0, 0, err3 }

	l1 := float64(binary.BigEndian.Uint32(bL1)) / 10.0 // Convert 0.1V to V
	l2 := float64(binary.BigEndian.Uint32(bL2)) / 10.0 // Convert 0.1V to V
	l3 := float64(binary.BigEndian.Uint32(bL3)) / 10.0 // Convert 0.1V to V

	return l1, l2, l3, nil
}

// identify implements the api.Identifier interface
func (wb *Raedian) identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegSerialNumber, 4) // Serial Number is 4 registers
	if err != nil {
		return "", err
	}

	// Assuming the serial number is a readable string or can be converted meaningfully
	// The document states "Serial Number: 4" (registers), "UNSIGNED" Data Type.
	// This usually means 4 16-bit registers (8 bytes).
	// Let's convert it to a hex string for uniqueness, or decimal if it's represented as a large number.
	// Example provided in the document for other registers: Value 10000(DEC) for current.
	// For serial number, a direct conversion to string is better if it's text, otherwise to a large int or hex.
	// Assuming it's a numeric serial, reading as a single 64-bit unsigned integer is most robust.
	serial := binary.BigEndian.Uint64(b)
	return strconv.FormatUint(serial, 10), nil // Return as decimal string
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Raedian) phases1p3p(phases int) error {
	var u uint16
	switch phases {
	case 1:
		u = 0x01 // 0x01: Set the charging to single phase
	case 3:
		u = 0x02 // 0x02: Set the charging to three phases
	default:
		return fmt.Errorf("invalid phases: %d. Only 1 or 3 phases are supported.", phases)
	}

	_, err := wb.conn.WriteSingleRegister(raedianRegSetChargingPhase, u)
	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *Raedian) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegSetChargingPhase, 1)
	if err != nil {
		return 0, err
	}
	state := binary.BigEndian.Uint16(b)
	switch state {
	case 0x01:
		return 1, nil
	case 0x02:
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown phase setting: %d", state)
	}
}

var _ api.Diagnosis = (*Raedian)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Raedian) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegSerialNumber, 4); err == nil {
		serial := binary.BigEndian.Uint64(b)
		fmt.Printf("\tSerial Number:\t%s\n", strconv.FormatUint(serial, 10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegFirmwareVersion, 2); err == nil {
		// Assuming firmware is presented as two 16-bit values (Major.Minor)
		// Or if it's like Keba with 4 bytes: Major.Minor.Patch.Build
		// The document simply says "Size: 2, Data Type: UNSIGNED".
		// Let's assume it's two 16-bit numbers for now, representing major and minor.
		major := binary.BigEndian.Uint16(b[0:2])
		minor := binary.BigEndian.Uint16(b[2:4])
		fmt.Printf("\tFirmware Version:\t%d.%d\n", major, minor)
	}
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegErrorCode, 2); err == nil {
		errorCode := binary.BigEndian.Uint32(b)
		fmt.Printf("\tError Code:\t%d\n", errorCode)
		if errorCode != 0 {
			fmt.Printf("\t(Refer to user manual for error details)\n")
		}
	}
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegSocketLockState, 2); err == nil {
		lockState := binary.BigEndian.Uint32(b)
		fmt.Printf("\tSocket Lock State:\t0x%04X\n", lockState)
	}
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegChargingState, 2); err == nil {
		chargingState := binary.BigEndian.Uint32(b)
		fmt.Printf("\tCharging State:\t0x%02X\n", chargingState)
	}
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegCurrentChargingCurrentLimit, 2); err == nil {
		limit := float64(binary.BigEndian.Uint32(b)) / 1000.0
		fmt.Printf("\tCurrent Charging Current Limit:\t%.3fA\n", limit)
	}
	if b, err := wb.conn.ReadHoldingRegisters(raedianRegActivePower, 2); err == nil {
		power := float64(binary.BigEndian.Uint32(b))
		fmt.Printf("\tActive Power:\t%.1fW\n", power)
	}
}
