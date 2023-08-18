package charger

// LICENSE

// Copyright (c) 2019-2023 andig

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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Schneider charger implementation
type Schneider struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	schneiderRegEvState       = 1
	schneiderRegCurrents      = 2999
	schneiderRegVoltages      = 3027
	schneiderRegPower         = 3059
	schneiderRegEnergy        = 3203
	schneiderRegLifebit       = 4000
	schneiderRegSetCommand    = 4001
	schneiderRegSetPoint      = 4004
	schneiderRegChargingTime  = 4007
	schneiderRegSessionEnergy = 4012
)

func init() {
	registry.Add("schneider", NewSchneiderFromConfig)
}

// https://download.schneider-electric.com/files?p_enDocType=Other+technical+guide&p_File_Name=GEX1969300-04.pdf&p_Doc_Ref=GEX1969300

// NewSchneiderFromConfig creates a Schneider charger from generic config
func NewSchneiderFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Timeout         time.Duration
	}{
		Settings: modbus.Settings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSchneider(cc.URI, cc.ID, cc.Timeout)
}

// NewSchneider creates Schneider charger
func NewSchneider(uri string, id uint8, timeout time.Duration) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if timeout > 0 {
		conn.Timeout(timeout)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("schneider")
	conn.Logger(log.TRACE)

	wb := &Schneider{
		log:  log,
		conn: conn,
		curr: 6,
	}

	// heartbeat
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegLifebit, 1)
	if err != nil {
		return nil, fmt.Errorf("heartbeat timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u != 2 {
		go wb.heartbeat(time.Duration(2) * time.Second)
	}

	return wb, nil
}

func (wb *Schneider) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.WriteSingleRegister(schneiderRegLifebit, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Schneider) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegEvState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	switch s {
	case 0, 2, 6:
		return api.StatusA, nil
	case 1, 3, 4, 5, 7:
		return api.StatusB, nil
	case 8, 9:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Schneider) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegSetPoint, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 99, nil
}

// Enable implements the api.Charger interface
func (wb *Schneider) Enable(enable bool) error {
	var u uint16 = 99
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(schneiderRegSetPoint, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Schneider) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(schneiderRegSetPoint, uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *Schneider) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32LswFirst(b)) * 1000, nil
}

var _ api.MeterEnergy = (*Schneider)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Schneider) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	// return float64(encoding.Uint64LswFirst(b)) / 1000, nil
	return float64(uint64(b[6])<<56|uint64(b[7])<<48|uint64(b[4])<<40|uint64(b[5])<<32|uint64(b[2])<<24|uint64(b[3])<<16|uint64(b[0])<<8|uint64(b[1])) / 1000, nil
}

var _ api.PhaseCurrents = (*Schneider)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Schneider) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegCurrents, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res []float64
	for i := 0; i < 3; i++ {
		res = append(res, float64(encoding.Float32LswFirst(b[4*i:])))
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*Schneider)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Schneider) Voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegVoltages, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res []float64
	for i := 0; i < 3; i++ {
		fmt.Println(binary.BigEndian.Uint32(b[4*i:]))
		res = append(res, float64(encoding.Float32LswFirst(b[4*i:])))
	}

	return res[0], res[1], res[2], nil
}
