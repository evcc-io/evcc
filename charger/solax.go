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

// Solax charger implementation
type Solax struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	// holding (FC 0x03, 0x06, 0x10)
	solaxRegDeviceMode      = 0x060D // uint16
	solaxRegStartChargeMode = 0x0610 // uint16
	solaxRegPhases          = 0x0625 // uint16
	solaxRegCommandControl  = 0x0627 // uint16
	solaxRegMaxCurrent      = 0x0628 // uint16 0.01A

	// input (FC 0x04)
	solaxRegVoltages    = 0x0000 // 3x uint16 0.01V
	solaxRegCurrents    = 0x0004 // 3x uint16 0.01A
	solaxRegActivePower = 0x000B // uint16 1W
	solaxRegTotalEnergy = 0x0010 // uint32s 0.1kWh
	solaxRegState       = 0x001D // uint16
)

func init() {
	registry.Add("solax", NewSolaxFromConfig)
}

// NewSolaxFromConfig creates a Solax charger from generic config
func NewSolaxFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSolax(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewSolax creates Solax charger
func NewSolax(uri, device, comset string, baudrate int, proto modbus.Protocol, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("solax")
	conn.Logger(log.TRACE)

	wb := &Solax{
		log:  log,
		conn: conn,
		curr: 600, // assume min current
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

func (wb *Solax) setCurrent(current uint16) error {
	_, err := wb.conn.WriteSingleRegister(solaxRegMaxCurrent, current)

	return err
}

func (wb *Solax) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(solaxRegMaxCurrent, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// getPhaseValues returns 3 sequential register values
func (wb *Solax) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / 100
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Solax) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := encoding.Uint16(b); s {
	case
		0, // "Available"
		5: // "Unavailable"
		return api.StatusA, nil
	case
		1, // "Preparing"
		8, // "SuspendedEVSE"
		7, // "SuspendedEV"
		3: // "Finishing"
		return api.StatusB, nil
	case
		2: // "Charging"
		return api.StatusC, nil
	case
		6, // "Reserved"
		4: // "Faulted"
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Solax) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr != 0, err
}

// Enable implements the api.Charger interface
func (wb *Solax) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *Solax) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Solax)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Solax) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	wb.curr = uint16(current * 100)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*Solax)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Solax) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegActivePower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), err
}

var _ api.MeterEnergy = (*Solax)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Solax) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

var _ api.PhaseCurrents = (*Solax)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Solax) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(solaxRegCurrents)
}

var _ api.PhaseVoltages = (*Solax)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Solax) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(solaxRegVoltages)
}

var _ api.PhaseSwitcher = (*Solax)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Solax) Phases1p3p(phases int) error {
	var u uint16

	if phases == 1 {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(solaxRegPhases, u)

	return err
}
