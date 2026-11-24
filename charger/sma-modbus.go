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

// SmaModbus charger implementation for the SMA eCharger (Modbus profile SW 2.7)
type SmaModbus struct {
	conn     *modbus.Connection
	phases   int
	current  float64
	minPower float64
	maxPower float64
}

const (
	// input registers
	smaModbusRegMainModel   = 30051 // UINT32
	smaModbusRegModel       = 30053 // UINT32
	smaModbusRegVendor      = 30055 // UINT32
	smaModbusRegPkgRev      = 30059 // UINT32
	smaModbusRegSerial      = 35549 // STRING
	smaModbusRegSwRev       = 35565 // STRING
	smaModbusRegHealth      = 35465 // ENUM32
	smaModbusRegEvent       = 30213 // UINT32
	smaModbusRegLimitSrc    = 35547 // UINT32
	smaModbusRegMaxPowerRtg = 33019 // INT32 W
	smaModbusRegPowers      = 30777 // INT32 W L1-L3
	smaModbusRegVoltages    = 30783 // UINT32 0.01 V L1-L3
	smaModbusRegCurrents    = 30977 // INT32 0.001 A L1-L3
	smaModbusRegSession     = 35453 // UINT64 Wh
	smaModbusRegPower       = 35457 // INT32 W
	smaModbusRegEnergy      = 35459 // UINT64 Wh
	smaModbusRegStatus      = 35463 // ENUM32

	// holding registers
	smaModbusRegMaxCurrent = 40480 // UINT32 0.001 A
	smaModbusRegMinCurrent = 41473 // UINT32 0.1 A
	smaModbusRegMode       = 41471 // ENUM32
	smaModbusRegRelease    = 41475 // ENUM32
	smaModbusRegChargeLock = 41477 // ENUM32, access management: free charging mode or app/RFID release
	smaModbusRegStpWhenFl  = 41485 // UINT32, disconnect after full charge
	smaModbusRegSetpoint   = 44383 // INT32 W
	smaModbusRegMinPower   = 44387 // INT32 W

	smaModbusYes          = 1129
	smaModbusChargeLockOn = smaModbusYes
	smaModbusReleased     = 5172
	smaModbusModeFast     = 4718
	smaModbusModeStop     = 4721

	smaModbusNaN32 = 0x80000000

	// the charger delivers about 0.5 A less than the setpoint and drops to 1p below 6 A at 3p
	smaModbusMinCurrent3p = 6.5 // A
)

func init() {
	registry.AddCtx("sma-modbus", NewSmaModbusFromConfig)
}

// NewSmaModbusFromConfig creates a SMA eCharger from generic config
func NewSmaModbusFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 3,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSmaModbus(ctx, cc)
}

// NewSmaModbus creates SMA eCharger charger
func NewSmaModbus(ctx context.Context, settings modbus.Settings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("sma-modbus")
	conn.Logger(log.TRACE)

	wb := &SmaModbus{
		conn:   conn,
		phases: 3,
	}

	// pv surplus and target modes would regulate charging on their own
	if v, err := wb.readHolding32(smaModbusRegMode); err != nil {
		return nil, fmt.Errorf("mode: %w", err)
	} else if v != smaModbusModeStop && v != smaModbusModeFast {
		log.WARN.Printf("charging mode is %d, fast charging (%d) required", v, smaModbusModeFast)
	}

	// the charger may reject values outside its range, e.g. a 4500 W minimum on the EV Charger
	if v, err := wb.readHolding32(smaModbusRegMinPower); err == nil {
		wb.minPower = smaModbusInt32(v)
	}
	if v, err := wb.read32(smaModbusRegMaxPowerRtg, 1); err == nil {
		wb.maxPower = smaModbusInt32(v[0])
	}

	// the charger may already charge on 1p after a restart
	if s, err := wb.Status(); err == nil && s == api.StatusC {
		if l1, l2, l3, err := wb.Currents(); err == nil {
			var active int
			for _, l := range []float64{l1, l2, l3} {
				if l > 1 {
					active++
				}
			}
			if active == 1 {
				wb.phases = 1
			}
		}
	}

	// disconnecting after full charge ends the session on the charger side
	if v, err := wb.readHolding32(smaModbusRegStpWhenFl); err == nil && v == smaModbusYes {
		log.WARN.Println(`"Disconnect after full charge" must be set to "No"`)
	}

	return wb, nil
}

