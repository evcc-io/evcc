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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Voltie charger implementation
// https://voltie.eu
// Modbus API documentation v1.1
//
// Modbus TCP is supported from EVSE firmware 350 and charger software 1.3.40.
// Earlier firmware answers out-of-range reads with adjacent memory instead of
// an exception and silently accepts ineffective FC16 writes. The driver checks
// the firmware build number on startup; the charger software version is not
// exposed over Modbus and has to be checked in the Voltie app.
//
// The same register map is served over RS-485 (Modbus RTU) and over the Modbus
// TCP gateway. The gateway is a transparent bridge to the charger's MCU: it
// forwards one request at a time, at most two transactions per second, and
// abandons a request that is unanswered after 3s. The driver therefore fetches
// the register blocks through the shared bulk read cache instead of issuing a
// separate request per value, so an update cycle costs one request per block.
//
// Only function code 0x06 (write single register) is accepted for writes;
// 0x10 (write multiple) is rejected with exception 0x01.
//
// Firmware 352 extends both register blocks with phase switching, the
// communication timeout, the hardware current limit, lifetime energy and the
// phase powers. Those capabilities are only offered when the charger reports
// that firmware, and the block read lengths follow the firmware as well.

const (
	// register blocks fetched in bulk
	voltieRegInfoBlock   = 0x0000
	voltieLenInfoBlock   = 10 // 0x0000..0x0009
	voltieRegStatusBlock = 0x000A
	voltieLenStatusBlock = 12 // 0x000A..0x0015
	voltieRegMeterBlock  = 0x2000
	voltieLenMeterBlock  = 22 // 0x2000..0x2015

	// firmware 352 extends both blocks
	voltieLenStatusBlockExt = 15 // 0x000A..0x0018
	voltieLenMeterBlockExt  = 30 // 0x2000..0x201D

	// identification block. The 64 bit serial numbers are sent
	// least-significant word first, unlike the metering values.
	voltieRegChargerID   = 0x0000 // INT16 Voltie charger ID
	voltieRegFirmware    = 0x0001 // INT16 EVSE firmware build number
	voltieRegMcuSerial   = 0x0002 // INT64 MCU serial number
	voltieRegHpowSerial  = 0x0006 // INT64 power board serial number
	voltieSerialRegCount = 4

	// status block
	voltieRegStatus       = 0x000A // INT16 EVSE_STATE
	voltieRegAutoStart    = 0x000B // INT16 auto start enabled
	voltieRegChargeEnable = 0x000C // INT16 charging enabled
	voltieRegCharging     = 0x000D // INT16 charging
	// 0x000E counts the phases with mains voltage present on the input, not the
	// phases the vehicle charges on: on a three-phase supply it stays 3 even in
	// forced single phase mode, so it can only answer api.PhaseGetter once the
	// forced single phase register has been checked
	voltieRegPhases       = 0x000E // INT16 phases with mains voltage present
	voltieRegDlmSet       = 0x000F // INT16 stored DLM mode
	voltieRegStopReason   = 0x0012 // INT16 charge stop reason
	voltieRegCurrent      = 0x0014 // INT16 software current limit [mA]
	voltieRegDlmEffective = 0x0015 // INT16 effective DLM mode

	// status block, firmware 352
	voltieRegSinglePhase  = 0x0016 // INT16 forced single phase
	voltieRegQueryTimeout = 0x0017 // INT16 communication timeout [s]
	voltieRegMaxCurrent   = 0x0018 // INT16 hardware current limit [mA]

	// meter block
	voltieRegVoltages = 0x2000 // 3x INT32 phase voltage [mV]
	voltieRegCurrents = 0x2006 // 3x INT32 phase charging current [mA]
	voltieRegDuration = 0x200C // INT32 charge duration [s]
	voltieRegEnergy   = 0x200E // INT32 charged energy in session [Ws]
	voltieRegPower    = 0x2010 // INT32 charging power [W]
	voltieRegCapacity = 0x2012 // INT32 instantaneous current capacity [mA]

	// meter block, firmware 352
	voltieRegTotalEnergy = 0x2016 // INT32 lifetime energy [Wh]
	voltieRegPowers      = 0x2018 // 3x INT32 phase power [W]

	// the charger's ampacity range [A]. The current limit register is typed
	// INT16, so the milliampere value must stay below the sign boundary.
	voltieMinCurrent = 6
	voltieMaxCurrent = 32

	// firmware that fixes the Modbus slave address checks and rejects FC16
	voltieMinFirmware = 350

	// firmware that adds phase switching, the communication timeout, the
	// hardware current limit, lifetime energy and the phase powers
	voltieExtFirmware = 352

	// the L2/L3 contactor must not be switched under load, so the firmware
	// rejects a phase switch while charging and holds the contactor for up to
	// 500ms of arc suppression after charging is disabled
	voltiePhaseSwitchWait  = 500 * time.Millisecond
	voltiePhaseSwitchTries = 6

	// evcc's default update interval, used to judge whether the charger's
	// communication timeout would cut the current between polls. The driver
	// cannot read the configured interval.
	voltieDefaultInterval = 30

	// the charger's documented default slave address
	voltieDefaultSlaveID = 11

	// the gateway abandons a forwarded request after 3s, so the client must
	// wait longer than that to receive the resulting exception
	voltieTimeout = 5 * time.Second

	// the gateway forwards at most two transactions per second
	voltieDelay = 500 * time.Millisecond
)

