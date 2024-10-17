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

// MennekesHcc3 Xtra/Premium charger implementation
type MennekesHcc3 struct {
	conn *modbus.Connection
	curr uint16
}

const (
	mennekesHcc3RegStatus     = 0x0302
	mennekesHcc3RegPhases     = 0x0308
	mennekesHcc3RegSerial     = 0x030B
	mennekesHcc3RegEnergy     = 0x030D
	mennekesHcc3RegName       = 0x0311
	mennekesHcc3RegPower      = 0x030F
	mennekesHcc3RegAmpsConfig = 0x0400
)

func init() {
	registry.Add("mennekes-hcc3", NewMennekesHcc3FromConfig)
}

// NewMennekesHcc3FromConfig creates a Mennekes mennekesHcc3 charger from generic config
func NewMennekesHcc3FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMennekesHcc3(cc.URI, cc.ID)
}

// NewMennekesHcc3 creates Mennekes HCC3 charger
func NewMennekesHcc3(uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("mennekes-hcc3")
	conn.Logger(log.TRACE)

	wb := &MennekesHcc3{
		conn: conn,
		curr: 6,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *MennekesHcc3) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(mennekesHcc3RegStatus, 1)
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
func (wb *MennekesHcc3) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesHcc3RegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u != 0, nil
}

// Enable implements the api.Charger interface
func (wb *MennekesHcc3) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(mennekesHcc3RegAmpsConfig, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *MennekesHcc3) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	cur := uint16(current)

	_, err := wb.conn.WriteSingleRegister(mennekesHcc3RegAmpsConfig, cur)
	if err == nil {
		wb.curr = cur
	}

	return err
}

var _ api.Meter = (*MennekesHcc3)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MennekesHcc3) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(mennekesHcc3RegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b), nil
}

var _ api.ChargeRater = (*MennekesHcc3)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *MennekesHcc3) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(mennekesHcc3RegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, nil
}

var _ api.Diagnosis = (*MennekesHcc3)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *MennekesHcc3) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(mennekesHcc3RegName, 11); err == nil {
		fmt.Printf("Name: %s\n", encoding.StringLsbFirst(b))
	}

	if b, err := wb.conn.ReadInputRegisters(mennekesHcc3RegPhases, 1); err == nil {
		fmt.Printf("Phases: %d\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadInputRegisters(mennekesHcc3RegSerial, 2); err == nil {
		fmt.Printf("Serial: %d\n", binary.LittleEndian.Uint32(b))
	}
}
