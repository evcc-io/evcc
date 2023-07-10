package charger

// LICENSE

// Copyright (c) 2023 premultiply

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
)

// Weidmüller charger implementation
type Weidmüller struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	wmRegCarStatus    = 301  // GD_ID_EVCC_CAR_STATE CHAR
	wmRegEvccStatus   = 302  // GD_ID_EVCC_EVSE_STATE UINT16
	wmRegPhases       = 317  // GD_ID_EVCC_PHASES UINT16
	wmRegVoltages     = 400  // GD_ID_CM_VOLTAGE_PHASE UINT32
	wmRegCurrents     = 406  // GD_ID_CM_CURRENT_PHASE UINT32
	wmRegActivePower  = 418  // GD_ID_CM_ACTIVE_POWER UINT32
	wmRegTotalEnergy  = 457  // GD_ID_CM_CONSUMED_ENERGY_TOTAL_WH UINT64
	wmRegCurrentLimit = 702  // GD_ID_AUT_USER_CURRENT_LIMIT UINT16
	wmRegCardId       = 1000 // GD_ID_RFID_TAG_UID CHAR[21]
)

func init() {
	registry.Add("weidmüller", NewWeidmüllerFromConfig)
}

// NewWeidmüllerFromConfig creates a Weidmüller charger from generic config
func NewWeidmüllerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWeidmüller(cc.URI, cc.ID)
}

// NewWeidmüller creates Weidmüller charger
func NewWeidmüller(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("weidmüller")
	conn.Logger(log.TRACE)

	wb := &Weidmüller{
		log:  log,
		conn: conn,
		curr: 6, // assume min current
	}

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr != 0 {
		wb.curr = curr
	}

	return wb, err
}

func (wb *Weidmüller) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(wmRegCurrentLimit, 1, b)

	return err
}

func (wb *Weidmüller) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// getPhaseValues returns 3 sequential register values
func (wb *Weidmüller) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 1e3
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Weidmüller) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegCarStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := string(b[0]); s {
	case "A", "B", "C":
		return api.ChargeStatus(s), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Weidmüller) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr != 0, err
}

// Enable implements the api.Charger interface
func (wb *Weidmüller) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *Weidmüller) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	wb.curr = uint16(current * 10)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*Weidmüller)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Weidmüller) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

var _ api.MeterEnergy = (*Weidmüller)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Weidmüller) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

var _ api.PhaseCurrents = (*Weidmüller)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Weidmüller) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(wmRegCurrents)
}

var _ api.PhaseVoltages = (*Weidmüller)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Weidmüller) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(wmRegVoltages)
}

var _ api.Identifier = (*Weidmüller)(nil)

// Identify implements the api.Identifier interface
func (wb *Weidmüller) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegCardId, 11)
	if err != nil {
		return "", err
	}
	return bytesAsString(b), nil
}

var _ api.PhaseSwitcher = (*Weidmüller)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Weidmüller) Phases1p3p(phases int) error {
	b := make([]byte, 2)

	if phases == 3 {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(wmRegPhases, 1, b)

	return err
}

var _ api.Diagnosis = (*Weidmüller)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Weidmüller) Diagnose() {

}