// EVSE states. The names follow the charger's own error-state documentation, so a
// user sees the same wording in evcc as on the charger display and in its app.
const (
	voltieStateA = 0x01 // not connected
	voltieStateB = 0x02 // connected, ready
	voltieStateC = 0x03 // charging
)

var voltieStates = map[uint16]string{
	0x00: "state not yet determined",
	0x01: "vehicle state A, not connected",
	0x02: "vehicle state B, connected",
	0x03: "vehicle state C, charging",
	0x04: "vehicle state D, charging with ventilation",
	0x05: "control signal (CP)",
	0x06: "residual current detected",
	0x07: "no grounding",
	0x08: "stuck relay",
	0x09: "residual current sensor test failed",
	0x0A: "over temperature",
	0x0B: "over current",
	0x0C: "I²C bus fault",
	0x0D: "vehicle fault (state E)",
	0x0E: "over humidity",
	0x0F: "phase misconnected",
	0x10: "overvoltage",
	0x11: "undervoltage on AC supply",
	0x12: "charger disabled, not functioning",
	0x13: "booting",
	0x15: "unknown power board",
	0x18: "state undetermined",
	0x19: "uploading VoltieMeter firmware",
}

// charge stop reasons reported by the MCU, see the "EVSE charge stop reasons"
// chapter. Reasons 23..31 originate in the charger's control software and are
// not reported through Modbus.
var voltieStopReasons = map[uint16]string{
	0:   "none, charging in progress",
	1:   "unspecified reason",
	2:   "preset charge duration reached",
	3:   "preset energy amount charged",
	4:   "stopped by the user",
	5:   "error: residual current detected",
	6:   "charger disabled, out of order",
	7:   "firmware restart",
	8:   "charger in sleep mode, out of order",
	9:   "error: no voltage on the output (ground continuity or relay error)",
	10:  "vehicle disconnected",
	11:  "vehicle not accepting charge",
	12:  "error: I²C bus fault",
	13:  "error: residual current sensor test failed",
	14:  "error: over temperature",
	15:  "error: control signal (CP)",
	17:  "error: stuck relay",
	18:  "error: over current",
	21:  "error: over humidity",
	22:  "error: phase misconnected",
	100: "not enough free building current available (dynamic load management)",
	101: "not enough solar current available (eco/green mode)",
	102: "grid voltage is not high enough (grid-controlled mode)",
	103: "charge current limit set to zero",
	105: "error: overvoltage",
	106: "error: vehicle fault",
	107: "error: undervoltage on AC supply",
	108: "vehicle in state D while state D is disabled",
	109: "error: unknown power board",
}

