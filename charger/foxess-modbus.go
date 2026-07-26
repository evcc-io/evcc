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
	"fmt"
	"math"
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
	log        *util.Logger
	conn       *modbus.Connection
	mu         sync.Mutex
	current    float64 // tracks phase current, 0 if unset
	enabled    bool    // tracks enabled state
	session    bool    // tracks session state
	phases     int     // tracks phase count; the charger does not report it
	minPower   uint16  // min supported power, 0 if unknown
	maxPower   uint16  // max supported power, 0 if unknown
	minCurrent float64 // min supported current per phase, 0 if unknown
	maxCurrent float64 // max supported current per phase, 0 if unknown
	lp         loadpoint.API
}

const (
	// read-only registers (0x03)
	foxRegSwVersion     = 0x1001 // software version, byte1 major / byte0 minor
	foxRegStopReason    = 0x1002 // reason the last charging session ended, see spec appendix 1
	foxRegStatus        = 0x1003 // EVC status
	foxCpStatus         = 0x1004 // CP status
	foxCableStatus      = 0x1005 // Cable status
	foxRegVoltages      = 0x1008 // A/B/C phase voltage, 3 registers, 0.1V
	foxRegCurrents      = 0x100B // A/B/C phase current, 3 registers, 0.1A
	foxRegPower         = 0x100E // active power, 0.1kW
	foxRegPhaseSequence = 0x1010 // current phase sequence
	foxRegMaxSupPower   = 0x1011 // max supported power, 0.1kW (§2.18)
	foxRegMinSupPower   = 0x1012 // min supported power, 0.1kW (§2.19)
	foxRegMaxSupCurrent = 0x1013 // max supported current per phase, 0.1A (§2.20)
	foxRegMinSupCurrent = 0x1014 // min supported current per phase, 0.1A (§2.21)
	foxRegAlarm         = 0x1015 // system alarm, bit-coded, see spec appendix 3
	foxRegTotalEnergy   = 0x1016 // total energy, uint32, 0.1kWh; never resets
	foxRegSessionEnergy = 0x1018 // session energy, uint32, 0.1kWh; resets on session start
	foxRegFault         = 0x101A // system fault, uint32, bit-coded, see spec appendix 2
	foxRegRFID          = 0x101C // last RFID card, uint32

	// read/write registers (write with 0x10)
	foxRegWorkMode        = 0x3000 // work mode
	foxRegMaxCurrent      = 0x3001 // max charging current, 0.1A
	foxRegMaxPower        = 0x3002 // max charging power, 0.1kW
	foxRegTimeValidity    = 0x3005 // command validity window, seconds
	foxRegDefaultCurrent  = 0x3006 // fallback current when the EMS connection is lost, 0.1A
	foxRegAutoPhaseSwitch = 0x300A // single/three-phase automatic switching
	foxRegSwitchInterval  = 0x300B // min interval between phase switches, minutes

	// write-only registers (write with 0x06)
	foxRegSessionControl = 0x4001 // start/stop session

	foxSessionNoAction   = 0
	foxSessionStart      = 1
	foxSessionStop       = 2
	foxTimeValidity      = 60 // maximum command validity window in seconds (§2.34: 10-60s)
	foxDefaultCurrent    = 60 // 6.0A fallback current on EMS loss (§2.35: 6-32A)
	foxMinSwitchInterval = 5  // minimum phase switching interval in minutes (§2.39: 5-30min)

	// Without a phase-cutting box the charger derives the phase count from the power setpoint
	// (§2.38): >= 4.2kW three-phase, >= 1.4kW single-phase, below that charging is paused.
	// Setpoints are given in 0.1kW.
	foxMinPower3p = 42 // 1.4kW, the minimum power setpoint for a 3p charger
	foxMinPower1p = 14 // 1.4kW, the minimum power setpoint for a 1p or switchable charger
	foxMaxPower1p = 73 // 7.3kW, the maximum power setpoint for a 1p charger

	// foxHeartbeatInterval is the interval at which the heartbeat runs.
	// Must be less than foxTimeValidity so the charger never considers evcc offline.
	foxHeartbeatInterval = 4 * time.Second
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
	foxStatusLocked    = 8 // locked, no operations possible
	foxStatusSwitching = 9 // undocumented: automatic phase switch in progress
)

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
		Caps:   implement.New(),
		log:    log,
		conn:   conn,
		phases: 3,
	}

	// device limits are model-specific and constant, so read them once (§2.20/§2.21)
	if b, err := wb.conn.ReadHoldingRegisters(foxRegMinSupCurrent, 1); err == nil {
		wb.minCurrent = float64(binary.BigEndian.Uint16(b)) / 10
	}
	if b, err := wb.conn.ReadHoldingRegisters(foxRegMaxSupCurrent, 1); err == nil {
		wb.maxCurrent = float64(binary.BigEndian.Uint16(b)) / 10
	}

	if b, err := wb.conn.ReadHoldingRegisters(foxRegMinSupPower, 1); err == nil {
		wb.minPower = binary.BigEndian.Uint16(b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(foxRegMaxSupPower, 1); err == nil {
		wb.maxPower = binary.BigEndian.Uint16(b)
	}

	is3pCharger := wb.maxPower > foxMaxPower1p
	if !is3pCharger {
		wb.phases = 1
	}

	var hasAutoSwEn bool
	if b, err := wb.conn.ReadHoldingRegisters(foxRegAutoPhaseSwitch, 1); err == nil {
		hasAutoSwEn = binary.BigEndian.Uint16(b) > 0
	}

	if is3pCharger && hasAutoSwEn {
		// 3p charger with enabled auto phase-switching
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))

		// keep the interal charge pause and switching protection interval as short as possible
		if err := wb.ensureReg(foxRegSwitchInterval, foxMinSwitchInterval); err != nil {
			wb.log.WARN.Printf("switch interval: %v", err)
		}
	}

	// get inital status to initialize session and enabled state
	if _, err = wb.Status(); err != nil {
		return nil, err
	}

	// the charger goes to fallback current if no setpoint is received within the
	// time validity window, so we must re-send the setpoint periodically
	go wb.heartbeat(ctx)

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

