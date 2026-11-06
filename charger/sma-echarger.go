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
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// SmaECharger charger implementation for the SMA eCharger (Modbus profile SW 2.7)
type SmaECharger struct {
	conn    *modbus.Connection
	phases  int
	current float64
}

const (
	// input registers
	smaEChargerRegMainModel   = 30051 // UINT32
	smaEChargerRegModel       = 30053 // UINT32
	smaEChargerRegVendor      = 30055 // UINT32
	smaEChargerRegPkgRev      = 30059 // UINT32
	smaEChargerRegSerial      = 35549 // STRING
	smaEChargerRegSwRev       = 35565 // STRING
	smaEChargerRegHealth      = 35465 // ENUM32
	smaEChargerRegEvent       = 30213 // UINT32
	smaEChargerRegLimitSrc    = 35547 // UINT32
	smaEChargerRegMaxPowerRtg = 33019 // INT32 W
	smaEChargerRegPowers      = 30777 // INT32 W L1-L3
	smaEChargerRegVoltages    = 30783 // UINT32 0.01 V L1-L3
	smaEChargerRegCurrents    = 30977 // INT32 0.001 A L1-L3
	smaEChargerRegSession     = 35453 // UINT64 Wh
	smaEChargerRegPower       = 35457 // INT32 W
	smaEChargerRegEnergy      = 35459 // UINT64 Wh
	smaEChargerRegStatus      = 35463 // ENUM32

	// holding registers
	smaEChargerRegMaxCurrent = 40480 // UINT32 0.001 A
	smaEChargerRegMinCurrent = 41473 // UINT32 0.1 A
	smaEChargerRegMode       = 41471 // ENUM32
	smaEChargerRegEnable     = 41475 // ENUM32
	smaEChargerRegChargeLock = 41477 // ENUM32, charging release required via app, RFID or UI
	smaEChargerRegSetpoint   = 44383 // INT32 W
	smaEChargerRegMinPower   = 44387 // INT32 W

	smaEChargerDisabled      = 5171
	smaEChargerEnabled       = 5172
	smaEChargerChargeLockOn  = 1129
	smaEChargerChargeLockOff = 1130
	smaEChargerModeFast      = 4718

	smaEChargerNaN32 = 0x80000000
)

func init() {
	registry.AddCtx("sma-echarger", NewSmaEChargerFromConfig)
}

// NewSmaEChargerFromConfig creates a SMA eCharger from generic config
func NewSmaEChargerFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 3,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSmaECharger(ctx, cc)
}

// NewSmaECharger creates SMA eCharger charger
func NewSmaECharger(ctx context.Context, settings modbus.Settings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("sma-echarger")
	conn.Logger(log.TRACE)

	wb := &SmaECharger{
		conn:   conn,
		phases: 3,
	}

	// pv surplus and target modes would regulate charging on their own
	if v, err := wb.readHolding32(smaEChargerRegMode); err != nil {
		return nil, fmt.Errorf("mode: %w", err)
	} else if v != smaEChargerModeFast {
		log.WARN.Printf("charging mode is %d, fast charging (%d) required", v, smaEChargerModeFast)
	}

	return wb, nil
}

func (wb *SmaECharger) write(reg uint16, val uint32) error {
	b := binary.BigEndian.AppendUint32(nil, val)
	_, err := wb.conn.WriteMultipleRegisters(reg, 2, b)
	return err
}

// read32 reads count consecutive 32 bit input registers
func (wb *SmaECharger) read32(reg uint16, count int) ([]uint32, error) {
	b, err := wb.conn.ReadInputRegisters(reg, uint16(2*count))
	if err != nil {
		return nil, err
	}

	res := make([]uint32, count)
	for i := range res {
		res[i] = binary.BigEndian.Uint32(b[4*i:])
	}

	return res, nil
}

// smaEChargerInt32 converts a signed 32 bit value, treating NaN as zero
func smaEChargerInt32(u uint32) float64 {
	if u == smaEChargerNaN32 {
		return 0
	}
	return float64(int32(u))
}

