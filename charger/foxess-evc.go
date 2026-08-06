package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// FoxESS EV Charger, Modbus TCP Protocol 1.6
// https://github.com/evcc-io/evcc/discussions/26218
// Section references below refer to that document.

// FoxESSEVC charger implementation
type FoxESSEVC struct {
	implement.Caps
	log        *util.Logger
	conn       *modbus.Connection
	mu         sync.Mutex // guards the tracked state below against the heartbeat goroutine
	current    float64    // tracks phase current, 0 if unset
	enabled    bool       // tracks enabled state
	phases     int        // tracks phase count; the charger does not report it
	setpoint   uint16     // last known value of foxRegMaxPower
	status     uint16     // last known value of foxRegStatus
	switchable bool       // charger switches 1p/3p on its own, derived from the power setpoint
	minPower   uint16     // min supported power
	maxPower   uint16     // max supported power
	minCurrent float64    // min supported current per phase
	maxCurrent float64    // max supported current per phase
}

// Register map per spec §2. Read-only and read/write registers are read with 0x03.
// Per §2 note (2) read/write registers must be written with 0x10, write-only registers with 0x06.
const (
	// read-only registers
	foxRegDeviceAddress = 0x1000 // device address (§2.1)
	foxRegSwVersion     = 0x1001 // software version, byte1 major / byte0 minor (§2.2)
	foxRegStopReason    = 0x1002 // reason the last charging session ended, see spec appendix 1 (§2.3)
	foxRegStatus        = 0x1003 // EVC status (§2.4)
	foxRegCpStatus      = 0x1004 // CP status (§2.5)
	foxRegCableStatus   = 0x1005 // CC status (§2.6)
	foxRegPortTemp      = 0x1006 // charging port temperature, 0.1°C, offset 50°C (§2.7)
	foxRegAmbientTemp   = 0x1007 // EVC environment temperature, 0.1°C, offset 50°C (§2.8)
	foxRegVoltages      = 0x1008 // A/B/C phase voltage, 3 registers, 0.1V (§2.9-§2.11)
	foxRegCurrents      = 0x100B // A/B/C phase current, 3 registers, 0.1A (§2.12-§2.14)
	foxRegPower         = 0x100E // active power, 0.1kW (§2.15)
	foxRegLockStatus    = 0x100F // electronic lock status (§2.16)
	foxRegPhaseSequence = 0x1010 // current phase sequence, only meaningful with a phase switch box (§2.17)
	foxRegMaxSupPower   = 0x1011 // max supported power, 0.1kW (§2.18)
	foxRegMinSupPower   = 0x1012 // min supported power, 0.1kW (§2.19)
	foxRegMaxSupCurrent = 0x1013 // max supported current per phase, 0.1A (§2.20)
	foxRegMinSupCurrent = 0x1014 // min supported current per phase, 0.1A (§2.21)
	foxRegAlarm         = 0x1015 // system alarm, bit-coded, see spec appendix 3 (§2.22)
	foxRegTotalEnergy   = 0x1016 // internal meter reading, uint32, 0.1kWh; never resets (§2.23)
	foxRegSessionEnergy = 0x1018 // energy of the current charge, uint32, 0.1kWh (§2.24)
	foxRegFault         = 0x101A // system fault, uint32, bit-coded, see spec appendix 2 (§2.25)
	foxRegRFID          = 0x101C // last RFID card, uint32 (§2.26)
	foxRegModel         = 0x101E // model code, 4 registers, ASCII (§2.27)
	foxRegSerial        = 0x1022 // serial number, 16 registers, ASCII (§2.28)

	// read/write registers (write with 0x10)
	foxRegWorkMode        = 0x3000 // work mode (§2.29)
	foxRegMaxCurrent      = 0x3001 // max charging current, 0.1A (§2.30)
	foxRegMaxPower        = 0x3002 // max charging power, 0.1kW (§2.31)
	foxRegChargeTime      = 0x3003 // allowable charge time, minutes (§2.32)
	foxRegChargeEnergy    = 0x3004 // allowable charge energy, kWh (§2.33)
	foxRegTimeValidity    = 0x3005 // command validity window, seconds (§2.34)
	foxRegDefaultCurrent  = 0x3006 // fallback current when the EMS connection is lost, 0.1A (§2.35)
	foxRegOtaStatus       = 0x3007 // OTA status (§2.36)
	foxRegOtaSize         = 0x3008 // OTA firmware size, uint32 (§2.37)
	foxRegAutoPhaseSwitch = 0x300A // single/three-phase automatic switching (§2.38)
	foxRegSwitchInterval  = 0x300B // min interval between phase switches, minutes (§2.39)
	foxRegLockControl     = 0x4000 // electronic lock control, write-only (§2.40)
	foxRegSessionControl  = 0x4001 // start/stop session, write-only (§2.41)
	foxRegPhaseControl    = 0x4002 // phase sequence switching, write-only (§2.42)
	foxRegRestart         = 0x4003 // restart, write-only (§2.43)
)

