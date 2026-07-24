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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/core/loadpoint"
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
	log           *util.Logger
	conn          *modbus.Connection
	mu            sync.Mutex
	current       uint16  // last setpoint in register units (0.1A or 0.1kW depending on pbox)
	currentPhases int     // phase count used to derive current from the power setpoint (non-pbox only)
	enabled       bool    // tracks enabled state for the heartbeat
	lastEnabled   bool    // last enabled state successfully sent to the charger
	pbox          bool    // phase-cutting box present; uses current register instead of power
	finished      bool    // charger reported status 5 (finish); cleared only once the car disconnects
	minCurrent    float64 // min supported current per phase, 0 if unknown
	maxCurrent    float64 // max supported current per phase, 0 if unknown
	lp            loadpoint.API
}

const (
	// read-only registers (0x03)
	foxRegSwVersion     = 0x1001 // software version, byte1 major / byte0 minor
	foxRegStopReason    = 0x1002 // reason the last charging session ended, see spec appendix 1
	foxRegStatus        = 0x1003 // EVC status
	foxRegVoltages      = 0x1008 // A/B/C phase voltage, 3 registers, 0.1V
	foxRegCurrents      = 0x100B // A/B/C phase current, 3 registers, 0.1A
	foxRegPower         = 0x100E // active power, 0.1kW
	foxRegPhaseSequence = 0x1010 // current phase sequence
	foxRegMaxSupCurrent = 0x1013 // max supported current per phase, 0.1A (§2.20)
	foxRegMinSupCurrent = 0x1014 // min supported current per phase, 0.1A (§2.21)
	foxRegAlarm         = 0x1015 // system alarm, bit-coded, see spec appendix 3
	// 0x1018 is documented as "total charging energy" but describes the energy consumed by the
	// currently charging car (§2.24) and resets when charging stops. 0x1016 is the reading of the
	// meter inside the charger (§2.23) and never resets, so it is the actual total energy.
	foxRegTotalEnergy = 0x1016 // total energy, uint32, 0.1kWh; never resets
	foxRegFault       = 0x101A // system fault, uint32, bit-coded, see spec appendix 2
	foxRegRFID        = 0x101C // last RFID card, uint32

	// read/write registers (write with 0x10)
	foxRegWorkMode       = 0x3000 // work mode
	foxRegMaxCurrent     = 0x3001 // max charging current, 0.1A
	foxRegMaxPower       = 0x3002 // max charging power, 0.1kW
	foxRegTimeValidity   = 0x3005 // command validity window, seconds
	foxRegDefaultCurrent = 0x3006 // fallback current when the EMS connection is lost, 0.1A
	foxRegAutoSwitch     = 0x300A // single/three-phase automatic switching (no PBOX)
	foxRegSwitchInterval = 0x300B // min interval between phase switches, minutes

	// write-only registers (write with 0x06)
	foxRegChargingControl = 0x4001 // start/stop charging
	foxRegPhaseSwitching  = 0x4002 // phase sequence switching (requires PBOX)

	foxWorkModeControlled = 0 // external command required
	foxChargingStart      = 1
	foxChargingStop       = 2
	foxTimeValidity       = 60 // maximum command validity window in seconds (§2.34: 10-60s)
	foxDefaultCurrent     = 60 // 6.0A fallback current on EMS loss (§2.35: 6-32A)
	foxSwitchInterval     = 5  // minimum phase switching interval in minutes (§2.39: 5-30min)

	// Without a phase-cutting box the charger derives the phase count from the power setpoint
	// (§2.38): >= 4.2kW three-phase, >= 1.4kW single-phase, below that charging is paused.
	// Setpoints are given in 0.1kW.
	foxPower3p = 42
	foxPower1p = 14

	// foxHeartbeatInterval is the interval at which the heartbeat runs.
	// Must be less than foxTimeValidity so the charger never considers evcc offline.
	foxHeartbeatInterval = 25 * time.Second
)

// foxStatus values of the EVC status register (§2.4). Status 9 is undocumented but reported
// by the charger while an automatic phase switch is in progress.
const (
	foxStatusIdle      = 0 // no faults, car not connected
	foxStatusConnect   = 1 // car connected, waiting for the start command
	foxStatusStart     = 2 // start command received, waiting for the car
	foxStatusCharging  = 3 // charging
	foxStatusPause     = 4 // charging suspended
	foxStatusFinish    = 5 // charging finished
	foxStatusFault     = 6 // faulty, cannot charge
	foxStatusLocked    = 8 // locked, no operations possible
	foxStatusSwitching = 9 // undocumented: automatic phase switch in progress
)

