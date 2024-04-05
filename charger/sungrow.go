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
	"github.com/volkszaehler/mbmd/encoding"
)

// Sungrow charger implementation
type Sungrow struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	// holding
	sgRegEnable      = 21210 // uint16
	sgRegMaxCurrent  = 21202 // uint16 0.01A
	sgRegPhases      = 21203 // uint16
	sgRegWorkingMode = 21262 // uint16

	// input
	sgRegState         = 21316 // uint16
	sgRegActivePower   = 21307 // uint16 1W
	sgRegChargedEnergy = 21309 // uint32 1Wh
	sgRegTotalEnergy   = 21299 // uint32 1Wh
	sgRegPhasesState   = 21269 // uint16
)

var (
	sgRegCurrents = []uint16{21301, 21303, 21305} // 3x uint16 0.1V
	sgRegVoltages = []uint16{21302, 21304, 21306} // 3x uint16 0.1A
)

func init() {
	registry.Add("sungrow", NewSungrowFromConfig)
}

// NewSungrowFromConfig creates a Sungrow charger from generic config
func NewSungrowFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
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

		res[i] = float64(binary.BigEndian.Uint16(b)) / divider
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Sungrow) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(sgRegState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := encoding.Uint16(b); s {
	case
		1, // "Available"
		8: // "Disabled"
		return api.StatusA, nil
	case
		2, // "Standby"
		4, // "SuspendedEVSE"
		5, // "SuspendedEV"
		6: // "Completed"
		return api.StatusB, nil
	case
		3: // "Charging"
		return api.StatusC, nil
	case
		7, // "Reserved"
		9: // "Faulted"
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Sungrow) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(sgRegEnable, 1)
	if err != nil {
		return false, err
	}

	return encoding.Uint16(b) == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Sungrow) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(sgRegEnable, u)

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

	_, err := wb.conn.WriteSingleRegister(sgRegMaxCurrent, uint16(current*10))

	return err
}

var _ api.Meter = (*Sungrow)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Sungrow) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(sgRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

var _ api.PhaseCurrents = (*Sungrow)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Sungrow) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(sgRegCurrents, 10)
}

var _ api.PhaseVoltages = (*Sungrow)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Sungrow) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(sgRegVoltages, 1)
}

var _ api.MeterEnergy = (*Sungrow)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Sungrow) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(sgRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

var _ api.PhaseSwitcher = (*Sungrow)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Sungrow) Phases1p3p(phases int) error {
	var u uint16

	if phases == 1 {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(sgRegPhases, u)

	return err
}