const (
	foxSessionNoAction   = 0 // session control values (§2.41)
	foxSessionStart      = 1
	foxSessionStop       = 2
	foxTimeValidity      = 60 // maximum command validity window in seconds (§2.34: 10-60s)
	foxDefaultCurrent    = 60 // 6.0A fallback current on EMS loss (§2.35: 6-32A)
	foxMinSwitchInterval = 5  // minimum phase switching interval in minutes (§2.39: 5-30min)

	// Without a phase-cutting box the charger derives the phase count from the power setpoint
	// (§2.38): >= 4.2kW three-phase, >= 1.4kW single-phase, below that charging is paused.
	// Setpoints are given in 0.1kW.
	foxMinPower3p = 42 // 4.2kW, the minimum power setpoint for a 3p charger
	foxMinPower1p = 14 // 1.4kW, the minimum power setpoint for a 1p or switchable charger
	foxMaxPower1p = 73 // 7.3kW, the maximum power setpoint for a 1p charger
)

// foxStatus values of the EVC status register (§2.4).
const (
	foxStatusIdle      = 0 // no faults, car not connected
	foxStatusConnect   = 1 // car connected, waiting for the start command
	foxStatusStart     = 2 // start command received, waiting for the car
	foxStatusCharging  = 3 // charging
	foxStatusPause     = 4 // charging suspended
	foxStatusFinish    = 5 // charging finished
	foxStatusFault     = 6 // faulty, cannot charge
	foxStatusReserved  = 7 // reserved
	foxStatusLocked    = 8 // locked, no operations possible
	foxStatusSwitching = 9 // undocumented: automatic phase switch in progress
)

func init() {
	registry.AddCtx("foxess-evc", NewFoxESSEVCFromConfig)
}

// NewFoxESSEVCFromConfig creates a FoxESS EV charger from generic config
func NewFoxESSEVCFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewFoxESSEVC(ctx, cc.TcpSettings)
}

