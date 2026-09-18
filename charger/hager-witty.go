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
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Hager witty plus (XVL122xx), Modbus TCP as used by the Hager XEM470 energy manager.
// Register map from https://github.com/evcc-io/evcc/issues/33327.
//
// The map is a user-contributed summary, no vendor document is publicly available. Assumptions
// that could not be verified against hardware or a second source are marked TODO below.

// HagerWitty charger implementation
type HagerWitty struct {
	implement.Caps
	log     *util.Logger
	conn    *modbus.Connection
	mu      sync.Mutex // guards the cached state
	current uint16     // last non-zero current setpoint in 0.1A
	enabled bool
	status  uint32 // last value of hagerRegStatus, only read in the numeric status path
	cpAscii bool   // hagerRegCpState holds two ASCII characters instead of a numeric code
}

// Register map. Addresses are used as documented, i.e. as Modbus PDU addresses.
// TODO the source does not state whether the documented addresses are 0-based PDU addresses or
// 1-based register numbers. The table is internally consistent, so a global offset is unlikely.
const (
	// device information
	hagerRegProduct   = 0x1003 // product reference, 16 registers, ASCII
	hagerRegSerial    = 0x1013 // unique id, 16 registers, ASCII
	hagerRegSwVersion = 0x1023 // software version, 3 registers, major.minor.revision.build

	// charging session
	hagerRegHwMaxCurrent  = 0x3006 // internal hardware current limit, 0-32A
	hagerRegPpMaxCurrent  = 0x3007 // cable capability (proximity pilot), A
	hagerRegCpState       = 0x3008 // CP state per IEC 61851, A1-F
	hagerRegPwm           = 0x3009 // control pilot duty cycle, %
	hagerRegCurrents      = 0x300A // current L1-L3, 3 registers, int16, 0.1A
	hagerRegVoltages      = 0x300D // voltage L1-L3, 3 registers, uint16, 0.1V
	hagerRegPowers        = 0x3010 // power L1-L3, 3 registers, int16, W
	hagerRegPower         = 0x3013 // total power, int32, W
	hagerRegMaxCurrent    = 0x301E // charging current setpoint, uint16, 0.1A (0-320)
	hagerRegFallbackCurr  = 0x301F // current setpoint applied on communication loss, 0.1A
	hagerRegAvailability  = 0x3020 // EVCS availability, 0=unavailable, 1=available
	hagerRegFallbackAvail = 0x3021 // availability applied on communication loss
	hagerRegSwitch3to1    = 0x3022 // 0=use three phases, 1=use single phase
	hagerRegLockState     = 0x3025 // cable lock, 0=unlocked, 1=locked, 2=tethered cable
	hagerRegFallbackDly   = 0x3026 // 0=fallback immediately, 1=fallback after 30s
	hagerRegStatus        = 0x302B // wallbox status and error code, uint32

	// HMI / RFID
	hagerRegRfidSize = 0x4001 // length of the RFID uid in bytes, 4, 7 or 10
	hagerRegRfidUid  = 0x4002 // RFID uid, 5 registers

	// energy
	hagerRegSessionEnergy   = 0x5000 // energy of the running session, uint32, Wh
	hagerRegSessionDuration = 0x500A // effective charging duration (CP state C), uint32, s
)

// Values of hagerRegStatus, taken as mainstate = status >> 8.
// TODO only IDLE and CHARGING are documented. The remaining values are derived from the Hager
// witty one BLE state machine (https://github.com/ngraziano/hass-witty), which uses the same
// encoding, and are unverified for the witty plus.
const (
	hagerStateIdle       = 1  // no vehicle connected
	hagerStateWait       = 2  // vehicle connected, waiting
	hagerStateWaitEnergy = 4  // waiting for authorization or energy release
	hagerStateCharging   = 6  // charging
	hagerStateFinished   = 8  // session finished
	hagerStateReserved   = 16 // reserved for a pending session

	hagerStateError = 0xF00000 // error, the low bits carry the error detail
)

const (
	hagerAvailable    = 1   // hagerRegAvailability: EVCS available
	hagerMinCurrent   = 60  // hagerRegMaxCurrent: lowest setpoint that starts charging, 6.0A
	hagerMaxCurrent   = 320 // hagerRegMaxCurrent: highest accepted setpoint, 32.0A
	hagerCurrentScale = 10  // hagerRegMaxCurrent and friends use 0.1A steps
)