func (wb *SmaModbus) write(reg uint16, val uint32) error {
	b := binary.BigEndian.AppendUint32(nil, val)
	_, err := wb.conn.WriteMultipleRegisters(reg, 2, b)
	return err
}

// read32 reads count consecutive 32 bit input registers
func (wb *SmaModbus) read32(reg uint16, count int) ([]uint32, error) {
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

// smaModbusInt32 converts a signed 32 bit value, treating NaN as zero
func smaModbusInt32(u uint32) float64 {
	if u == smaModbusNaN32 {
		return 0
	}
	return float64(int32(u))
}

// Status implements the api.Charger interface
func (wb *SmaModbus) Status() (api.ChargeStatus, error) {
	v, err := wb.read32(smaModbusRegStatus, 1)
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

var _ api.StatusReasoner = (*SmaModbus)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *SmaModbus) StatusReason() (api.Reason, error) {
	lock, err := wb.readHolding32(smaModbusRegChargeLock)
	if err != nil || lock != smaModbusChargeLockOn {
		return api.ReasonUnknown, err
	}

	release, err := wb.readHolding32(smaModbusRegRelease)
	if err != nil || release == smaModbusReleased {
		return api.ReasonUnknown, err
	}

	return api.ReasonWaitingForAuthorization, nil
}

// Enabled implements the api.Charger interface
func (wb *SmaModbus) Enabled() (bool, error) {
	v, err := wb.readHolding32(smaModbusRegMode)
	if err != nil {
		return false, err
	}

	// the charger falls back to its default pv surplus mode on its own, report it as disabled to restore fast charging
	return v == smaModbusModeFast, nil
}

// Enable implements the api.Charger interface
func (wb *SmaModbus) Enable(enable bool) error {
	mode := uint32(smaModbusModeStop)
	if enable {
		mode = smaModbusModeFast
	}

	return wb.write(smaModbusRegMode, mode)
}

// MaxCurrent implements the api.Charger interface
func (wb *SmaModbus) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SmaModbus)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SmaModbus) MaxCurrentMillis(current float64) error {
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
func (wb *SmaModbus) setPower(current float64, phases int) error {
	if phases > 1 {
		current = max(current, smaModbusMinCurrent3p)
	}

	power := grid.CurrentToPower(current, phases) * wb.voltageFactor(phases)

	// limits are reported as negative consumption values
	if minPower := math.Abs(wb.minPower); minPower > 0 {
		power = max(power, minPower)
	}
	if maxPower := math.Abs(wb.maxPower); maxPower > 0 {
		power = min(power, maxPower)
	}

	// the charger rejects positive setpoints, consumption is negative
	return wb.write(smaModbusRegSetpoint, uint32(int32(-math.Ceil(power))))
}

// voltageFactor scales from nominal to measured voltage since the charger derives the phase current from the latter
func (wb *SmaModbus) voltageFactor(phases int) float64 {
	u1, u2, u3, err := wb.Voltages()
	if err != nil {
		return 1
	}

	u := u1
	if phases > 1 {
		u = max(u1, u2, u3)
	}
	if u <= 0 {
		return 1
	}

	return u / grid.Voltage
}

var _ api.PhaseSwitcher = (*SmaModbus)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *SmaModbus) Phases1p3p(phases int) error {
	// loadpoint may not rewrite an unchanged current, hence apply the new phases immediately
	if wb.current > 0 {
		if err := wb.setPower(wb.current, phases); err != nil {
			return err
		}
	}

	wb.phases = phases

	return nil
}

var _ api.PhaseGetter = (*SmaModbus)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *SmaModbus) GetPhases() (int, error) {
	return wb.phases, nil
}

var _ api.Meter = (*SmaModbus)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SmaModbus) CurrentPower() (float64, error) {
	v, err := wb.read32(smaModbusRegPower, 1)
	if err != nil {
		return 0, err
	}

	return smaModbusInt32(v[0]), nil
}

// energy reads a 64 bit Wh counter in kWh, treating NaN as zero
func (wb *SmaModbus) energy(reg uint16) (float64, error) {
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

var _ api.MeterEnergy = (*SmaModbus)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SmaModbus) TotalEnergy() (float64, error) {
	return wb.energy(smaModbusRegEnergy)
}