// NewFoxESSEVC creates a FoxESS EV charger
func NewFoxESSEVC(ctx context.Context, settings modbus.TcpSettings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("foxess-evc")
	conn.Logger(log.TRACE)

	wb := &FoxESSEVC{
		Caps: implement.New(),
		log:  log,
		conn: conn,
	}

	// device limits are model-specific and constant, so read them once (§2.18-§2.21)
	minCurrent, err := wb.readUint16(foxRegMinSupCurrent)
	if err != nil {
		return nil, err
	}
	wb.minCurrent = float64(minCurrent) / 10

	maxCurrent, err := wb.readUint16(foxRegMaxSupCurrent)
	if err != nil {
		return nil, err
	}
	wb.maxCurrent = float64(maxCurrent) / 10

	if wb.minPower, err = wb.readUint16(foxRegMinSupPower); err != nil {
		return nil, err
	}
	if wb.maxPower, err = wb.readUint16(foxRegMaxSupPower); err != nil {
		return nil, err
	}

	if wb.minCurrent == 0 || wb.minCurrent > wb.maxCurrent {
		return nil, fmt.Errorf("invalid current limits: %.1f/%.1fA", wb.minCurrent, wb.maxCurrent)
	}
	if wb.minPower == 0 || wb.minPower > wb.maxPower {
		return nil, fmt.Errorf("invalid power limits: %d/%d", wb.minPower, wb.maxPower)
	}

	// derive the hardware phase count from the device limits
	wb.phases = 1
	if math.Round(float64(wb.maxPower)*100/(230*wb.maxCurrent)) >= 3 {
		wb.phases = 3
	}

	if wb.phases == 3 {
		autoSw, err := wb.readUint16(foxRegAutoPhaseSwitch)
		if err != nil {
			return nil, err
		}
		wb.switchable = autoSw > 0
	}

	if wb.switchable {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))

		// keep the internal charge pause and switching protection interval as short as possible
		if err := wb.writeReg(foxRegSwitchInterval, foxMinSwitchInterval); err != nil {
			wb.log.WARN.Printf("switch interval: %v", err)
		}
	}

	// seed the state from the charger
	if wb.status, err = wb.readUint16(foxRegStatus); err != nil {
		return nil, err
	}
	setpoint, err := wb.readSetpoint()
	if err != nil {
		return nil, err
	}
	if setpoint > 0 && wb.sessionActive(wb.status) {
		wb.enabled = true
		wb.current, wb.phases = wb.decodeSetpoint(setpoint)
	}

	// keep the charger from considering evcc offline; see heartbeat (§2.34).
	// widening the window to its maximum keeps the heartbeat rate low- firmware ranges differ,
	// so a rejected write is not fatal.
	if err := wb.writeReg(foxRegTimeValidity, foxTimeValidity); err != nil {
		wb.log.WARN.Printf("time validity: %v", err)
	}

	timeValidity, err := wb.readUint16(foxRegTimeValidity)
	if err != nil {
		return nil, err
	}
	if timeValidity == 0 {
		return nil, fmt.Errorf("invalid time validity: %d", timeValidity)
	}

	go wb.heartbeat(ctx, time.Duration(timeValidity)*time.Second/2)

	return wb, nil
}

// readUint16 reads a register as a uint16
func (wb *FoxESSEVC) readUint16(reg uint16) (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// readUint32 reads two consecutive registers as a big-endian uint32
func (wb *FoxESSEVC) readUint32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

// readString reads consecutive registers as a zero-padded ASCII string
func (wb *FoxESSEVC) readString(reg, words uint16) (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, words)
	if err != nil {
		return "", err
	}

	return bytesAsString(bytes.TrimRight(b, "\x00")), nil
}

// getPhaseValues returns 3 sequential register values scaled by divider
func (wb *FoxESSEVC) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

// writeReg writes a single read/write register (0x10)
func (wb *FoxESSEVC) writeReg(reg, val uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)

	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

// readSetpoint reads the power setpoint register and updates the cached value.
// Callers must hold mu.
func (wb *FoxESSEVC) readSetpoint() (uint16, error) {
	val, err := wb.readUint16(foxRegMaxPower)
	if err == nil {
		wb.setpoint = val
	}

	return val, err
}

// sessionActive reports whether the given status belongs to a running charging session.
// Only then is the power setpoint in effect (§2.31) instead of being restored to the device
// maximum, i.e. only then does a non-zero setpoint mean the charger is enabled.
func (wb *FoxESSEVC) sessionActive(status uint16) bool {
	switch status {
	case foxStatusStart, foxStatusCharging, foxStatusPause, foxStatusSwitching:
		return true
	default:
		return false
	}
}

// powerLimits returns the power setpoint bounds for the given phase count.
// A charger doing its own 1p/3p switching picks the phase count from the setpoint alone (§2.38),
// so the setpoint must stay inside the band belonging to the requested phase count. Otherwise the
// charger silently switches phases behind evcc's back- and while its minimum switching interval
// (§2.39) blocks the switch, a three-phase setpoint is delivered on a single phase.
func (wb *FoxESSEVC) powerLimits(phases int) (uint16, uint16) {
	lo, hi := wb.minPower, wb.maxPower

	if wb.switchable {
		if phases == 1 {
			lo, hi = foxMinPower1p, foxMinPower3p-1
		} else {
			lo = foxMinPower3p
		}
	}

	return lo, hi
}

