package charger

// LICENSE

// Copyright (c) 2019-2021 andig

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

// https://github.com/evcc-io/evcc/discussions/1965

// Amtron charger implementation
type Amtron struct {
	conn *modbus.Connection
}

const (
	amtronRegEnergy     = 0x030D
	amtronRegPower      = 0x030F
	amtronRegStatus     = 0x0312
	amtronRegEnabled    = 0x0401
	amtronRegAmpsConfig = 0x0400
	amtronRegSerial     = 0x030B
	amtronRegName       = 0x0311
)

func init() {
	registry.Add("amtron", NewAmtronFromConfig)
}

// NewAmtronFromConfig creates a Mennekes Amtron charger from generic config
func NewAmtronFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 0xff,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAmtron(cc.URI, cc.Device, "", 0, cc.ID)
}

// NewAmtron creates Amtron charger
func NewAmtron(uri, device, comset string, baudrate int, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("amtron")
	conn.Logger(log.TRACE)

	wb := &Amtron{
		conn: conn,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Amtron) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(amtronRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch b[0] {
	case 1, 2:
		return api.StatusA, nil
	case 3, 4:
		return api.StatusB, nil
	case 5, 6:
		// TODO check C1 -> B?
		return api.StatusC, nil
	case 7, 8:
		return api.StatusD, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[:1])
	}
}

// Enabled implements the api.Charger interface
func (wb *Amtron) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(amtronRegEnabled, 1)
	if err != nil {
		return false, err
	}

	var res bool
	switch b[0] {
	case 0, 4:
		res = true
	}

	return res, nil
}

// Enable implements the api.Charger interface
func (wb *Amtron) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		b[0] = 0x04
	}

	_, err := wb.conn.WriteMultipleRegisters(amtronRegEnabled, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Amtron) MaxCurrent(current int64) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(amtronRegAmpsConfig, 1, b)

	return err
}

var _ api.Meter = (*Amtron)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Amtron) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(amtronRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.LittleEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*Amtron)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Amtron) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(amtronRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.LittleEndian.Uint32(b)) / 1e3, err
}

// var _ api.ChargePhases = (*Amtron)(nil)

// // Phases1p3p implements the api.ChargePhases interface
// func (c *Amtron) Phases1p3p(phases int) error {
// 	_, err := c.conn.WriteSingleRegister(amtronRegPhases, uint16(phases))
// 	return err
// }

var _ api.Diagnosis = (*Amtron)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Amtron) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(amtronRegName, 11); err == nil {
		fmt.Printf("Name: %s\n", string(b))
	}

	if b, err := wb.conn.ReadInputRegisters(amtronRegSerial, 2); err == nil {
		fmt.Printf("Serial: %d\n", binary.LittleEndian.Uint32(b))
	}
}