// foxFaults are the bit names of the system fault register (spec appendix 2)
var foxFaults = []string{
	"emergency stop", "overvoltage", "undervoltage", "overcurrent", "charging port temperature",
	"PE grounding", "leakage current", "frequency", "CP", "connector", "AC contactor",
	"electronic lock", "breaker", "CC", "external meter communication", "metering chip",
	"environment temperature", "access control",
}

// foxAlarms are the bit names of the system alarm register (spec appendix 3)
var foxAlarms = []string{"card reader", "phase cutting box", "phase loss"}

// foxStopReasons are the stop reason codes of the last charging session (spec appendix 1)
var foxStopReasons = []string{
	"none", "stopped on command", "timed charging completed", "S2 timeout",
	"charging pause timeout", "emergency stop button pressed", "abnormal CP voltage",
	"abnormality in drawing a charging connector", "abnormal AC contactor",
	"abnormal electronic lock", "abnormal card reader", "abnormal overcurrent",
	"abnormal overvoltage", "abnormal undervoltage", "charging port over-temperature",
	"abnormal leakage current", "N line reverse connection", "abnormal frequency",
	"charging stop button pressed", "abnormal circuit breaker", "phase loss", "abnormal PE",
	"abnormal external electric meter", "environment over-temperature", "metering chip failure",
	"access control failure", "PBOX switching phase sequence is abnormal",
	"reach the predetermined energy",
}

func init() {
	registry.AddCtx("foxess-modbus", NewFoxESSEVCFromConfig)
}

// NewFoxESSEVCFromConfig creates a FoxESS EV charger from generic config
func NewFoxESSEVCFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Pbox               bool
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewFoxESSEVC(ctx, cc.URI, cc.ID, cc.Pbox)
}

// NewFoxESSEVC creates a FoxESS EV charger
func NewFoxESSEVC(ctx context.Context, uri string, slaveID uint8, pbox bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("foxess-modbus")
	conn.Logger(log.TRACE)

	wb := &FoxESSEVC{
		Caps: implement.New(),
		log:  log,
		conn: conn,
		pbox: pbox,
	}

	if b, err := wb.conn.ReadHoldingRegisters(foxRegSwVersion, 1); err == nil {
		v := binary.BigEndian.Uint16(b)
		log.DEBUG.Printf("software version: %d.%02d", v>>8, v&0xFF)
	}

	// take control of the charger. The register is state-dependent (§2.29), so a failed
	// write must not prevent startup as long as the charger already is in controlled mode.
	if err := wb.ensureReg(foxRegWorkMode, foxWorkModeControlled); err != nil {
		wb.log.WARN.Printf("work mode: %v (make sure the charger is set to Modbus TCP in the Fox Switch app)", err)
	}
	if err := wb.ensureReg(foxRegTimeValidity, foxTimeValidity); err != nil {
		return nil, fmt.Errorf("time validity: %w", err)
	}

	// limit the current the charger falls back to when it considers evcc offline (§2.35)
	if err := wb.ensureReg(foxRegDefaultCurrent, foxDefaultCurrent); err != nil {
		wb.log.WARN.Printf("default current: %v", err)
	}

	// device limits are model-specific and constant, so read them once (§2.20/§2.21)
	if b, err := wb.conn.ReadHoldingRegisters(foxRegMinSupCurrent, 1); err == nil {
		wb.minCurrent = float64(binary.BigEndian.Uint16(b)) / 10
	}
	if b, err := wb.conn.ReadHoldingRegisters(foxRegMaxSupCurrent, 1); err == nil {
		wb.maxCurrent = float64(binary.BigEndian.Uint16(b)) / 10
	}

	// the charger goes to fallback current if no setpoint is received within the
	// time validity window, so we must re-send the setpoint periodically
	go wb.heartbeat(ctx)

	if pbox {
		// phase switching is commanded explicitly via the phase-cutting box
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))

		// only the current setpoint offers sub-ampere resolution; the power setpoint used
		// without a phase-cutting box is limited to 0.1kW steps
		implement.Has(wb, implement.ChargerEx(wb.maxCurrentMillis))
	} else {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3pAuto))

		// without a phase-cutting box the charger switches phases autonomously
		// based on the power setpoint (spec §2.38); the register defaults to
		// enabled, so a failed write here is not fatal
		if err := wb.ensureReg(foxRegAutoSwitch, 1); err != nil {
			wb.log.WARN.Printf("auto switch: %v", err)
		}

		// while the interval is not met the charger pauses instead of switching phases
		// (§2.39), so keep it as short as the charger allows
		if err := wb.ensureReg(foxRegSwitchInterval, foxSwitchInterval); err != nil {
			wb.log.WARN.Printf("switch interval: %v", err)
		}
	}

	// seed the enabled state from the charger so the heartbeat does not send a
	// spurious stop command on its first tick. The setpoint registers are not
	// seeded: outside of charging they read back the device maximum (§2.30/§2.31).
	if b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1); err == nil {
		wb.lastEnabled = foxEnabled(binary.BigEndian.Uint16(b))
	}

	return wb, nil
}

