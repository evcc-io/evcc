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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Victron charger implementation
type Victron struct {
	conn *modbus.Connection
}

const (
	victronRegMode       = 3815
	victronRegEnergy     = 3816
	victronRegPower      = 3821
	victronRegStatus     = 3824
	victronRegSetCurrent = 3825
	victronRegEnabled    = 3826
)

func init() {
	registry.Add("victron", NewVictronFromConfig)
}

// NewVictronFromConfig creates a ABB charger from generic config
func NewVictronFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 100,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVictron(cc.URI, cc.ID)
}

// NewVictron creates Victron charger
func NewVictron(uri string, slaveID uint8) (api.Charger, error) {
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
	}

	b, err := wb.conn.ReadHoldingRegisters(victronRegMode, 1)
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
	b, err := wb.conn.ReadHoldingRegisters(victronRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string('A' + rune(binary.BigEndian.Uint16(b))))
}

// Enabled implements the api.Charger interface
func (wb *Victron) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronRegEnabled, 1)
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

	_, err := wb.conn.WriteSingleRegister(victronRegEnabled, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Victron) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(victronRegSetCurrent, uint16(current))
	return err
}

var _ api.Meter = (*Victron)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Victron) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.ChargeRater = (*Victron)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Victron) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 100, nil
}