var _ api.ChargeRater = (*SmaModbus)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *SmaModbus) ChargedEnergy() (float64, error) {
	return wb.energy(smaModbusRegSession)
}

// phaseValues reads three 32 bit phase values; abs since grid measurements may be signed from the grid perspective
func (wb *SmaModbus) phaseValues(reg uint16, scale float64, signed bool) (float64, float64, float64, error) {
	v, err := wb.read32(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i, u := range v {
		if signed {
			res[i] = math.Abs(smaModbusInt32(u)) * scale
		} else if u != math.MaxUint32 {
			res[i] = float64(u) * scale
		}
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhasePowers = (*SmaModbus)(nil)

// Powers implements the api.PhasePowers interface
func (wb *SmaModbus) Powers() (float64, float64, float64, error) {
	return wb.phaseValues(smaModbusRegPowers, 1, true)
}

var _ api.PhaseCurrents = (*SmaModbus)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SmaModbus) Currents() (float64, float64, float64, error) {
	return wb.phaseValues(smaModbusRegCurrents, 0.001, true)
}

var _ api.PhaseVoltages = (*SmaModbus)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *SmaModbus) Voltages() (float64, float64, float64, error) {
	return wb.phaseValues(smaModbusRegVoltages, 0.01, false)
}

// readString reads a UTF-8 string input register of count 16 bit words
func (wb *SmaModbus) readString(reg uint16, count uint16) (string, error) {
	b, err := wb.conn.ReadInputRegisters(reg, count)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimRight(string(b), "\x00")), nil
}

// readHolding32 reads a 32 bit holding register
func (wb *SmaModbus) readHolding32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

// smaModbusUint32 converts an unsigned 32 bit value, treating NaN as zero
func smaModbusUint32(u uint32) float64 {
	if u == math.MaxUint32 {
		return 0
	}
	return float64(u)
}

var _ api.CurrentLimiter = (*SmaModbus)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *SmaModbus) GetMinMaxCurrent() (float64, float64, error) {
	minCurrent, err := wb.readHolding32(smaModbusRegMinCurrent)
	if err != nil {
		return 0, 0, err
	}

	maxCurrent, err := wb.readHolding32(smaModbusRegMaxCurrent)
	if err != nil {
		return 0, 0, err
	}

	return smaModbusUint32(minCurrent) / 10, smaModbusUint32(maxCurrent) / 1e3, nil
}

var _ api.Diagnosis = (*SmaModbus)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SmaModbus) Diagnose() {
	for _, r := range []struct {
		name string
		reg  uint16
	}{
		{"Main model", smaModbusRegMainModel},
		{"Model", smaModbusRegModel},
		{"Vendor", smaModbusRegVendor},
		{"Package revision", smaModbusRegPkgRev},
		{"Status", smaModbusRegStatus},
		{"Health", smaModbusRegHealth},
		{"Event", smaModbusRegEvent},
		{"Power limit source", smaModbusRegLimitSrc},
		{"Rated max power", smaModbusRegMaxPowerRtg},
	} {
		if v, err := wb.read32(r.reg, 1); err == nil {
			fmt.Printf("\t%s:\t%d\n", r.name, v[0])
		}
	}

	for _, r := range []struct {
		name string
		reg  uint16
	}{
		{"Serial", smaModbusRegSerial},
		{"Software", smaModbusRegSwRev},
	} {
		if s, err := wb.readString(r.reg, 16); err == nil {
			fmt.Printf("\t%s:\t%s\n", r.name, s)
		}
	}

	for _, r := range []struct {
		name string
		reg  uint16
	}{
		{"Mode", smaModbusRegMode},
		{"Release", smaModbusRegRelease},
		{"Charge lock", smaModbusRegChargeLock},
		{"Disconnect after full charge", smaModbusRegStpWhenFl},
		{"Max power", smaModbusRegSetpoint},
		{"Min power", smaModbusRegMinPower},
		{"Rated max current", smaModbusRegMaxCurrent},
		{"Min current", smaModbusRegMinCurrent},
	} {
		if b, err := wb.conn.ReadHoldingRegisters(r.reg, 2); err == nil {
			fmt.Printf("\t%s:\t%d\n", r.name, int32(binary.BigEndian.Uint32(b)))
		}
	}
}
