package charger

// LICENSE

// Copyright (c) 2023 andig

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
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/spf13/cast"
)

// Victron charger implementation
type Victron struct {
	conn *modbus.Connection
	regs victronRegs
}

type victronRegs struct {
	Mode       uint16
	Energy     uint16
	Power      uint16
	Status     uint16
	SetCurrent uint16
	Enabled    uint16
	isGX       bool
}

var (
	victronGX = victronRegs{
		Mode:       3815,
		Energy:     3816,
		Power:      3821,
		Status:     3824,
		SetCurrent: 3825,
		Enabled:    3826,
		isGX:       true,
	}

	victronEVCS = victronRegs{
		Mode:       5009,
		Energy:     5021,
		Power:      5014,
		Status:     5015,
		SetCurrent: 5016,
		Enabled:    5010,
		isGX:       false,
	}
)

func init() {
	registry.Add("victron", NewVictronGXFromConfig)
	registry.Add("victron-evcs", NewVictronEVCSFromConfig)
}

// NewVictronGXFromConfig creates a ABB charger from generic config
func NewVictronGXFromConfig(other map[string]interface{}) (api.Charger, error) {
	return NewVictronFromConfig(other, victronGX)
}

// NewVictronEVCSFromConfig creates a ABB charger from generic config
func NewVictronEVCSFromConfig(other map[string]interface{}) (api.Charger, error) {
	return NewVictronFromConfig(other, victronEVCS)
}

// NewVictronFromConfig creates a ABB charger from generic config
func NewVictronFromConfig(other map[string]interface{}, regs victronRegs) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: cast.ToUint8(regs.isGX) * 100,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVictron(cc.URI, cc.ID, victronEVCS)
}

// NewVictron creates Victron charger
func NewVictron(uri string, slaveID uint8, regs victronRegs) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("victron")
	conn.Logger(log.TRACE)

	wb := &Victron{
		conn: conn,
		regs: regs,
	}

	b, err := wb.conn.ReadHoldingRegisters(wb.regs.Mode, 1)
	if err != nil {
		return nil, err
	}

	if binary.BigEndian.Uint16(b) != 0 {
		return nil, errors.New("charger must be in manual mode")
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Victron) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.regs.Status, 1)
	if err != nil {
		return api.StatusNone, err
	}

	u := binary.BigEndian.Uint16(b)
	switch u {
	case 0, 1, 2, 3:
		return api.ChargeStatusString(string('A' + rune(binary.BigEndian.Uint16(b))))
	case 5, 6, 21:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *Victron) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.regs.Enabled, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Victron) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(wb.regs.Enabled, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Victron) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(wb.regs.SetCurrent, uint16(current))
	return err
}

var _ api.Meter = (*Victron)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Victron) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.regs.Power, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.ChargeRater = (*Victron)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Victron) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.regs.Energy, cast.ToUint16(wb.regs.isGX))
	if err != nil {
		return 0, err
	}

	if wb.regs.isGX {
		return float64(binary.BigEndian.Uint32(b)) / 100, nil
	}

	return float64(binary.BigEndian.Uint16(b)) / 100, nil
}
