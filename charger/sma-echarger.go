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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// SmaECharger charger implementation for the SMA eCharger (Modbus profile SW 2.7)
type SmaECharger struct {
	conn *modbus.Connection
}

const (
	// input registers
	smaEChargerRegPowers   = 30777 // INT32 W L1-L3
	smaEChargerRegVoltages = 30783 // UINT32 0.01 V L1-L3
	smaEChargerRegCurrents = 30977 // INT32 0.001 A L1-L3
	smaEChargerRegSession  = 35453 // UINT64 Wh
	smaEChargerRegPower    = 35457 // INT32 W
	smaEChargerRegEnergy   = 35459 // UINT64 Wh
	smaEChargerRegStatus   = 35463 // ENUM32

	// holding registers
	smaEChargerRegPhases  = 40183 // UINT32 1/3 (assumed)
	smaEChargerRegMode    = 41471 // ENUM32
	smaEChargerRegCurrent = 41473 // UINT32 0.1 A
	smaEChargerRegAppLock = 41477 // ENUM32

	smaEChargerModeFast = 4718
	smaEChargerModeStop = 4721
	smaEChargerLockNo   = 1130

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

	conn.Logger(util.NewLogger("sma-echarger").TRACE)

	wb := &SmaECharger{
		conn: conn,
	}

	// app lock overrides the charge status and prevents vehicle detection
	if err := wb.write(smaEChargerRegAppLock, smaEChargerLockNo); err != nil {
		return nil, fmt.Errorf("app lock: %w", err)
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

// Enabled implements the api.Charger interface
func (wb *SmaECharger) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(smaEChargerRegMode, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) != smaEChargerModeStop, nil
}

// Enable implements the api.Charger interface
func (wb *SmaECharger) Enable(enable bool) error {
	mode := uint32(smaEChargerModeStop)
	if enable {
		mode = smaEChargerModeFast
	}

	return wb.write(smaEChargerRegMode, mode)
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

	return wb.write(smaEChargerRegCurrent, uint32(math.Round(current*10)))
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

// phases reads three 32 bit phase values; abs since grid measurements may be signed from the grid perspective
func (wb *SmaECharger) phases(reg uint16, scale float64, signed bool) (float64, float64, float64, error) {
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
	return wb.phases(smaEChargerRegPowers, 1, true)
}

var _ api.PhaseCurrents = (*SmaECharger)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SmaECharger) Currents() (float64, float64, float64, error) {
	return wb.phases(smaEChargerRegCurrents, 0.001, true)
}

var _ api.PhaseVoltages = (*SmaECharger)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *SmaECharger) Voltages() (float64, float64, float64, error) {
	return wb.phases(smaEChargerRegVoltages, 0.01, false)
}

var _ api.PhaseSwitcher = (*SmaECharger)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *SmaECharger) Phases1p3p(phases int) error {
	return wb.write(smaEChargerRegPhases, uint32(phases))
}

var _ api.PhaseGetter = (*SmaECharger)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *SmaECharger) GetPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(smaEChargerRegPhases, 2)
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint32(b)), nil
}