// Status implements the api.Charger interface
func (wb *SmaECharger) Status() (api.ChargeStatus, error) {
	v, err := wb.read32(smaEChargerRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := v[0]; s {
	case 200111: // not connected
		return api.StatusA, nil
	case 200112, 200114, 5169: // sleeping, charge target reached, locked
		return api.StatusB, nil
	case 200113: // charging
		return api.StatusC, nil
	default: // 200106 warning, 200115 error
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

var _ api.StatusReasoner = (*SmaECharger)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *SmaECharger) StatusReason() (api.Reason, error) {
	v, err := wb.read32(smaEChargerRegStatus, 1)
	if err != nil || v[0] != 5169 {
		return api.ReasonUnknown, err
	}

	// locked status may also be caused by disabling charging, only the charge lock requires authorization
	lock, err := wb.readHolding32(smaEChargerRegChargeLock)
	if err != nil || lock != smaEChargerChargeLockOn {
		return api.ReasonUnknown, err
	}

	return api.ReasonWaitingForAuthorization, nil
}

// Enabled implements the api.Charger interface
func (wb *SmaECharger) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(smaEChargerRegEnable, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) == smaEChargerEnabled, nil
}

// Enable implements the api.Charger interface
func (wb *SmaECharger) Enable(enable bool) error {
	val := uint32(smaEChargerDisabled)
	if enable {
		val = smaEChargerEnabled
	}

	return wb.write(smaEChargerRegEnable, val)
}

// MaxCurrent implements the api.Charger interface
func (wb *SmaECharger) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SmaECharger)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SmaECharger) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	if err := wb.setPower(current, wb.phases); err != nil {
		return err
	}

	wb.current = current

	return nil
}

// setPower writes the power limit, the charger selects 1p or 3p internally based on it
func (wb *SmaECharger) setPower(current float64, phases int) error {
	power := int32(math.Round(grid.CurrentToPower(current, phases)))
	return wb.write(smaEChargerRegSetpoint, uint32(power))
}

var _ api.PhaseSwitcher = (*SmaECharger)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *SmaECharger) Phases1p3p(phases int) error {
	// loadpoint may not rewrite an unchanged current, hence apply the new phases immediately
	if wb.current > 0 {
		if err := wb.setPower(wb.current, phases); err != nil {
			return err
		}
	}

	wb.phases = phases

	return nil
}

var _ api.PhaseGetter = (*SmaECharger)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *SmaECharger) GetPhases() (int, error) {
	return wb.phases, nil
}

var _ api.Meter = (*SmaECharger)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SmaECharger) CurrentPower() (float64, error) {
	v, err := wb.read32(smaEChargerRegPower, 1)
	if err != nil {
		return 0, err
	}

	return smaEChargerInt32(v[0]), nil
}

// energy reads a 64 bit Wh counter in kWh, treating NaN as zero
func (wb *SmaECharger) energy(reg uint16) (float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 4)
	if err != nil {
		return 0, err
	}

	u := binary.BigEndian.Uint64(b)
	if u == math.MaxUint64 {
		return 0, nil
	}

	return float64(u) / 1e3, nil
}

var _ api.MeterEnergy = (*SmaECharger)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SmaECharger) TotalEnergy() (float64, error) {
	return wb.energy(smaEChargerRegEnergy)
}

var _ api.ChargeRater = (*SmaECharger)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *SmaECharger) ChargedEnergy() (float64, error) {
	return wb.energy(smaEChargerRegSession)
}