// Voltie is an api.Charger implementation for Voltie wallboxes
type Voltie struct {
	implement.Caps
	conn   *modbus.Connection
	log    *util.Logger
	cache  *modbus.Cache
	status modbus.Block
	meter  modbus.Block
	info   modbus.Block
	ext    bool // firmware provides the extended register blocks
}

// read fetches a register block through the shared bulk read cache, so all
// values taken from the same block within a poll cycle cost one request
func (wb *Voltie) read(block modbus.Block) ([]byte, error) {
	key := fmt.Sprintf("%s/holding/%d/%d", wb.conn.Addr(), block.Register, block.Count)

	payload, _, err := wb.cache.Fetch(key, func() ([]byte, error) {
		return wb.conn.ReadHoldingRegisters(block.Register, block.Count)
	})

	return payload, err
}

// voltieSerial decodes a 64 bit serial number, which is sent least-significant
// word first unlike the 32 bit metering values
func voltieSerial(b []byte, off int) uint64 {
	var res uint64
	for i := range voltieSerialRegCount {
		res |= uint64(binary.BigEndian.Uint16(b[off+2*i:])) << (16 * i)
	}

	return res
}

// voltieU16 returns the register at addr within a cached block payload
func voltieU16(block modbus.Block, b []byte, addr uint16) uint16 {
	return binary.BigEndian.Uint16(b[block.ByteOffset(addr):])
}

// voltieU32 returns the 32 bit value at addr within a cached block payload,
// most-significant word first
func voltieU32(block modbus.Block, b []byte, addr uint16) uint32 {
	return binary.BigEndian.Uint32(b[block.ByteOffset(addr):])
}

func init() {
	registry.AddCtx("voltie", NewVoltieFromConfig)
}

// NewVoltieFromConfig creates a Voltie charger from generic config
func NewVoltieFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Cache              time.Duration
	}{
		TcpSettings: modbus.TcpSettings{
			ID:      voltieDefaultSlaveID,
			Timeout: voltieTimeout,
			Delay:   voltieDelay,
		},
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVoltie(ctx, cc.TcpSettings, cc.Cache)
}

// NewVoltie creates a Voltie charger
func NewVoltie(ctx context.Context, settings modbus.TcpSettings, cache time.Duration) (*Voltie, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("voltie")
	conn.Logger(log.TRACE)

	wb := &Voltie{
		Caps:   implement.New(),
		conn:   conn,
		log:    log,
		cache:  modbus.NewCache(cache),
		info:   modbus.Block{Register: voltieRegInfoBlock, Count: voltieLenInfoBlock},
		status: modbus.Block{Register: voltieRegStatusBlock, Count: voltieLenStatusBlock},
		meter:  modbus.Block{Register: voltieRegMeterBlock, Count: voltieLenMeterBlock},
	}

	// the register blocks grew with firmware 352, so the read lengths follow the
	// firmware: reading past the end of a block yields exception 0x02
	if b, err := wb.read(wb.info); err == nil {
		fw := voltieU16(wb.info, b, voltieRegFirmware)

		switch {
		case fw < voltieMinFirmware:
			log.WARN.Printf("firmware %d is outdated, Modbus TCP requires %d or later", fw, voltieMinFirmware)
		case fw >= voltieExtFirmware:
			wb.ext = true
			wb.status.Count = voltieLenStatusBlockExt
			wb.meter.Count = voltieLenMeterBlockExt
		default:
			log.DEBUG.Printf("firmware %d predates %d, phase switching, lifetime energy, phase powers and the current limits are unavailable", fw, voltieExtFirmware)
		}
	}

	if err := wb.checkSettings(); err != nil {
		return nil, err
	}

	if wb.ext {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))
		implement.Has(wb, implement.MeterEnergy(wb.totalEnergy))
		implement.Has(wb, implement.PhasePowers(wb.powers))
		implement.Has(wb, implement.CurrentLimiter(wb.getMinMaxCurrent))
	}

	return wb, nil
}