// foxEnabled reports whether the given EVC status means the charger is enabled, i.e. it would
// deliver energy if the car asked for it. Pause (§2.4) is included because the charger also
// enters it on its own when the power setpoint is below its threshold or the phase switching
// interval is not met (§2.38/§2.39), not only when the car suspends charging.
func foxEnabled(s uint16) bool {
	switch s {
	case foxStatusStart, foxStatusCharging, foxStatusPause, foxStatusSwitching:
		return true
	default:
		return false
	}
}

// foxCharging reports whether the charger is actively charging. The setpoint registers only
// take effect in this state (§2.30/§2.31).
func foxCharging(s uint16) bool {
	return s == foxStatusCharging || s == foxStatusSwitching
}

// foxBits returns the names of the bits set in mask
func foxBits(mask uint32, names []string) []string {
	var res []string
	for i, name := range names {
		if mask&(1<<i) != 0 {
			res = append(res, name)
		}
	}
	return res
}

// writeReg writes a single read/write register (0x10)
func (wb *FoxESSEVC) writeReg(reg, val uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)

	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

// ensureReg writes a value to a read/write register only if it differs from
// the current value, avoiding spurious write errors on registers that reject
// redundant writes (e.g. Modbus exception 3 when value is unchanged).
func (wb *FoxESSEVC) ensureReg(reg, val uint16) error {
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return err
	}
	if binary.BigEndian.Uint16(b) == val {
		return nil
	}
	return wb.writeReg(reg, val)
}

// heartbeat keeps the charger from considering evcc offline. Any message resets the Time
// Validity timer (§2.34), so the status read alone is a sufficient keepalive and prevents
// the charger from falling back to the Default Current setting and auto-starting. While
// charging, the last setpoint is re-sent to keep the charger on the requested limit.
func (wb *FoxESSEVC) heartbeat(ctx context.Context) {
	for tick := time.Tick(foxHeartbeatInterval); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		wb.mu.Lock()
		cur := wb.current
		pbox := wb.pbox
		enabled := wb.enabled
		lastEnabled := wb.lastEnabled
		wb.mu.Unlock()

		// keepalive read, also used to decide whether a setpoint write would be accepted
		b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
		if err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
			continue
		}
		status := binary.BigEndian.Uint16(b)

		switch {
		case !enabled && lastEnabled:
			// transition: send stop once
			if _, err = wb.conn.WriteSingleRegister(foxRegChargingControl, foxChargingStop); err == nil {
				wb.mu.Lock()
				wb.lastEnabled = false
				wb.mu.Unlock()
			}

		case enabled && cur != 0 && foxCharging(status):
			// the setpoint registers only take effect while charging (§2.30/§2.31);
			// writing them in any other state is rejected with a modbus exception
			if pbox {
				err = wb.ensureReg(foxRegMaxCurrent, cur)
			} else {
				err = wb.ensureReg(foxRegMaxPower, cur)
			}
		}

		if err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// readUint32 reads two consecutive registers as a big-endian uint32
func (wb *FoxESSEVC) readUint32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
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