func init() {
	registry.AddCtx("hager-witty", NewHagerWittyFromConfig)
}

// NewHagerWittyFromConfig creates a Hager witty plus charger from generic config
func NewHagerWittyFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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

	return NewHagerWitty(ctx, cc.TcpSettings)
}

// NewHagerWitty creates a Hager witty plus charger
func NewHagerWitty(ctx context.Context, settings modbus.TcpSettings) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("hager")
	conn.Logger(log.TRACE)

	wb := newHagerWitty(conn, log)

	if err := wb.initialize(); err != nil {
		return nil, err
	}

	return wb, nil
}

// newHagerWitty wires the struct without sponsor gate and device probe (also used by tests)
func newHagerWitty(conn *modbus.Connection, log *util.Logger) *HagerWitty {
	return &HagerWitty{
		Caps:    implement.New(),
		log:     log,
		conn:    conn,
		current: hagerMinCurrent, // assume min current
	}
}

// initialize probes the device and registers the conditional capabilities
func (wb *HagerWitty) initialize() error {
	// determine the CP state encoding once, it also serves as connection check
	b, err := wb.conn.ReadHoldingRegisters(hagerRegCpState, 1)
	if err != nil {
		return err
	}
	wb.cpAscii = hagerIsAscii(b)
	wb.log.DEBUG.Printf("cp state register %#04x decoded as %s", binary.BigEndian.Uint16(b),
		map[bool]string{true: "text", false: "number"}[wb.cpAscii])

	// the charger may have been taken out of service by another controller
	if _, err := wb.conn.WriteSingleRegister(hagerRegAvailability, hagerAvailable); err != nil {
		return fmt.Errorf("availability: %w", err)
	}

	// adopt the current setpoint, it survives an evcc restart
	setpoint, err := wb.readUint16(hagerRegMaxCurrent)
	if err != nil {
		return err
	}
	if setpoint > 0 {
		wb.current = setpoint
		wb.enabled = true
	}

	// phase switching is a witty plus feature, but the register may be missing on other models
	if _, err := wb.conn.ReadHoldingRegisters(hagerRegSwitch3to1, 1); err == nil {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))
	}

	implement.May(wb, implement.CurrentLimiter(wb.currentLimits()))

	// TODO the fallback registers (hagerRegFallbackCurr, hagerRegFallbackAvail, hagerRegFallbackDly)
	// are left untouched since neither the communication timeout nor whether petting it requires a
	// write is documented. The control loop polls often enough to keep the connection busy.

	return nil
}

// hagerIsAscii reports whether the register contains a printable CP state like "A1" or "C2"
// instead of a numeric code
func hagerIsAscii(b []byte) bool {
	return len(b) == 2 && b[0] >= 'A' && b[0] <= 'F'
}

func (wb *HagerWitty) readUint16(reg uint16) (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

func (wb *HagerWitty) readUint32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

// readString reads consecutive registers as ASCII text
func (wb *HagerWitty) readString(reg, words uint16) (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, words)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

// getPhaseValues returns 3 sequential register values scaled by divider
func (wb *HagerWitty) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(int16(binary.BigEndian.Uint16(b[2*i:]))) / divider
	}

	return res[0], res[1], res[2], nil
}

// setCurrent writes the charging current setpoint in 0.1A. Callers must hold mu.
func (wb *HagerWitty) setCurrent(current uint16) error {
	_, err := wb.conn.WriteSingleRegister(hagerRegMaxCurrent, current)

	return err
}

