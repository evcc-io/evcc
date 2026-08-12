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
	"time"

	"github.com/evcc-io/evcc/api"
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

const (
	// register blocks fetched in bulk
	voltieRegInfoBlock   = 0x0000
	voltieLenInfoBlock   = 10 // 0x0000..0x0009
	voltieRegStatusBlock = 0x000A
	voltieLenStatusBlock = 12 // 0x000A..0x0015
	voltieRegMeterBlock  = 0x2000
	voltieLenMeterBlock  = 22 // 0x2000..0x2015

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
	voltieRegPhases       = 0x000E // INT16 number of phases in use
	voltieRegDlmSet       = 0x000F // INT16 stored DLM mode
	voltieRegStopReason   = 0x0012 // INT16 charge stop reason
	voltieRegCurrent      = 0x0014 // INT16 software current limit [mA]
	voltieRegDlmEffective = 0x0015 // INT16 effective DLM mode

	// meter block
	voltieRegVoltages  = 0x2000 // 3x INT32 phase voltage [mV]
	voltieRegCurrents  = 0x2006 // 3x INT32 phase charging current [mA]
	voltieRegDuration  = 0x200C // INT32 charge duration [s]
	voltieRegEnergy    = 0x200E // INT32 charged energy in session [Ws]
	voltieRegPower     = 0x2010 // INT32 charging power [W]
	voltieRegCapacity  = 0x2012 // INT32 instantaneous current capacity [mA]
	voltieRegTempMcu   = 0x2014 // INT16 MCU temperature [°C]
	voltieRegTempPower = 0x2015 // INT16 power board temperature [°C]

	// the charger's ampacity range [A]. The current limit register is typed
	// INT16, so the milliampere value must stay below the sign boundary.
	voltieMinCurrent = 6
	voltieMaxCurrent = 32

	// firmware that fixes the Modbus slave address checks and rejects FC16
	voltieMinFirmware = 350

	// the charger's documented default slave address
	voltieDefaultSlaveID = 11

	// the gateway abandons a forwarded request after 3s, so the client must
	// wait longer than that to receive the resulting exception
	voltieTimeout = 5 * time.Second

	// the gateway forwards at most two transactions per second
	voltieDelay = 500 * time.Millisecond
)

// EVSE states, see the "EVSE states" chapter of the Modbus API documentation
const (
	voltieStateA = 0x01 // not connected
	voltieStateB = 0x02 // connected, ready
	voltieStateC = 0x03 // charging
	voltieStateD = 0x04 // charging, ventilation required
)

var voltieStates = map[uint16]string{
	0x00: "state not yet determined",
	0x01: "vehicle state A, not connected",
	0x02: "vehicle state B, connected",
	0x03: "vehicle state C, charging",
	0x04: "vehicle state D, charging with ventilation",
	0x05: "diode check failed",
	0x06: "GFCI fault",
	0x07: "bad ground",
	0x08: "relay stuck",
	0x09: "GFI self-test failure",
	0x0A: "over temperature",
	0x0B: "over current",
	0x0C: "hardware fault (voltage, current or temperature sensor)",
	0x0D: "vehicle state E, vehicle error",
	0x0E: "over humidity",
	0x0F: "input power phase misconnected",
	0x10: "overvoltage on the grid",
	0x11: "undervoltage on the grid",
	0x12: "charger disabled, not functioning",
	0x13: "booting",
	0x14: "no MID meter detected",
	0x15: "power board unidentified",
	0x18: "state undetermined",
	0x19: "uploading VoltieMeter firmware",
}

// charge stop reasons reported by the MCU, see the "EVSE charge stop reasons"
// chapter. Reasons 23..31 originate in the charger's control software and are
// not reported through Modbus.
var voltieStopReasons = map[uint16]string{
	1:   "unspecified reason",
	2:   "preset charge duration reached",
	3:   "preset energy amount charged",
	4:   "stopped by the user",
	5:   "GFCI sensor tripped",
	6:   "charger disabled, out of order",
	7:   "firmware restart",
	8:   "charger in sleep mode, out of order",
	9:   "no voltage on the output (ground continuity or relay error)",
	10:  "vehicle disconnected",
	11:  "vehicle not accepting charge",
	12:  "power board I2C bus fault",
	13:  "GFCI self test failed",
	14:  "over temperature",
	15:  "diode error",
	16:  "PE-N over-voltage",
	17:  "relay stuck",
	18:  "over current",
	21:  "over humidity",
	22:  "wrong phase order on the input",
	100: "not enough free building current available (dynamic load management)",
	101: "not enough solar current available (eco/green mode)",
	102: "grid voltage is not high enough (grid-controlled mode)",
	103: "charge current limit set to zero",
	104: "no MID meter available",
	105: "overvoltage",
	106: "vehicle error",
	107: "undervoltage",
	108: "vehicle in state D while state D is disabled",
	109: "power board unidentified",
}

