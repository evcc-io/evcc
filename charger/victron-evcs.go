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
)

// Victron charger implementation
type VictronEVCS struct {
	conn *modbus.Connection
}

const (
	victronEvcsRegMode       = 5009
	victronEvcsRegEnergy     = 5021
	victronEvcsRegPower      = 5014
	victronEvcsRegStatus     = 5015
	victronEvcsRegSetCurrent = 5016
	victronEvcsRegEnabled    = 5010
)

func init() {
	registry.Add("victron-evcs", NewVictronEVCSFromConfig)
}

// NewVictronEVCSFromConfig creates Victron EVCS charger
func NewVictronEVCSFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVictronEVCS(cc.URI, cc.ID)
}

// NewVictronEVCS creates Victron EVCS charger
func NewVictronEVCS(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("victron-evcs")
	conn.Logger(log.TRACE)

	wb := &VictronEVCS{
		conn: conn,
	}

	b, err := wb.conn.ReadHoldingRegisters(victronEvcsRegMode, 1)
	if err != nil {
		return nil, err
	}

	if binary.BigEndian.Uint16(b) != 0 {
		return nil, errors.New("charger must be in manual mode")
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *VictronEVCS) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronEvcsRegStatus, 1)
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
func (wb *VictronEVCS) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronEvcsRegEnabled, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *VictronEVCS) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(victronEvcsRegEnabled, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *VictronEVCS) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(victronEvcsRegSetCurrent, uint16(current))
	return err
}

var _ api.Meter = (*Victron)(nil)

// CurrentPower implements the api.Meter interface
func (wb *VictronEVCS) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronEvcsRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.ChargeRater = (*Victron)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *VictronEVCS) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(victronEvcsRegEnergy, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 100, nil
}
