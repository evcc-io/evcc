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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// DeyeEvse charger implementation for the Deye SUN-EVSE11K01/22K01-EU-AC.
// Registers are reverse engineered, see https://github.com/davidrapan/ha-solarman/pull/1073.
// The charger takes a power setpoint in W; writing 0 stops charging.
type DeyeEvse struct {
	lp    loadpoint.API
	conn  *modbus.Connection
	power uint16
}

const (
	deyeEvseRegPowerLimit = 18 // 0x0012 charging power limit W, 0 stops charging
	deyeEvseRegEnergy     = 34 // 0x0022 total charged energy kWh
	deyeEvseRegPowers     = 38 // 0x0026 charging power L1-L3 ×10 W, 3 regs
	deyeEvseRegVoltages   = 44 // 0x002C charging voltage L1-L3 ×0.1 V, 3 regs
	deyeEvseRegCurrents   = 47 // 0x002F charging current L1-L3 ×0.1 A, 3 regs
	deyeEvseRegState      = 57 // 0x0039 charging state, low byte
)

func init() {
	registry.AddCtx("deye-evse", NewDeyeEvseFromConfig)
}

// NewDeyeEvseFromConfig creates a Deye EVSE charger from generic config
func NewDeyeEvseFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 3,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDeyeEvse(ctx, cc)
}

// NewDeyeEvse creates Deye EVSE charger
func NewDeyeEvse(ctx context.Context, settings modbus.Settings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("deye-evse")
	conn.Logger(log.TRACE)

	wb := &DeyeEvse{
		conn:  conn,
		power: uint16(3 * voltage * 6), // assume min power
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *DeyeEvse) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(deyeEvseRegState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b) & 0x7F; s {
	case 0x01, 0x02: // available, finishing
		return api.StatusA, nil
	case 0x04, 0x08, 0x10: // preparing, suspended by EV, suspended by EVSE
		return api.StatusB, nil
	case 0x20: // charging
		return api.StatusC, nil
	default: // 0x40 faulted
		return api.StatusNone, fmt.Errorf("invalid status: %#x", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *DeyeEvse) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(deyeEvseRegPowerLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *DeyeEvse) Enable(enable bool) error {
	var power uint16
	if enable {
		power = wb.power
	}

	return wb.setPower(power)
}

// setPower writes the power limit in W
func (wb *DeyeEvse) setPower(power uint16) error {
	_, err := wb.conn.WriteSingleRegister(deyeEvseRegPowerLimit, power)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *DeyeEvse) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*DeyeEvse)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *DeyeEvse) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	phases := 3
	if wb.lp != nil {
		if p := wb.lp.ActivePhases(); p != 0 {
			phases = p
		}
	}

	power := uint16(voltage * current * float64(phases))

	err := wb.setPower(power)
	if err == nil {
		wb.power = power
	}

	return err
}

func (wb *DeyeEvse) phaseValues(reg uint16, scale float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var v [3]float64
	for i := range v {
		v[i] = float64(binary.BigEndian.Uint16(b[2*i:])) * scale
	}

	return v[0], v[1], v[2], nil
}

var _ api.Meter = (*DeyeEvse)(nil)

// CurrentPower implements the api.Meter interface
func (wb *DeyeEvse) CurrentPower() (float64, error) {
	l1, l2, l3, err := wb.Powers()
	return l1 + l2 + l3, err
}

var _ api.PhasePowers = (*DeyeEvse)(nil)

// Powers implements the api.PhasePowers interface
func (wb *DeyeEvse) Powers() (float64, float64, float64, error) {
	return wb.phaseValues(deyeEvseRegPowers, 10)
}

var _ api.PhaseCurrents = (*DeyeEvse)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *DeyeEvse) Currents() (float64, float64, float64, error) {
	return wb.phaseValues(deyeEvseRegCurrents, 0.1)
}

var _ api.PhaseVoltages = (*DeyeEvse)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *DeyeEvse) Voltages() (float64, float64, float64, error) {
	return wb.phaseValues(deyeEvseRegVoltages, 0.1)
}

var _ api.MeterEnergy = (*DeyeEvse)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *DeyeEvse) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(deyeEvseRegEnergy, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ loadpoint.Controller = (*DeyeEvse)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *DeyeEvse) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