// Voltie is an api.Charger implementation for Voltie wallboxes
type Voltie struct {
	conn   *modbus.Connection
	log    *util.Logger
	cache  *modbus.Cache
	status modbus.Block
	meter  modbus.Block
	info   modbus.Block
}

// voltieBlockReader decodes registers out of a fetched block payload by their
// register address. Decoding errors are collected and checked once by the
// caller, so a register outside the block cannot pass unnoticed.
type voltieBlockReader struct {
	block   modbus.Block
	payload []byte
	err     error
}

func (r *voltieBlockReader) registers(addr, count uint16) []byte {
	if r.err != nil {
		return make([]byte, 2*count)
	}

	b, err := r.block.Extract(modbus.RegisterOperation{Addr: addr, Length: count}, r.payload)
	if err != nil {
		r.err = err
		return make([]byte, 2*count)
	}

	return b
}

func (r *voltieBlockReader) u16(addr uint16) uint16 {
	return binary.BigEndian.Uint16(r.registers(addr, 1))
}

func (r *voltieBlockReader) i16(addr uint16) int16 {
	return int16(r.u16(addr))
}

func (r *voltieBlockReader) u32(addr uint16) uint32 {
	return binary.BigEndian.Uint32(r.registers(addr, 2))
}

func (r *voltieBlockReader) i32(addr uint16) int32 {
	return int32(r.u32(addr))
}

// serial decodes a 64 bit serial number, which is sent least-significant word
// first unlike the 32 bit metering values
func (r *voltieBlockReader) serial(addr uint16) uint64 {
	b := r.registers(addr, voltieSerialRegCount)

	var res uint64
	for i := range voltieSerialRegCount {
		res |= uint64(binary.BigEndian.Uint16(b[2*i:])) << (16 * i)
	}

	return res
}

// read fetches a register block through the shared bulk read cache, so all
// values taken from the same block within a poll cycle cost one request
func (wb *Voltie) read(block modbus.Block) (*voltieBlockReader, error) {
	key := fmt.Sprintf("%s/holding/%d/%d", wb.conn.Addr(), block.Register, block.Count)

	payload, _, err := wb.cache.Fetch(key, func() ([]byte, error) {
		return wb.conn.ReadHoldingRegisters(block.Register, block.Count)
	})
	if err != nil {
		return nil, err
	}

	return &voltieBlockReader{block: block, payload: payload}, nil
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
		conn:   conn,
		log:    log,
		cache:  modbus.NewCache(cache),
		info:   modbus.Block{Register: voltieRegInfoBlock, Count: voltieLenInfoBlock},
		status: modbus.Block{Register: voltieRegStatusBlock, Count: voltieLenStatusBlock},
		meter:  modbus.Block{Register: voltieRegMeterBlock, Count: voltieLenMeterBlock},
	}

	if r, err := wb.read(wb.info); err == nil {
		if fw := r.u16(voltieRegFirmware); r.err == nil && fw < voltieMinFirmware {
			log.WARN.Printf("firmware %d is outdated, Modbus TCP requires %d or later", fw, voltieMinFirmware)
		}
	}

	if err := wb.checkSettings(); err != nil {
		return nil, err
	}

	return wb, nil
}

// checkSettings inspects the charger's settings once on startup. The charger
// must not start a session on its own while evcc is in control, and its own
// load management would silently cap the current requested by evcc.
func (wb *Voltie) checkSettings() error {
	r, err := wb.read(wb.status)
	if err != nil {
		return err
	}

	dlm := r.u16(voltieRegDlmEffective)
	autoStart := r.u16(voltieRegAutoStart)
	if r.err != nil {
		return r.err
	}

	if dlm != 0 {
		wb.log.WARN.Printf("charger-side load management is active (mode %d) and will cap the requested current", dlm)
	}

	// the auto start setting is persisted in the charger's EEPROM and stays off
	// after evcc is removed, so it is only written when actually enabled
	if autoStart == 0 {
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
	r, err := wb.read(wb.status)
	if err != nil {
		return api.StatusNone, err
	}

	state := r.u16(voltieRegStatus)
	if r.err != nil {
		return api.StatusNone, r.err
	}

	if status, ok := voltieChargeStatus(state); ok {
		return status, nil
	}

	return api.StatusNone, voltieStateError(r, state)
}

// voltieChargeStatus maps an EVSE state to a charge status. States the charger
// cannot charge in have no mapping.
func voltieChargeStatus(state uint16) (api.ChargeStatus, bool) {
	switch state {
	case voltieStateA:
		return api.StatusA, true
	case voltieStateB:
		return api.StatusB, true
	case voltieStateC, voltieStateD:
		return api.StatusC, true
	default:
		return api.StatusNone, false
	}
}

// voltieStateError describes an EVSE state the charger cannot charge in,
// adding the stop reason reported by the MCU where one is available
func voltieStateError(r *voltieBlockReader, state uint16) error {
	desc := voltieStateName(state)

	reason := r.u16(voltieRegStopReason)
	if reason == 0 {
		return fmt.Errorf("%s (0x%02X)", desc, state)
	}

	if txt, ok := voltieStopReasons[reason]; ok {
		return fmt.Errorf("%s (0x%02X): %s", desc, state, txt)
	}

	return fmt.Errorf("%s (0x%02X): stop reason %d", desc, state, reason)
}

// Enabled implements the api.Charger interface
func (wb *Voltie) Enabled() (bool, error) {
	r, err := wb.read(wb.status)
	if err != nil {
		return false, err
	}

	return r.u16(voltieRegChargeEnable) != 0, r.err
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
	r, err := wb.read(wb.status)
	if err != nil {
		return 0, err
	}

	return float64(r.u16(voltieRegCurrent)) / 1e3, r.err
}

var _ api.Meter = (*Voltie)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Voltie) CurrentPower() (float64, error) {
	r, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return float64(r.i32(voltieRegPower)), r.err
}