// checkSettings inspects the charger's settings once on startup. The charger
// must not start a session on its own while evcc is in control, and its own
// load management would silently cap the current requested by evcc.
func (wb *Voltie) checkSettings() error {
	b, err := wb.read(wb.status)
	if err != nil {
		return err
	}

	if dlm := voltieU16(wb.status, b, voltieRegDlmEffective); dlm != 0 {
		wb.log.WARN.Printf("charger-side load management is active (mode %d) and will cap the requested current", dlm)
	}

	// the charger cuts the current to 0 A when Modbus communication stops for
	// longer than its timeout. 0 and 255 disable the watchdog.
	if wb.ext {
		if to := voltieU16(wb.status, b, voltieRegQueryTimeout); to > 0 && to < 255 && to <= voltieDefaultInterval {
			wb.log.WARN.Printf("the charger reduces the current to 0 A after %ds without Modbus communication; set the timeout to 0 in the Voltie app or above the evcc update interval", to)
		}
	}

	// the auto start setting is persisted in the charger's EEPROM and stays off
	// after evcc is removed, so it is only written when actually enabled
	if voltieU16(wb.status, b, voltieRegAutoStart) == 0 {
		return nil
	}

	if _, err := wb.conn.WriteSingleRegister(voltieRegAutoStart, 0); err != nil {
		return fmt.Errorf("disable auto start: %w (is Modbus control enabled on the charger?)", err)
	}

	wb.cache.Clear()
	wb.log.WARN.Println("auto start disabled, the setting is persistent and must be restored in the Voltie app when evcc is removed")

	return nil
}

// Status implements the api.Charger interface
func (wb *Voltie) Status() (api.ChargeStatus, error) {
	b, err := wb.read(wb.status)
	if err != nil {
		return api.StatusNone, err
	}

	switch state := voltieU16(wb.status, b, voltieRegStatus); state {
	case voltieStateA:
		return api.StatusA, nil
	case voltieStateB:
		return api.StatusB, nil
	case voltieStateC:
		return api.StatusC, nil
	default:
		// any other state, including D where the vehicle requires ventilation,
		// is reported as an error together with the MCU's stop reason
		desc, ok := voltieStates[state]
		if !ok {
			desc = "unknown state"
		}

		if reason := voltieU16(wb.status, b, voltieRegStopReason); reason != 0 {
			if txt, ok := voltieStopReasons[reason]; ok {
				return api.StatusNone, fmt.Errorf("%s (0x%02X): %s", desc, state, txt)
			}
			return api.StatusNone, fmt.Errorf("%s (0x%02X): stop reason %d", desc, state, reason)
		}

		return api.StatusNone, fmt.Errorf("%s (0x%02X)", desc, state)
	}
}

// Enabled implements the api.Charger interface
func (wb *Voltie) Enabled() (bool, error) {
	b, err := wb.read(wb.status)
	if err != nil {
		return false, err
	}

	return voltieU16(wb.status, b, voltieRegChargeEnable) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Voltie) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(voltieRegChargeEnable, u)
	if err == nil {
		wb.cache.Clear()
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Voltie) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Voltie)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Voltie) MaxCurrentMillis(current float64) error {
	if current < voltieMinCurrent || current > voltieMaxCurrent {
		return fmt.Errorf("invalid current %.1f", current)
	}

	_, err := wb.conn.WriteSingleRegister(voltieRegCurrent, uint16(current*1e3))
	if err == nil {
		wb.cache.Clear()
	}

	return err
}

var _ api.CurrentGetter = (*Voltie)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *Voltie) GetMaxCurrent() (float64, error) {
	b, err := wb.read(wb.status)
	if err != nil {
		return 0, err
	}

	return float64(voltieU16(wb.status, b, voltieRegCurrent)) / 1e3, nil
}

var _ api.Meter = (*Voltie)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Voltie) CurrentPower() (float64, error) {
	b, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return float64(int32(voltieU32(wb.meter, b, voltieRegPower))), nil
}