// Status implements the api.Charger interface
func (wb *FoxESSEVC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case foxStatusIdle:
		// car disconnected: the charger accepts a start command again
		wb.mu.Lock()
		wb.finished = false
		wb.mu.Unlock()

		return api.StatusA, nil

	case foxStatusConnect, foxStatusStart, foxStatusPause:
		return api.StatusB, nil

	case foxStatusFinish:
		// the charger will reject a restart until the car disconnects; remember this
		// so Enable(true) can refuse early instead of hitting a modbus exception
		wb.mu.Lock()
		first := !wb.finished
		wb.finished = true
		wb.mu.Unlock()

		if first {
			wb.logStopReason()
		}

		return api.StatusB, nil

	case foxStatusCharging, foxStatusSwitching:
		return api.StatusC, nil

	case foxStatusFault:
		return api.StatusNone, fmt.Errorf("fault: %s", strings.Join(wb.faults(), ", "))

	case foxStatusLocked:
		return api.StatusNone, errors.New("charger locked")

	default: // reserved
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// faults returns the active fault and alarm conditions (spec appendix 2 and 3)
func (wb *FoxESSEVC) faults() []string {
	var res []string

	if v, err := wb.readUint32(foxRegFault); err == nil {
		res = foxBits(v, foxFaults)
	}

	if b, err := wb.conn.ReadHoldingRegisters(foxRegAlarm, 1); err == nil {
		res = append(res, foxBits(uint32(binary.BigEndian.Uint16(b)), foxAlarms)...)
	}

	if len(res) == 0 {
		res = []string{"unknown"}
	}

	return res
}

// logStopReason logs why the charging session ended (§2.3, spec appendix 1)
func (wb *FoxESSEVC) logStopReason() {
	b, err := wb.conn.ReadHoldingRegisters(foxRegStopReason, 1)
	if err != nil {
		return
	}

	if reason := int(binary.BigEndian.Uint16(b)); reason < len(foxStopReasons) {
		wb.log.DEBUG.Printf("session finished: %s", foxStopReasons[reason])
	} else {
		wb.log.DEBUG.Printf("session finished: reason %d", reason)
	}
}

var _ api.StatusReasoner = (*FoxESSEVC)(nil)

// StatusReason implements the api.StatusReasoner interface.
// After the charger has finished the session (status 5) it rejects any restart
// command until the car has been unplugged, so surface that to the user instead
// of silently ignoring all control commands.
func (wb *FoxESSEVC) StatusReason() (api.Reason, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if wb.finished {
		return api.ReasonDisconnectRequired, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface.
// The spec recommends reading the EVC status register to verify start/stop (§2.41).
func (wb *FoxESSEVC) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return false, err
	}

	return foxEnabled(binary.BigEndian.Uint16(b)), nil
}

// Enable implements the api.Charger interface
func (wb *FoxESSEVC) Enable(enable bool) error {
	wb.mu.Lock()
	finished := wb.finished
	wb.mu.Unlock()

	// the charger refuses a restart command with a modbus exception once it has
	// finished the session; the car must disconnect before it will accept one again
	if enable && finished {
		return api.ErrNotAvailable
	}

	wb.mu.Lock()
	wb.enabled = enable
	wb.mu.Unlock()

	val := uint16(foxChargingStop)
	if enable {
		val = foxChargingStart
	}

	_, err := wb.conn.WriteSingleRegister(foxRegChargingControl, val)
	if err != nil {
		return err
	}

	wb.mu.Lock()
	wb.lastEnabled = enable
	cur := wb.current
	pbox := wb.pbox
	wb.mu.Unlock()

	// Push the cached setpoint right after the start command so a charger that is already
	// charging picks up the limit without delay. When the charger is still in start state
	// the write is rejected (§2.30/§2.31) - that is expected, the heartbeat retries once
	// charging has begun.
	if enable && cur != 0 {
		reg := uint16(foxRegMaxPower)
		if pbox {
			reg = foxRegMaxCurrent
		}
		if err := wb.writeReg(reg, cur); err != nil {
			wb.log.DEBUG.Printf("setpoint not accepted yet: %v", err)
		}
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *FoxESSEVC) MaxCurrent(current int64) error {
	return wb.maxCurrentMillis(float64(current))
}

// maxCurrentMillis implements the api.ChargerEx interface. It is only registered as such
// when a phase-cutting box is present, since the power setpoint used otherwise is limited
// to 0.1kW steps, which is coarser than one ampere.
func (wb *FoxESSEVC) maxCurrentMillis(current float64) error {
	if min, max := wb.limits(); current < min || current > max {
		return fmt.Errorf("invalid current %.1f", current)
	}

	var reg, val uint16

	if wb.pbox {
		// PBOX present: use current setpoint directly
		reg = foxRegMaxCurrent
		val = uint16(10 * current)
	} else {
		// No PBOX: convert to power setpoint so the charger decides phase count.
		// P = V * I * phases, scaled to 0.1kW units.
		phases := 1
		if wb.lp != nil {
			if p := wb.lp.GetPhases(); p != 0 {
				phases = p
			}
		}
		reg = foxRegMaxPower

		// The charger derives the phase count from the power setpoint (§2.38). Both of evcc's
		// minimum currents land just below the respective threshold (1p 6A = 1.38kW, 3p 6A =
		// 4.14kW), which would pause charging resp. silently keep the charger single-phase, so
		// round to the nearest step and lift the setpoint to the threshold of the requested mode.
		val = uint16(math.Round(voltage * current * float64(phases) / 100))
		if phases == 3 {
			val = max(val, foxPower3p)
		} else {
			val = max(val, foxPower1p)
		}

		// cache the phase count used for this setpoint so GetMaxCurrent can invert
		// the power register with the same value; lp.GetPhases() may have moved on
		// by the time GetMaxCurrent runs, which would otherwise yield a bogus result
		wb.mu.Lock()
		wb.currentPhases = phases
		wb.mu.Unlock()
	}

	// Always cache the setpoint so Enable(true) and the heartbeat can push it.
	wb.mu.Lock()
	wb.current = val
	wb.mu.Unlock()

	// The setpoint registers only take effect while charging (§2.30/§2.31) and are
	// rejected with a modbus exception in any other state, so defer the write to
	// Enable(true) resp. the heartbeat instead of failing here.
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return err
	}
	if !foxCharging(binary.BigEndian.Uint16(b)) {
		return nil
	}

	return wb.writeReg(reg, val)
}

var _ api.CurrentLimiter = (*FoxESSEVC)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *FoxESSEVC) GetMinMaxCurrent() (float64, float64, error) {
	if wb.minCurrent == 0 || wb.maxCurrent == 0 {
		return 0, 0, api.ErrNotAvailable
	}

	return wb.minCurrent, wb.maxCurrent, nil
}

