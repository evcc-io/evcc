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
	"errors"
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// Alfen charger implementation
type Alfen struct {
	conn *modbus.Connection
	// curr uint16
}

const (
	// alfenRegName         = 100
	// alfenRegManufacturer = 117
	// alfenRegFirmware     = 123
	alfenRegPower        = 344
	alfenRegAvailability = 1200
	alfenRegStatus       = 1201 // 5 registers
	alfenRegAmpsConfig   = 1210
	alfenRegPhases       = 1215
)

var alfenRegCurrents = []uint16{320, 322, 324}

func init() {
	registry.Add("alfen", NewAlfenFromConfig)
}

// NewAlfenFromConfig creates a Alfen charger from generic config
func NewAlfenFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAlfen(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewAlfen creates Alfen charger
func NewAlfen(uri, device, comset string, baudrate int, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.AsciiFormat, slaveID)
	if err != nil {
		return nil, err
	}

	// if !sponsor.IsAuthorized() {
	// 	return nil, api.ErrSponsorRequired
	// }

	log := util.NewLogger("alfen")
	conn.Logger(log.TRACE)

	wb := &Alfen{
		conn: conn,
		// curr: uint16(6 / 0.06),
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Alfen) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegStatus, 5)
	if err != nil {
		return api.StatusNone, err
	}
	_ = b

	// r := rune(b[1]>>4-0x0A) + 'A'

	// switch r {
	// case 'A', 'B', 'C':
	// 	return api.ChargeStatus(r), nil
	// default:
	// 	status, ok := ablStatus[b[1]]
	// 	if !ok {
	// 		status = string(r)
	// 	}

	// 	return api.StatusNone, fmt.Errorf("invalid status: %s", status)
	// }

	return api.StatusNone, fmt.Errorf("invalid status: %0x", b)
}

// Enabled implements the api.Charger interface
func (wb *Alfen) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegAvailability, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 0xFF, nil
}

// Enable implements the api.Charger interface
func (wb *Alfen) Enable(enable bool) error {
	return errors.New("not implemented")
}

// MaxCurrent implements the api.Charger interface
func (wb *Alfen) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Alfen)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *Alfen) MaxCurrentMillis(current float64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(float32(current)))

	_, err := wb.conn.WriteMultipleRegisters(alfenRegAmpsConfig, 2, b)

	return err
}

var _ api.Meter = (*Alfen)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Alfen) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegPower, 2)
	return rs485.RTUIeee754ToFloat64(b), err
}

var _ api.MeterCurrent = (*Alfen)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Alfen) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, reg := range alfenRegCurrents {
		b, err := wb.conn.ReadHoldingRegisters(reg, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, rs485.RTUIeee754ToFloat64(b))
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.ChargePhases = (*Alfen)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (c *Alfen) Phases1p3p(phases int) error {
	_, err := c.conn.WriteSingleRegister(alfenRegPhases, uint16(phases<<8))
	return err
}

// var _ api.Diagnosis = (*Alfen)(nil)

// // Diagnose implements the api.Diagnosis interface
// func (wb *Alfen) Diagnose() {
// 	b, err := wb.conn.ReadHoldingRegisters(ablRegFirmware, 2)
// 	if err == nil {
// 		fmt.Printf("Firmware: %0 x\n", b)
// 	}
// }
