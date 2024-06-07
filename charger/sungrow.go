package charger

// LICENSE

// Copyright (c) 2024 premultiply

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
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// Sungrow charger implementation
type Sungrow struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	// input (read only)
	sgRegPhase             = 21224 // uint16 [1: Single-phase,	3: Three-phase]
	sgRegWorkMode          = 21262 // uint16 [0: Network, 2: Plug&Play, 6: EMS]
	sgRegRemCtrlStatus     = 21267 // uint16 [0: Disable, 1: Enable]
	sgRegPhaseSwitchStatus = 21269 // uint16
	sgRegTotalEnergy       = 21299 // uint32s 1Wh
	sgRegActivePower       = 21307 // uint32s 1W
	sgRegChargedEnergy     = 21309 // uint32s 1Wh
	sgRegStartMode         = 21313 // uint16 [1: Started by EMS, 2: Started by swiping card]
	sgRegPowerRequest      = 21314 // uint16 [0: Enable, 1: Close]
	sgRegPowerFlag         = 21315 // uint16 [0: Charging or power regulation is not allowed; 1: Charging or power regulation is allowed]
	sgRegState             = 21316 // uint16

	// holding
	sgRegSetOutI       = 21202 // uint16 0.01A
	sgRegPhaseSwitch   = 21203 // uint16 [0: Three-phase, 1: Single-phase]
	sgRegUnavailable   = 21210 // uint16
	sgRegRemoteControl = 21211 // uint16 [0: Start, 1: Stop]
)

var (
	sgRegVoltages = []uint16{21301, 21303, 21305} // uint16 0.1V
	sgRegCurrents = []uint16{21302, 21304, 21306} // uint16 0.1A
)

func init() {
	registry.Add("sungrow", NewSungrowFromConfig)
}

// NewSungrowFromConfig creates a Sungrow charger from generic config
func NewSungrowFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 248,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSungrow(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewSungrow creates Sungrow charger
func NewSungrow(uri, device, comset string, baudrate int, proto modbus.Protocol, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("sungrow")
	conn.Logger(log.TRACE)

	wb := &Sungrow{
		log:  log,
		conn: conn,
		curr: 60,
	}

	return wb, err
}

// getPhaseValues returns 3 non-sequential register values
func (wb *Sungrow) getPhaseValues(regs []uint16, divider float64) (float64, float64, float64, error) {
	var res [3]float64
	for i, reg := range regs {
		b, err := wb.conn.ReadInputRegisters(reg, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = rs485.RTUUint16ToFloat64(b) / divider
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Sungrow) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 1: // Idle
		return api.StatusA, nil
	case
		2, // Standby
		4, // SuspendedEVSE
		5, // SuspendedEV
		6: // Completed
		return api.StatusB, nil
	case 3: // Charging
		return api.StatusC, nil
	case
		7, // Reserved
		8, // Disabled
		9: // Faulted
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Sungrow) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegStartMode, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Sungrow) Enable(enable bool) error {
	var u uint16
	if !enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(sgRegRemoteControl, u)

	if err == nil && enable {
		_, err = wb.conn.WriteSingleRegister(sgRegSetOutI, wb.curr)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Sungrow) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Sungrow)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Sungrow) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	curr := uint16(10 * current)

	_, err := wb.conn.WriteSingleRegister(sgRegSetOutI, curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*Sungrow)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Sungrow) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b), err
}

var _ api.PhaseCurrents = (*Sungrow)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Sungrow) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(sgRegCurrents, 10)
}

var _ api.PhaseVoltages = (*Sungrow)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Sungrow) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(sgRegVoltages, 10)
}

var _ api.ChargeRater = (*Sungrow)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Sungrow) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, err
}

var _ api.MeterEnergy = (*Sungrow)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Sungrow) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, err
}

var _ api.PhaseSwitcher = (*Sungrow)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Sungrow) Phases1p3p(phases int) error {
	var u uint16

	if phases == 1 {
		u = 1
	}

	enabled, err := wb.Enabled()
	if err == nil && enabled {
		if err = wb.Enable(false); err != nil {
			return err
		}
	}

	// Switch phases
	_, err = wb.conn.WriteSingleRegister(sgRegPhaseSwitch, u)

	// Re-enable charging if it was previously enabled
	if err == nil && enabled {
		err = wb.Enable(true)
	}

	return err
}

var _ api.PhaseGetter = (*Sungrow)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *Sungrow) GetPhases() (int, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegPhaseSwitchStatus, 1)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint16(b) == 0 {
		return 3, nil
	}

	return 1, nil
}

var _ api.Diagnosis = (*Sungrow)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Sungrow) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(sgRegSetOutI, 1); err == nil {
		fmt.Printf("\tSetOutI:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(sgRegPhaseSwitch, 1); err == nil {
		fmt.Printf("\tPhaseSwitch:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(sgRegUnavailable, 1); err == nil {
		fmt.Printf("\tUnavailable:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(sgRegRemoteControl, 1); err == nil {
		fmt.Printf("\tRemoteControl:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegWorkMode, 1); err == nil {
		fmt.Printf("\tWorkMode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPhase, 1); err == nil {
		fmt.Printf("\tPhase:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPhaseSwitchStatus, 1); err == nil {
		fmt.Printf("\tPhasesState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegStartMode, 1); err == nil {
		fmt.Printf("\tStartMode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegState, 1); err == nil {
		fmt.Printf("\tState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegRemCtrlStatus, 1); err == nil {
		fmt.Printf("\tRemCtrlStatus:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPowerRequest, 1); err == nil {
		fmt.Printf("\tPowerRequest:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(sgRegPowerFlag, 1); err == nil {
		fmt.Printf("\tPowerFlag:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