// limits returns the supported current range, falling back to the values that
// are common to all models if the charger did not report its own (§2.20/§2.21)
func (wb *FoxESSEVC) limits() (float64, float64) {
	min, max, err := wb.GetMinMaxCurrent()
	if err != nil {
		return 6, 32
	}

	return min, max
}

var _ api.CurrentGetter = (*FoxESSEVC)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *FoxESSEVC) GetMaxCurrent() (float64, error) {
	// outside of charging the setpoint registers are restored to the device maximum
	// (§2.30/§2.31), so they don't reflect the requested limit
	s, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return 0, err
	}
	if !foxCharging(binary.BigEndian.Uint16(s)) {
		return 0, api.ErrNotAvailable
	}

	if wb.pbox {
		b, err := wb.conn.ReadHoldingRegisters(foxRegMaxCurrent, 1)
		if err != nil {
			return 0, err
		}
		return float64(binary.BigEndian.Uint16(b)) / 10, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(foxRegMaxPower, 1)
	if err != nil {
		return 0, err
	}

	wb.mu.Lock()
	phases := wb.currentPhases
	wb.mu.Unlock()
	if phases == 0 {
		phases = 1
	}

	// invert the MaxCurrentMillis formula: val (0.1kW) -> amps. Uses the phase
	// count cached at write time (see MaxCurrentMillis), not the loadpoint's
	// current phase count, since the charger may not have completed its
	// autonomous phase switch yet and lp.GetPhases() may have moved on.
	return float64(binary.BigEndian.Uint16(b)) * 100 / (voltage * float64(phases)), nil
}

var _ api.Meter = (*FoxESSEVC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *FoxESSEVC) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) * 100, nil
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
	// the phase cutting box only accepts the command while charging, and not within the
	// first minute after charging started (§2.42). Report the command as unavailable
	// rather than pretending success, so evcc keeps its own phase state untouched.
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return err
	}
	if !foxCharging(binary.BigEndian.Uint16(b)) {
		return api.ErrNotAvailable
	}

	// 0: three-phase, 1: single-phase (L2)
	var val uint16
	if phases == 1 {
		val = 1
	}

	// the switch takes effect asynchronously; getPhases reports the result and
	// evcc's charger sync corrects its phase state if the box did not follow
	_, err = wb.conn.WriteSingleRegister(foxRegPhaseSwitching, val)

	return err
}

// phases1p3pAuto implements the api.PhaseSwitcher interface as a no-op.
// Without a phase-cutting box the charger switches phases autonomously based
// on the power setpoint (spec §2.38); this only updates evcc's internal phase
// bookkeeping so pvScalePhases can compute correct 1p/3p current thresholds.
func (wb *FoxESSEVC) phases1p3pAuto(_ int) error {
	return nil
}

// getPhases implements the api.PhaseGetter interface
func (wb *FoxESSEVC) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegPhaseSequence, 1)
	if err != nil {
		return 0, err
	}

	// 0: three-phase output, 1: L2, 2: L3
	if binary.BigEndian.Uint16(b) == 0 {
		return 3, nil
	}

	return 1, nil
}

var _ loadpoint.Controller = (*FoxESSEVC)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *FoxESSEVC) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