// Status implements the api.Charger interface
func (wb *HagerWitty) Status() (api.ChargeStatus, error) {
	if wb.cpAscii {
		s, err := wb.readString(hagerRegCpState, 1)
		if err != nil {
			return api.StatusNone, err
		}

		return api.ChargeStatusString(s)
	}

	// TODO fallback for firmware returning a numeric CP state. The mapping of the CP state
	// register itself is unknown, so the wallbox status register is used instead.
	status, err := wb.readUint32(hagerRegStatus)
	if err != nil {
		return api.StatusNone, err
	}

	wb.mu.Lock()
	wb.status = status
	wb.mu.Unlock()

	switch status >> 8 {
	case hagerStateIdle:
		return api.StatusA, nil

	case hagerStateWait, hagerStateWaitEnergy, hagerStateFinished, hagerStateReserved:
		return api.StatusB, nil

	case hagerStateCharging:
		return api.StatusC, nil

	case hagerStateError:
		return api.StatusNone, fmt.Errorf("charger error: %#08x", status)

	default:
		return api.StatusNone, fmt.Errorf("invalid status: %#08x", status)
	}
}

var _ api.StatusReasoner = (*HagerWitty)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *HagerWitty) StatusReason() (api.Reason, error) {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	// uses the status cached by Status(), which the loadpoint calls immediately before
	if !wb.cpAscii && wb.status>>8 == hagerStateWaitEnergy {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *HagerWitty) Enabled() (bool, error) {
	setpoint, err := wb.readUint16(hagerRegMaxCurrent)
	if err != nil {
		return false, err
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	wb.enabled = setpoint > 0
	if wb.enabled {
		wb.current = setpoint
	}

	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *HagerWitty) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	// TODO the wallbox is disabled by withdrawing the charging current. Whether writing
	// hagerRegAvailability would be the better stop mechanism is undocumented. For Hager's OCPP
	// implementation availability is known to be unreliable for starting and stopping.
	var current uint16
	if enable {
		current = wb.current
	}

	if err := wb.setCurrent(current); err != nil {
		return err
	}

	wb.enabled = enable

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *HagerWitty) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*HagerWitty)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *HagerWitty) MaxCurrentMillis(current float64) error {
	if current < hagerMinCurrent/hagerCurrentScale {
		return fmt.Errorf("invalid current: %.1fA", current)
	}

	setpoint := min(uint16(current*hagerCurrentScale), hagerMaxCurrent)

	wb.mu.Lock()
	defer wb.mu.Unlock()

	if wb.enabled {
		if err := wb.setCurrent(setpoint); err != nil {
			return err
		}
	}

	wb.current = setpoint

	return nil
}

var _ api.CurrentGetter = (*HagerWitty)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *HagerWitty) GetMaxCurrent() (float64, error) {
	setpoint, err := wb.readUint16(hagerRegMaxCurrent)
	if err != nil {
		return 0, err
	}

	return float64(setpoint) / hagerCurrentScale, nil
}

// currentLimits returns the api.CurrentLimiter implementation if the charger reports a plausible
// hardware limit, nil otherwise. The cable limit (hagerRegPpMaxCurrent) is not included since it
// is only meaningful while a cable is attached.
func (wb *HagerWitty) currentLimits() func() (float64, float64, error) {
	hw, err := wb.readUint16(hagerRegHwMaxCurrent)
	if err != nil {
		wb.log.DEBUG.Printf("hardware current limit: %v", err)
		return nil
	}

	// TODO the source documents the hardware limit as "0-32A" while the setpoint uses 0.1A steps.
	// Values above 32 are therefore assumed to use 0.1A steps, too.
	if hw > hagerMaxCurrent/hagerCurrentScale {
		hw /= hagerCurrentScale
	}

	if hw < hagerMinCurrent/hagerCurrentScale || hw > hagerMaxCurrent/hagerCurrentScale {
		wb.log.DEBUG.Printf("implausible hardware current limit: %dA", hw)
		return nil
	}

	return func() (float64, float64, error) {
		return hagerMinCurrent / hagerCurrentScale, float64(hw), nil
	}
}

var _ api.Meter = (*HagerWitty)(nil)

// CurrentPower implements the api.Meter interface
func (wb *HagerWitty) CurrentPower() (float64, error) {
	power, err := wb.readUint32(hagerRegPower)
	if err != nil {
		return 0, err
	}

	return float64(int32(power)), nil
}

var _ api.ChargeRater = (*HagerWitty)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *HagerWitty) ChargedEnergy() (float64, error) {
	energy, err := wb.readUint32(hagerRegSessionEnergy)
	if err != nil {
		return 0, err
	}

	return float64(energy) / 1e3, nil
}

// TODO no lifetime energy register is documented, hence no api.MeterEnergy

