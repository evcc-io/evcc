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
	"github.com/volkszaehler/mbmd/encoding"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://update.mennekes.de/hcc3/1.13/Description%20Modbus_AMTRON%20HCC3_v01_2021-06-25_en.pdf

// Amtron Xtra/Premium charger implementation
type Amtron struct {
	conn *modbus.Connection
	curr uint16
}

const (
	amtronRegStatus     = 0x0302
	amtronRegPhases     = 0x0308
	amtronRegSerial     = 0x030B
	amtronRegEnergy     = 0x030D
	amtronRegName       = 0x0311
	amtronRegPower      = 0x030F
	amtronRegAmpsConfig = 0x0400
)

func init() {
	registry.Add("amtron", NewAmtronFromConfig)
}

// NewAmtronFromConfig creates a Mennekes Amtron charger from generic config
func NewAmtronFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAmtron(cc.URI, cc.ID)
}

// NewAmtron creates Amtron charger
func NewAmtron(uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("amtron")
	conn.Logger(log.TRACE)

	wb := &Amtron{
		conn: conn,
		curr: 6,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Amtron) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(amtronRegStatus, 1)
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
func (wb *Amtron) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(amtronRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Amtron) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(amtronRegAmpsConfig, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Amtron) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	cur := uint16(current)

	_, err := wb.conn.WriteSingleRegister(amtronRegAmpsConfig, cur)
	if err == nil {
		wb.curr = cur
	}

	return err
}

var _ api.Meter = (*Amtron)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Amtron) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(amtronRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b), nil
}

var _ api.ChargeRater = (*Amtron)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Amtron) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(amtronRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 1e3, nil
}

var _ api.Diagnosis = (*Amtron)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Amtron) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(amtronRegName, 11); err == nil {
		fmt.Printf("Name: %s\n", encoding.StringLsbFirst(b))
	}

	if b, err := wb.conn.ReadInputRegisters(amtronRegPhases, 1); err == nil {
		fmt.Printf("Phases: %d\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadInputRegisters(amtronRegSerial, 2); err == nil {
		fmt.Printf("Serial: %d\n", binary.LittleEndian.Uint32(b))
	}
}