var _ api.ChargeRater = (*Voltie)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Voltie) ChargedEnergy() (float64, error) {
	b, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return float64(voltieU32(wb.meter, b, voltieRegEnergy)) / 3.6e6, nil // Ws to kWh
}

var _ api.ChargeTimer = (*Voltie)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Voltie) ChargeDuration() (time.Duration, error) {
	b, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return time.Duration(voltieU32(wb.meter, b, voltieRegDuration)) * time.Second, nil
}

// getPhaseValues returns 3 sequential 32 bit values from the meter block, divided
// by divisor. The voltages and currents are in milli units, the powers in watts.
func (wb *Voltie) getPhaseValues(reg uint16, divisor float64) (float64, float64, float64, error) {
	b, err := wb.read(wb.meter)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(voltieU32(wb.meter, b, reg+uint16(2*i))) / divisor
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Voltie)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Voltie) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(voltieRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*Voltie)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Voltie) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(voltieRegVoltages, 1e3)
}

// phases1p3p implements the api.PhaseSwitcher interface. The register is only
// writable on chargers manufactured from 2026, which carry an HPOW108 power
// board; other hardware rejects the write with exception 0x03, as does a relay
// or EEPROM error. The hardware generation cannot be detected up front: the
// serial numbers exposed over Modbus are the MCU and power board serials, not
// the charger serial that carries the generation prefix.
func (wb *Voltie) phases1p3p(phases int) error {
	if phases != 1 && phases != 3 {
		return fmt.Errorf("invalid phases: %d", phases)
	}

	b, err := wb.read(wb.status)
	if err != nil {
		return err
	}

	// the L2/L3 contactor must not be switched under load, so charging is paused
	// and the contactor given time to open before the switch
	resume := voltieU16(wb.status, b, voltieRegChargeEnable) != 0
	if resume {
		if err := wb.Enable(false); err != nil {
			return fmt.Errorf("pause before phase switch: %w", err)
		}
	}

	err = wb.awaitContactorOpen()
	if err == nil {
		var u uint16
		if phases == 1 {
			u = 1
		}

		if _, err = wb.conn.WriteSingleRegister(voltieRegSinglePhase, u); err != nil {
			err = fmt.Errorf("switch phases: %w", err)
		}

		wb.cache.Clear()
	}

	// restore the previous state even when the switch failed
	if resume {
		err = errors.Join(err, wb.Enable(true))
	}

	return err
}

// awaitContactorOpen waits until the charger reports that charging has stopped,
// which is the condition the firmware checks before allowing a phase switch
func (wb *Voltie) awaitContactorOpen() error {
	for i := range voltiePhaseSwitchTries {
		if i > 0 {
			time.Sleep(voltiePhaseSwitchWait)
		}

		wb.cache.Clear()

		b, err := wb.read(wb.status)
		if err != nil {
			return err
		}

		if voltieU16(wb.status, b, voltieRegCharging) == 0 {
			return nil
		}
	}

	return errors.New("charging did not stop, cannot switch phases")
}

// getPhases implements the api.PhaseGetter interface, reporting the phases the
// vehicle can charge on. The forced single phase register decides first: the
// phase register 0x000E cannot answer on its own because it counts the phases
// with mains voltage on the input, so on a three-phase supply it stays 3 even
// while the L2/L3 relay is open. It is the better answer in the normal case
// though, where it keeps a charger wired to a single-phase supply from being
// reported as three-phase.
func (wb *Voltie) getPhases() (int, error) {
	b, err := wb.read(wb.status)
	if err != nil {
		return 0, err
	}

	if voltieU16(wb.status, b, voltieRegSinglePhase) != 0 {
		return 1, nil
	}

	if phases := int(voltieU16(wb.status, b, voltieRegPhases)); phases >= 1 && phases <= 3 {
		return phases, nil
	}

	return 3, nil
}

