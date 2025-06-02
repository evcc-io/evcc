package charger

// LICENSE

// Copyright (c) 2025 andig

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
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

	// Meter
	kathreinRegVoltages         = 0x0030 // float32 Line 1 to Neutral Volts (V)
	kathreinRegCurrents         = 0x0036 // float32 Line 1 Current (A)
	kathreinRegPowers           = 0x003C // float32 Line 1 Power (W)
	kathreinRegTotalActivePower = 0x0054 // float32 Total active power (W)
	kathreinRegTotalEnergy      = 0x005C // float32 Total Energy (since production) (Wh)

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

	// EMS-Control - Control register (uint16)
	//   0x8000 : Enable EMS-Control
	//   Default = 0x0000 (EMS-Control disabled)
	kathreinRegEMSControlRegister = 0x00A0

	// EMS-Control - Setpoint Relais-Matrix (uint16)
	//   0x0001 : Line 1
	//   0x0002 : Line 2 (reserved for future)
	//   0x0004 : Line 3 (reserved for future)
	//   Default = 0x0007 (3 Lines)
	kathreinRegEMSSetpointRelais = 0x00A1

	// EMS-Control - Setpoint Charging Current (mA) (uint16)
	//   0 : Charging Paused
	//   6000 … 32000 : Charging
	//   0xFFFF : Charging Cancel
	//   Default = max. Current according to Power-Class
	kathreinRegEMSSetpointChargingCurrent = 0x00A2

	// EMS-Control - Timeout period (s) (uint16)
	//   0 : Timeout deactivated (default)
	//   >0 : Timeout activated (each Setpoint-Write-Cycle resets the Timer)
	kathreinRegEMSTimeoutPeriod = 0x00A3

	// EMS-Control - Timeout fallback pattern (uint16)
	//   0x0001 : Line 1
	//   0x0002 : Line 2
	//   0x0004 : Line 3
	//   Default = 0x0007 (3 Lines)
	kathreinRegEMSTimeOutFallbackPattern = 0x00A4

	// EMS-Control - Timeout fallback current (mA) (uint16)
	//   0, 6000 … 32000 mA
	//   Default = 6000  mA
	kathreinRegEMSTimeOutFallbackCurrent = 0x00A5
)

func init() {
	registry.AddCtx("kathrein", NewKathreinFromConfig)
}

// NewKathreinFromConfig creates a Kathrein charger from generic config
func NewKathreinFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cc modbus.TcpSettings

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewKathrein(ctx, cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	return wb, nil
}

// NewKathrein creates Kathrein charger
func NewKathrein(ctx context.Context, uri string, id uint8) (*Kathrein, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
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
		curr: 6000,
	}

	return wb, err
}

// getPhaseValues returns 3 sequential register values
func (wb *Kathrein) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(math.Float32frombits(binary.BigEndian.Uint32(b[4*i:]))) / divider
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Kathrein) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegCPState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 0: // A (EV not detected, standby)
		return api.StatusA, nil
	case 1: // B (EV detected, ready to charge)
		return api.StatusB, nil
	case 2, 3: // C (EV charging), D (EV charging with fan)
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Kathrein) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSSetpointChargingCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Kathrein) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	// EMS-Control must be enabled before sending first WriteReg Command
	if _, err := wb.conn.WriteSingleRegister(kathreinRegEMSControlRegister, 0x8000); err != nil {
		return err
	}

	_, err := wb.conn.WriteSingleRegister(kathreinRegEMSSetpointChargingCurrent, u)

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

	curr := uint16(current * 1e3)

	_, err := wb.conn.WriteSingleRegister(kathreinRegEMSSetpointChargingCurrent, curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*Kathrein)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Kathrein) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegTotalActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(math.Float32frombits(binary.BigEndian.Uint32(b))), err
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

var _ api.ChargeTimer = (*Kathrein)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Kathrein) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegChargingDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.ChargeRater = (*Kathrein)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Kathrein) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegChargingEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

var _ api.MeterEnergy = (*Kathrein)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Kathrein) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(math.Float32frombits(binary.BigEndian.Uint32(b))) / 1e3, err
}

var _ api.PhaseSwitcher = (*Kathrein)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Kathrein) Phases1p3p(phases int) error {
	var u uint16 = 0x0007 // Three phase charging

	if phases == 1 {
		u = 0x0001 // One phase charging
	}

	enabled, err := wb.Enabled()
	if err != nil {
		return err
	}

	// EMS-Control must be enabled before sending first WriteReg Command
	if _, err := wb.conn.WriteSingleRegister(kathreinRegEMSControlRegister, 0x8000); err != nil {
		return err
	}

	// Switch phases
	if _, err := wb.conn.WriteSingleRegister(kathreinRegEMSSetpointRelais, u); err != nil {
		return err
	}

	// Disable and re-enable charging to apply the new phase setting
	if err := wb.Enable(false); err != nil {
		return err
	}

	if enabled {
		return wb.Enable(true)
	}

	return nil
}

var _ api.PhaseGetter = (*Kathrein)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *Kathrein) GetPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSSetpointRelais, 1)
	if err != nil {
		return 0, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 0x0001:
		return 1, nil
	case 0x0007:
		return 3, nil
	default:
		return 0, nil
	}
}

var _ api.Diagnosis = (*Kathrein)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Kathrein) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegHeader, 1); err == nil {
		fmt.Printf("Header - Mapping Version:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegDeviceInfo, 1); err == nil {
		fmt.Printf("Device - Info:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegDeviceNumber, 8); err == nil {
		fmt.Printf("Device - Number:\t%s\n", b)
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegDeviceType, 8); err == nil {
		fmt.Printf("Device - Type:\t%s\n", b)
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegDeviceSerial, 8); err == nil {
		fmt.Printf("Device - Serial:\t%s\n", b)
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegChargingState, 1); err == nil {
		fmt.Printf("EVSE - Charging-State:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegErrorState, 1); err == nil {
		fmt.Printf("EVSE - Error-State:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegCPState, 1); err == nil {
		fmt.Printf("EVSE - CP-State:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegRelaisState, 1); err == nil {
		fmt.Printf("EVSE - Relais-State:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegGrantedCurrent, 1); err == nil {
		fmt.Printf("EVSE - Granted Current:\t%d mA\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegGrantedPower, 1); err == nil {
		fmt.Printf("EVSE - Granted Power:\t%d W\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSControlRegister, 1); err == nil {
		fmt.Printf("EMS-Control - Control-Register:\t%d\n", b[0])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSSetpointRelais, 1); err == nil {
		fmt.Printf("EMS-Control - Setpoint Relais-Matrix:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSSetpointChargingCurrent, 1); err == nil {
		fmt.Printf("EMS-Control - Setpoint Charging Current:\t%d mA\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSTimeoutPeriod, 1); err == nil {
		fmt.Printf("EMS-Control - Timeout Period:\t%d s\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSTimeOutFallbackPattern, 1); err == nil {
		fmt.Printf("EMS-Control - Timeout Fallback Pattern:\t%d\n", b[1])
	}

	if b, err := wb.conn.ReadHoldingRegisters(kathreinRegEMSTimeOutFallbackCurrent, 1); err == nil {
		fmt.Printf("EMS-Control - Timeout Fallback Current:\t%d mA\n", binary.BigEndian.Uint16(b))
	}
}