// calcSetpoint converts the enable state and phase current into the power setpoint register value
func (wb *FoxESSEVC) calcSetpoint(enabled bool, current float64, phases int) uint16 {
	if !enabled {
		return 0
	}

	lo, hi := wb.powerLimits(phases)
	power := 230 * float64(phases) * min(max(current, wb.minCurrent), wb.maxCurrent)

	return min(max(uint16(math.Round(power/100)), lo), hi)
}

// decodeSetpoint converts a power setpoint register value back into the phase current and the
// phase count the charger derives from it
func (wb *FoxESSEVC) decodeSetpoint(setpoint uint16) (float64, int) {
	phases := wb.phases

	if wb.switchable {
		switch {
		case setpoint >= foxMinPower3p:
			phases = 3
		case setpoint >= foxMinPower1p:
			phases = 1
		}
	}

	return min(float64(setpoint)*100/(230*float64(phases)), wb.maxCurrent), phases
}

// applySetpoint writes the combined enable state and charging limit. Callers must hold mu.
func (wb *FoxESSEVC) applySetpoint(val uint16) error {
	if err := wb.writeReg(foxRegMaxPower, val); err != nil {
		return err
	}
	wb.setpoint = val

	return nil
}

// heartbeat re-asserts the power setpoint. The charger honours the last EMS command only for the
// duration of the command validity window (foxRegTimeValidity, §2.34) and reverts to its max
// supported power once it expires, so the interval must be shorter than that window.
func (wb *FoxESSEVC) heartbeat(ctx context.Context, interval time.Duration) {
	for tick := time.Tick(interval); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		wb.mu.Lock()
		err := wb.writeReg(foxRegMaxPower, wb.setpoint)
		wb.mu.Unlock()

		if err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *FoxESSEVC) Status() (api.ChargeStatus, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	s, err := wb.readUint16(foxRegStatus)
	if err != nil {
		return api.StatusNone, err
	}
	wb.status = s

	switch s {
	case foxStatusIdle:
		return api.StatusA, nil

	case foxStatusConnect, foxStatusStart, foxStatusPause, foxStatusSwitching, foxStatusFinish:
		return api.StatusB, nil

	case foxStatusCharging:
		return api.StatusC, nil

	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

var _ api.StatusReasoner = (*FoxESSEVC)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *FoxESSEVC) StatusReason() (api.Reason, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	// uses the status cached by Status(), which the loadpoint calls immediately before
	switch wb.status {
	case foxStatusConnect:
		return api.ReasonWaitingForAuthorization, nil

	case foxStatusFinish:
		return api.ReasonDisconnectRequired, nil

	default:
		return api.ReasonUnknown, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *FoxESSEVC) Enabled() (bool, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	val, err := wb.readSetpoint()
	if err != nil {
		return false, err
	}

	if val == 0 {
		wb.enabled = false
	}

	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *FoxESSEVC) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if err := wb.applySetpoint(wb.calcSetpoint(enable, wb.current, wb.phases)); err != nil {
		return err
	}

	wb.enabled = enable

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *FoxESSEVC) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*FoxESSEVC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *FoxESSEVC) MaxCurrentMillis(current float64) error {
	if current < wb.minCurrent {
		return fmt.Errorf("invalid current: %.1fA", current)
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	if err := wb.applySetpoint(wb.calcSetpoint(wb.enabled, current, wb.phases)); err != nil {
		return err
	}

	wb.current = current

	return nil
}

var _ api.CurrentLimiter = (*FoxESSEVC)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *FoxESSEVC) GetMinMaxCurrent() (float64, float64, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	lo, hi := wb.powerLimits(wb.phases)

	minCurrent, _ := wb.decodeSetpoint(lo)
	maxCurrent, _ := wb.decodeSetpoint(hi)

	return max(wb.minCurrent, minCurrent), maxCurrent, nil
}