// totalEnergy implements the api.MeterEnergy interface. The counter is updated
// when the session ends, so it does not move while charging.
func (wb *Voltie) totalEnergy() (float64, error) {
	b, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return float64(voltieU32(wb.meter, b, voltieRegTotalEnergy)) / 1e3, nil // Wh to kWh
}

// powers implements the api.PhasePowers interface
func (wb *Voltie) powers() (float64, float64, float64, error) {
	return wb.getPhaseValues(voltieRegPowers, 1)
}

// getMinMaxCurrent implements the api.CurrentLimiter interface. The maximum is
// the ampacity set by the potentiometer on the EVSE board; the cable's proximity
// pilot rating is deliberately not included by the firmware.
func (wb *Voltie) getMinMaxCurrent() (float64, float64, error) {
	b, err := wb.read(wb.status)
	if err != nil {
		return 0, 0, err
	}

	// an unset or implausible potentiometer reading must not shrink the limit
	// below the J1772 minimum or raise it above what MaxCurrentMillis accepts
	max := float64(voltieU16(wb.status, b, voltieRegMaxCurrent)) / 1e3
	if max < voltieMinCurrent || max > voltieMaxCurrent {
		max = voltieMaxCurrent
	}

	return voltieMinCurrent, max, nil
}

var _ api.Diagnosis = (*Voltie)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Voltie) Diagnose() {
	if b, err := wb.read(wb.info); err == nil {
		fmt.Printf("\tCharger ID:\t%d\n", voltieU16(wb.info, b, voltieRegChargerID))
		fmt.Printf("\tFirmware:\t%d\n", voltieU16(wb.info, b, voltieRegFirmware))
		fmt.Printf("\tMCU serial:\t%d\n", voltieSerial(b, wb.info.ByteOffset(voltieRegMcuSerial)))
		fmt.Printf("\tPower serial:\t%d\n", voltieSerial(b, wb.info.ByteOffset(voltieRegHpowSerial)))
	}

	if b, err := wb.read(wb.status); err == nil {
		state := voltieU16(wb.status, b, voltieRegStatus)
		fmt.Printf("\tStatus:\t\t0x%02X (%s)\n", state, voltieStates[state])
		fmt.Printf("\tAuto start:\t%d\n", voltieU16(wb.status, b, voltieRegAutoStart))
		fmt.Printf("\tCharging:\t%d\n", voltieU16(wb.status, b, voltieRegCharging))
		fmt.Printf("\tPhases:\t\t%d\n", voltieU16(wb.status, b, voltieRegPhases))
		fmt.Printf("\tDLM mode:\t%d set, %d effective\n", voltieU16(wb.status, b, voltieRegDlmSet), voltieU16(wb.status, b, voltieRegDlmEffective))
		fmt.Printf("\tCurrent limit:\t%d mA\n", voltieU16(wb.status, b, voltieRegCurrent))

		reason := voltieU16(wb.status, b, voltieRegStopReason)
		fmt.Printf("\tStop reason:\t%d (%s)\n", reason, voltieStopReasons[reason])

		if wb.ext {
			fmt.Printf("\tSingle phase:\t%d\n", voltieU16(wb.status, b, voltieRegSinglePhase))
			fmt.Printf("\tQuery timeout:\t%d s\n", voltieU16(wb.status, b, voltieRegQueryTimeout))
			fmt.Printf("\tMax capacity:\t%d mA\n", voltieU16(wb.status, b, voltieRegMaxCurrent))
		}
	}

	if b, err := wb.read(wb.meter); err == nil {
		fmt.Printf("\tCapacity:\t%d mA\n", voltieU32(wb.meter, b, voltieRegCapacity))

		if wb.ext {
			fmt.Printf("\tLifetime:\t%d Wh\n", voltieU32(wb.meter, b, voltieRegTotalEnergy))
			fmt.Printf("\tPhase power:\t%d W, %d W, %d W\n",
				voltieU32(wb.meter, b, voltieRegPowers),
				voltieU32(wb.meter, b, voltieRegPowers+2),
				voltieU32(wb.meter, b, voltieRegPowers+4))
		}
	}
}