func (wb *FoxESSEVC) calcPower(enabled bool, current float64, phases int) float64 {
	if !enabled {
		return 0
	}

	return 230 * float64(phases) * min(max(current, wb.minCurrent), wb.maxCurrent)
}

// applySetpoint writes the combined enable state and charging limit during an active session.
// Outside of a session the charger ignores the setpoint.
func (wb *FoxESSEVC) applySetpoint(power float64) error {
	var val uint16

	// if !wb.session {
	// 	return errors.New("no active session")
	// }

	if power != 0 {
		val = min(max(uint16(math.Round(power/100.0)), wb.minPower), wb.maxPower)
	}

	return wb.ensureReg(foxRegMaxPower, val)
}

// heartbeat keeps the charger from considering evcc offline
func (wb *FoxESSEVC) heartbeat(ctx context.Context) {
	for tick := time.Tick(foxHeartbeatInterval); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		// keepalive write, no-op
		_, err := wb.conn.ReadHoldingRegisters(foxRegSessionControl, 0)
		if err != nil {
			wb.log.DEBUG.Printf("heartbeat: %v", err)
		}
	}
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
		wb.session = false
		return api.StatusA, nil

	case foxStatusConnect, foxStatusFinish:
		wb.session = false
		return api.StatusB, nil

	case foxStatusStart:
		wb.session = true
		return api.StatusB, nil

	case foxStatusPause:
		// wb.enabled = false
		wb.session = true
		return api.StatusB, nil

	case foxStatusSwitching:
		wb.enabled = true
		wb.session = true
		return api.StatusB, nil

	case foxStatusCharging:
		wb.enabled = true
		wb.session = true
		return api.StatusC, nil

	default:
		wb.session = false
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

var _ api.StatusReasoner = (*FoxESSEVC)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *FoxESSEVC) StatusReason() (api.Reason, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return api.ReasonUnknown, nil
	}

	switch s := binary.BigEndian.Uint16(b); s {
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
	val, err := wb.readUint16(foxRegMaxPower)
	if err != nil {
		return false, err
	}

	wb.mu.Lock()
	if val == 0 {
		wb.enabled = false
	}
	enabled := wb.enabled
	wb.mu.Unlock()

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *FoxESSEVC) Enable(enable bool) error {
	err := wb.applySetpoint(wb.calcPower(enable, wb.current, wb.phases))
	if err != nil {
		return err
	}

	wb.mu.Lock()
	wb.enabled = enable
	wb.mu.Unlock()

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *FoxESSEVC) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*FoxESSEVC)(nil)

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *FoxESSEVC) MaxCurrentMillis(current float64) error {
	if current < wb.minCurrent {
		return fmt.Errorf("invalid current: %.1fA", current)
	}

	err := wb.applySetpoint(wb.calcPower(wb.enabled, current, wb.phases))
	if err != nil {
		return err
	}

	wb.mu.Lock()
	wb.current = current
	wb.mu.Unlock()

	return nil
}

var _ api.CurrentLimiter = (*FoxESSEVC)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *FoxESSEVC) GetMinMaxCurrent() (float64, float64, error) {
	if wb.minCurrent == 0 || wb.maxCurrent == 0 {
		return 0, 0, api.ErrNotAvailable
	}

	return wb.minCurrent, wb.maxCurrent, nil
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

var _ api.ChargeRater = (*FoxESSEVC)(nil)

func (wb *FoxESSEVC) ChargedEnergy() (float64, error) {
	energy, err := wb.readUint32(foxRegSessionEnergy)
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
	wb.mu.Lock()
	wb.phases = phases
	wb.mu.Unlock()

	return nil
}

// getPhases implements the api.PhaseGetter interface
func (wb *FoxESSEVC) getPhases() (int, error) {
	val, err := wb.readUint16(foxRegMaxPower)
	if err != nil {
		return 0, err
	}

	wb.mu.Lock()
	phases := wb.phases
	wb.mu.Unlock()

	if val >= foxMinPower3p {
		phases = 3
	} else if val >= foxMinPower1p {
		phases = 1
	}

	return phases, nil
}