// phaseValues reads three 32 bit phase values; abs since grid measurements may be signed from the grid perspective
func (wb *SmaECharger) phaseValues(reg uint16, scale float64, signed bool) (float64, float64, float64, error) {
	v, err := wb.read32(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i, u := range v {
		if signed {
			res[i] = math.Abs(smaEChargerInt32(u)) * scale
		} else if u != math.MaxUint32 {
			res[i] = float64(u) * scale
		}
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhasePowers = (*SmaECharger)(nil)

// Powers implements the api.PhasePowers interface
func (wb *SmaECharger) Powers() (float64, float64, float64, error) {
	return wb.phaseValues(smaEChargerRegPowers, 1, true)
}

var _ api.PhaseCurrents = (*SmaECharger)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SmaECharger) Currents() (float64, float64, float64, error) {
	return wb.phaseValues(smaEChargerRegCurrents, 0.001, true)
}

var _ api.PhaseVoltages = (*SmaECharger)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *SmaECharger) Voltages() (float64, float64, float64, error) {
	return wb.phaseValues(smaEChargerRegVoltages, 0.01, false)
}

// readString reads a UTF-8 string input register of count 16 bit words
func (wb *SmaECharger) readString(reg uint16, count uint16) (string, error) {
	b, err := wb.conn.ReadInputRegisters(reg, count)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimRight(string(b), "\x00")), nil
}

// readHolding32 reads a 32 bit holding register
func (wb *SmaECharger) readHolding32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

// smaEChargerUint32 converts an unsigned 32 bit value, treating NaN as zero
func smaEChargerUint32(u uint32) float64 {
	if u == math.MaxUint32 {
		return 0
	}
	return float64(u)
}

var _ api.CurrentLimiter = (*SmaECharger)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *SmaECharger) GetMinMaxCurrent() (float64, float64, error) {
	minCurrent, err := wb.readHolding32(smaEChargerRegMinCurrent)
	if err != nil {
		return 0, 0, err
	}

	maxCurrent, err := wb.readHolding32(smaEChargerRegMaxCurrent)
	if err != nil {
		return 0, 0, err
	}

	return smaEChargerUint32(minCurrent) / 10, smaEChargerUint32(maxCurrent) / 1e3, nil
}

var _ api.PowerLimiter = (*SmaECharger)(nil)

// GetMinMaxPower implements the api.PowerLimiter interface
func (wb *SmaECharger) GetMinMaxPower() (float64, float64, error) {
	minPower, err := wb.readHolding32(smaEChargerRegMinPower)
	if err != nil {
		return 0, 0, err
	}

	maxPower, err := wb.read32(smaEChargerRegMaxPowerRtg, 1)
	if err != nil {
		return 0, 0, err
	}

	return smaEChargerInt32(minPower), smaEChargerInt32(maxPower[0]), nil
}

var _ api.Diagnosis = (*SmaECharger)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SmaECharger) Diagnose() {
	for _, r := range []struct {
		name string
		reg  uint16
	}{
		{"Main model", smaEChargerRegMainModel},
		{"Model", smaEChargerRegModel},
		{"Vendor", smaEChargerRegVendor},
		{"Package revision", smaEChargerRegPkgRev},
		{"Status", smaEChargerRegStatus},
		{"Health", smaEChargerRegHealth},
		{"Event", smaEChargerRegEvent},
		{"Power limit source", smaEChargerRegLimitSrc},
		{"Rated max power", smaEChargerRegMaxPowerRtg},
	} {
		if v, err := wb.read32(r.reg, 1); err == nil {
			fmt.Printf("\t%s:\t%d\n", r.name, v[0])
		}
	}

	for _, r := range []struct {
		name string
		reg  uint16
	}{
		{"Serial", smaEChargerRegSerial},
		{"Software", smaEChargerRegSwRev},
	} {
		if s, err := wb.readString(r.reg, 16); err == nil {
			fmt.Printf("\t%s:\t%s\n", r.name, s)
		}
	}

	for _, r := range []struct {
		name string
		reg  uint16
	}{
		{"Mode", smaEChargerRegMode},
		{"Enable", smaEChargerRegEnable},
		{"Charge lock", smaEChargerRegChargeLock},
		{"Max power", smaEChargerRegSetpoint},
		{"Min power", smaEChargerRegMinPower},
		{"Rated max current", smaEChargerRegMaxCurrent},
		{"Min current", smaEChargerRegMinCurrent},
	} {
		if b, err := wb.conn.ReadHoldingRegisters(r.reg, 2); err == nil {
			fmt.Printf("\t%s:\t%d\n", r.name, int32(binary.BigEndian.Uint32(b)))
		}
	}
}