var _ api.CurrentGetter = (*FoxESSEVC)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *FoxESSEVC) GetMaxCurrent() (float64, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	// outside an active session the setpoint may be restored to the max supported power (§2.31),
	// which the loadpoint would adopt as the offered current
	if !wb.sessionActive(wb.status) {
		return 0, api.ErrNotAvailable
	}

	val, err := wb.readSetpoint()
	if err != nil {
		return 0, err
	}

	current, _ := wb.decodeSetpoint(val)

	return current, nil
}

var _ api.Meter = (*FoxESSEVC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *FoxESSEVC) CurrentPower() (float64, error) {
	val, err := wb.readUint16(foxRegPower)
	if err != nil {
		return 0, err
	}

	return float64(val) * 100, nil
}

var _ api.MeterEnergy = (*FoxESSEVC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *FoxESSEVC) TotalEnergy() (float64, error) {
	energy, err := wb.readUint32(foxRegTotalEnergy)
	if err != nil {
		return 0, err
	}

	return float64(energy) / 10, nil
}

//
// removed since broken, see https://github.com/evcc-io/evcc/pull/32371
// var _ api.ChargeRater = (*FoxESSEVC)(nil)

var _ api.PhaseCurrents = (*FoxESSEVC)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *FoxESSEVC) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(foxRegCurrents, 10)
}

var _ api.PhaseVoltages = (*FoxESSEVC)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *FoxESSEVC) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(foxRegVoltages, 10)
}

var _ api.Identifier = (*FoxESSEVC)(nil)

// Identify implements the api.Identifier interface
func (wb *FoxESSEVC) Identify() (string, error) {
	id, err := wb.readUint32(foxRegRFID)
	if err != nil {
		return "", err
	}

	if id == 0 {
		return "", nil
	}

	return fmt.Sprintf("%08X", id), nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *FoxESSEVC) phases1p3p(phases int) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	// the setpoint band depends on the phase count, so it needs to be rewritten right away-
	// the loadpoint does not necessarily re-issue MaxCurrent after a phase switch
	if err := wb.applySetpoint(wb.calcSetpoint(wb.enabled, wb.current, phases)); err != nil {
		return err
	}

	wb.phases = phases

	return nil
}

// getPhases implements the api.PhaseGetter interface
func (wb *FoxESSEVC) getPhases() (int, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	// Since the setpoint is kept inside the band of the requested phase count, this is the count
	// the charger will settle on- its minimum switching interval (§2.39) may delay the actual switch.
	_, phases := wb.decodeSetpoint(wb.setpoint)

	return phases, nil
}

var _ api.Diagnosis = (*FoxESSEVC)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *FoxESSEVC) Diagnose() {
	if val, err := wb.readUint16(foxRegSwVersion); err == nil {
		fmt.Printf("\tSoftware version:\t%d.%d\n", val>>8, val&0xFF)
	}
	if s, err := wb.readString(foxRegModel, 4); err == nil {
		fmt.Printf("\tModel:\t%s\n", s)
	}
	if s, err := wb.readString(foxRegSerial, 16); err == nil {
		fmt.Printf("\tSerial:\t%s\n", s)
	}
	fmt.Printf("\tMax. Phases:\t%dp\n", wb.phases)
	fmt.Printf("\tAuto phase switching:\t%v\n", wb.switchable)
	fmt.Printf("\tPower range:\t%.1f-%.1fkW\n", float64(wb.minPower)/10, float64(wb.maxPower)/10)
	fmt.Printf("\tCurrent range:\t%.1f-%.1fA\n", wb.minCurrent, wb.maxCurrent)
	if val, err := wb.readUint16(foxRegWorkMode); err == nil {
		fmt.Printf("\tWork mode:\t%d\n", val)
	}
	if val, err := wb.readUint16(foxRegStopReason); err == nil {
		fmt.Printf("\tStop reason:\t%d\n", val) // see spec appendix 1
	}
}