var _ api.ChargeTimer = (*HagerWitty)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *HagerWitty) ChargeDuration() (time.Duration, error) {
	secs, err := wb.readUint32(hagerRegSessionDuration)
	if err != nil {
		return 0, err
	}

	return time.Duration(secs) * time.Second, nil
}

var _ api.PhaseCurrents = (*HagerWitty)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *HagerWitty) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(hagerRegCurrents, 10)
}

var _ api.PhaseVoltages = (*HagerWitty)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *HagerWitty) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(hagerRegVoltages, 10)
}

var _ api.PhasePowers = (*HagerWitty)(nil)

// Powers implements the api.PhasePowers interface
func (wb *HagerWitty) Powers() (float64, float64, float64, error) {
	return wb.getPhaseValues(hagerRegPowers, 1)
}

var _ api.Identifier = (*HagerWitty)(nil)

// Identify implements the api.Identifier interface
func (wb *HagerWitty) Identify() ([]string, error) {
	size, err := wb.readUint16(hagerRegRfidSize)
	if err != nil {
		return nil, err
	}

	if size == 0 || size > 10 {
		return nil, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(hagerRegRfidUid, 5)
	if err != nil {
		return nil, err
	}

	return []string{fmt.Sprintf("%X", b[:size])}, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *HagerWitty) phases1p3p(phases int) error {
	// TODO it is undocumented whether the charger requires a charging pause before switching
	_, err := wb.conn.WriteSingleRegister(hagerRegSwitch3to1, map[int]uint16{1: 1, 3: 0}[phases])

	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *HagerWitty) getPhases() (int, error) {
	single, err := wb.readUint16(hagerRegSwitch3to1)
	if err != nil {
		return 0, err
	}

	if single == 1 {
		return 1, nil
	}

	return 3, nil
}

var _ api.Diagnosis = (*HagerWitty)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *HagerWitty) Diagnose() {
	if s, err := wb.readString(hagerRegProduct, 16); err == nil {
		fmt.Printf("\tProduct:\t%s\n", s)
	}
	if s, err := wb.readString(hagerRegSerial, 16); err == nil {
		fmt.Printf("\tSerial:\t%s\n", s)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hagerRegSwVersion, 3); err == nil {
		fmt.Printf("\tSoftware:\t%d.%d.%d.%d\n", b[0], b[1], b[2], b[3])
	}
	if b, err := wb.conn.ReadHoldingRegisters(hagerRegCpState, 1); err == nil {
		fmt.Printf("\tCP state:\t%s (%#04x)\n", bytesAsString(b), binary.BigEndian.Uint16(b))
	}
	if val, err := wb.readUint32(hagerRegStatus); err == nil {
		fmt.Printf("\tStatus:\t%#08x\n", val)
	}
	if val, err := wb.readUint16(hagerRegPwm); err == nil {
		fmt.Printf("\tPWM:\t%d%%\n", val)
	}
	if val, err := wb.readUint16(hagerRegHwMaxCurrent); err == nil {
		fmt.Printf("\tHardware limit:\t%d\n", val)
	}
	if val, err := wb.readUint16(hagerRegPpMaxCurrent); err == nil {
		fmt.Printf("\tCable limit:\t%d\n", val)
	}
	if val, err := wb.readUint16(hagerRegLockState); err == nil {
		fmt.Printf("\tCable lock:\t%d\n", val)
	}
	if val, err := wb.readUint16(hagerRegAvailability); err == nil {
		fmt.Printf("\tAvailability:\t%d\n", val)
	}
	if val, err := wb.readUint16(hagerRegSwitch3to1); err == nil {
		fmt.Printf("\tSingle phase:\t%d\n", val)
	}
	if val, err := wb.readUint16(hagerRegFallbackCurr); err == nil {
		fmt.Printf("\tFallback current:\t%.1fA\n", float64(val)/hagerCurrentScale)
	}
	if val, err := wb.readUint16(hagerRegFallbackAvail); err == nil {
		fmt.Printf("\tFallback availability:\t%d\n", val)
	}
	if val, err := wb.readUint16(hagerRegFallbackDly); err == nil {
		fmt.Printf("\tFallback delay:\t%d\n", val)
	}
}