var _ api.ChargeRater = (*Voltie)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Voltie) ChargedEnergy() (float64, error) {
	r, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return float64(r.u32(voltieRegEnergy)) / 3.6e6, r.err // Ws to kWh
}

var _ api.ChargeTimer = (*Voltie)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Voltie) ChargeDuration() (time.Duration, error) {
	r, err := wb.read(wb.meter)
	if err != nil {
		return 0, err
	}

	return time.Duration(r.u32(voltieRegDuration)) * time.Second, r.err
}

var _ api.PhaseCurrents = (*Voltie)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Voltie) Currents() (float64, float64, float64, error) {
	r, err := wb.read(wb.meter)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(r.u32(voltieRegCurrents+uint16(2*i))) / 1e3 // mA to A
	}

	return res[0], res[1], res[2], r.err
}

var _ api.PhaseVoltages = (*Voltie)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Voltie) Voltages() (float64, float64, float64, error) {
	r, err := wb.read(wb.meter)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(r.u32(voltieRegVoltages+uint16(2*i))) / 1e3 // mV to V
	}

	return res[0], res[1], res[2], r.err
}

var _ api.PhaseGetter = (*Voltie)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *Voltie) GetPhases() (int, error) {
	r, err := wb.read(wb.status)
	if err != nil {
		return 0, err
	}

	return int(r.u16(voltieRegPhases)), r.err
}

var _ api.Diagnosis = (*Voltie)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Voltie) Diagnose() {
	if r, err := wb.read(wb.info); err == nil {
		fmt.Printf("\tCharger ID:\t%d\n", r.u16(voltieRegChargerID))
		fmt.Printf("\tFirmware:\t%d\n", r.u16(voltieRegFirmware))
		fmt.Printf("\tMCU serial:\t%d\n", r.serial(voltieRegMcuSerial))
		fmt.Printf("\tPower serial:\t%d\n", r.serial(voltieRegHpowSerial))
	}

	if r, err := wb.read(wb.status); err == nil {
		state := r.u16(voltieRegStatus)
		fmt.Printf("\tStatus:\t\t0x%02X (%s)\n", state, voltieStateName(state))
		fmt.Printf("\tAuto start:\t%d\n", r.u16(voltieRegAutoStart))
		fmt.Printf("\tCharging:\t%d\n", r.u16(voltieRegCharging))
		fmt.Printf("\tPhases:\t\t%d\n", r.u16(voltieRegPhases))
		fmt.Printf("\tDLM mode:\t%d set, %d effective\n", r.u16(voltieRegDlmSet), r.u16(voltieRegDlmEffective))
		fmt.Printf("\tCurrent limit:\t%d mA\n", r.u16(voltieRegCurrent))

		reason := r.u16(voltieRegStopReason)
		fmt.Printf("\tStop reason:\t%d (%s)\n", reason, voltieStopReasons[reason])
	}

	if r, err := wb.read(wb.meter); err == nil {
		fmt.Printf("\tCapacity:\t%d mA\n", r.u32(voltieRegCapacity))
		fmt.Printf("\tTemperature:\t%d °C MCU, %d °C power board\n",
			r.i16(voltieRegTempMcu), r.i16(voltieRegTempPower))
	}
}

// voltieStateName describes an EVSE state, falling back for states the
// firmware may add in the future
func voltieStateName(state uint16) string {
	if desc, ok := voltieStates[state]; ok {
		return desc
	}

	return "unknown state"
}
