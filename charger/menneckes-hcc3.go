package charger

// LICENSE

// Copyright (c) 2019-2022 andig
// Copyright (c) 2022 premultiply

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
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://update.mennekes.de/hcc3/1.13/Description%20Modbus_AMTRON%20HCC3_v01_2021-06-25_en.pdf

// MenneckesHcc3 Xtra/Premium charger implementation
type MenneckesHcc3 struct {
	conn *modbus.Connection
	curr uint16
}

const (
	menneckesHcc3RegStatus     = 0x0302
	menneckesHcc3RegPhases     = 0x0308
	menneckesHcc3RegSerial     = 0x030B
	menneckesHcc3RegEnergy     = 0x030D
	menneckesHcc3RegName       = 0x0311
	menneckesHcc3RegPower      = 0x030F
	menneckesHcc3RegAmpsConfig = 0x0400
)

func init() {
	registry.Add("menneckes-hcc3", NewMenneckesHcc3FromConfig)
}

// NewMenneckesHcc3FromConfig creates a Mennekes menneckesHcc3 charger from generic config
func NewMenneckesHcc3FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMenneckesHcc3(cc.URI, cc.ID)
}

// NewMenneckesHcc3 creates Menneckes HCC3 charger
func NewMenneckesHcc3(uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("menneckes-hcc3")
	conn.Logger(log.TRACE)

	wb := &MenneckesHcc3{
		conn: conn,
		curr: 6,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *MenneckesHcc3) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(menneckesHcc3RegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 1, 2:
		return api.StatusA, nil
	case 3, 4:
		return api.StatusB, nil
	case 5, 6:
		return api.StatusC, nil
	case 7, 8:
		return api.StatusD, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[1])
	}
}

// Enabled implements the api.Charger interface
func (wb *MenneckesHcc3) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(menneckesHcc3RegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u != 0, nil
}

// Enable implements the api.Charger interface
func (wb *MenneckesHcc3) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(menneckesHcc3RegAmpsConfig, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *MenneckesHcc3) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	cur := uint16(current)

	_, err := wb.conn.WriteSingleRegister(menneckesHcc3RegAmpsConfig, cur)
	if err == nil {
		wb.curr = cur
	}

	return err
}

var _ api.Meter = (*MenneckesHcc3)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MenneckesHcc3) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(menneckesHcc3RegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b), nil
}

var _ api.ChargeRater = (*MenneckesHcc3)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *MenneckesHcc3) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(menneckesHcc3RegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, nil
}

var _ api.Diagnosis = (*MenneckesHcc3)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *MenneckesHcc3) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(menneckesHcc3RegName, 11); err == nil {
		fmt.Printf("Name: %s\n", encoding.StringLsbFirst(b))
	}

	if b, err := wb.conn.ReadInputRegisters(menneckesHcc3RegPhases, 1); err == nil {
		fmt.Printf("Phases: %d\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadInputRegisters(menneckesHcc3RegSerial, 2); err == nil {
		fmt.Printf("Serial: %d\n", binary.LittleEndian.Uint32(b))
	}
}
