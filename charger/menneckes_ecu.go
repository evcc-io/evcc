package charger

// LICENSE

// Copyright (c) 2019-2022 andig

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
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://www.chargeupyourday.de/fileadmin/07_wissen/01_anwendungsfaelle/05_kompatible_energie_management_systeme/ECU_modbus_tcp_server_spec_rev_1.05.pdf

// Menneckes ECU-BRx and ECU-BBx (ChargeControl) charger implementation
type ChargeControl struct {
	conn *modbus.Connection
	curr uint16
}

const (
	chargeControlRegFirmware       = 100
	chargeControlRegStatus         = 122
	chargeControlRegChargedEnergy  = 716 // FW 5.22
	chargeControlRegChargeDuration = 718 // FW 5.22
	chargeControlRegHemsCurrent    = 1000
)

var (
	chargeControlRegEnergies = []uint16{200, 202, 204}
	chargeControlRegPowers   = []uint16{206, 208, 210}
	chargeControlRegCurrents = []uint16{212, 214, 216}
)

func init() {
	registry.Add("menneckes-chargecontrol", NewMenneckesChargeControlFromConfig)
	registry.Add("menneckes-eco", NewMenneckesChargeControlFromConfig)
}

// NewMenneckesChargeControlFromConfig creates a Mennekes ChargeControl charger from generic config
func NewMenneckesChargeControlFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 0x1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewChargeControl(cc.URI, cc.ID)
}

// NewChargeControl creates ChargeControl charger
func NewChargeControl(uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("mnks-cc")
	conn.Logger(log.TRACE)

	wb := &ChargeControl{
		conn: conn,
		curr: 6,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *ChargeControl) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargeControlRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 4:
		return api.StatusD, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[1])
	}
}

// Enabled implements the api.Charger interface
func (wb *ChargeControl) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargeControlRegHemsCurrent, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u != 0, nil
}

// Enable implements the api.Charger interface
func (wb *ChargeControl) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(chargeControlRegHemsCurrent, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *ChargeControl) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	cur := uint16(current)

	_, err := wb.conn.WriteSingleRegister(chargeControlRegHemsCurrent, cur)
	if err == nil {
		wb.curr = cur
	}

	return err
}

var _ api.Meter = (*ChargeControl)(nil)

// CurrentPower implements the api.Meter interface
func (wb *ChargeControl) CurrentPower() (float64, error) {
	var sum float64
	for idx, reg := range chargeControlRegPowers {
		b, err := wb.conn.ReadHoldingRegisters(reg, 2)
		if err != nil {
			return 0, err
		}

		if idx == 0 || binary.BigEndian.Uint32(b) != math.MaxUint32 {
			sum += rs485.RTUIeee754ToFloat64(b)
		}
	}

	return sum, nil
}

var _ api.MeterEnergy = (*ChargeControl)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *ChargeControl) TotalEnergy() (float64, error) {
	var sum float64
	for idx, reg := range chargeControlRegEnergies {
		b, err := wb.conn.ReadHoldingRegisters(reg, 2)
		if err != nil {
			return 0, err
		}

		if idx == 0 || binary.BigEndian.Uint32(b) != math.MaxUint32 {
			sum += rs485.RTUIeee754ToFloat64(b)
		}
	}

	return sum / 1e3, nil
}

var _ api.MeterCurrent = (*Alfen)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *ChargeControl) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, reg := range chargeControlRegCurrents {
		b, err := wb.conn.ReadHoldingRegisters(reg, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, rs485.RTUIeee754ToFloat64(b)/1e3)
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.ChargeRater = (*Amtron)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *ChargeControl) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargeControlRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUIeee754ToFloat64(b) / 1e3, nil
}

var _ api.ChargeTimer = (*EVSEWifi)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *ChargeControl) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargeControlRegChargeDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

// TODO RFID and EVCCID

var _ api.Diagnosis = (*ChargeControl)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ChargeControl) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(chargeControlRegFirmware, 2); err == nil {
		fmt.Printf("Firmware: %s\n", b)
	}

	// if b, err := wb.conn.ReadHoldingRegisters(chargeControlRegPhases, 1); err == nil {
	// 	fmt.Printf("Phases: %d\n", binary.BigEndian.Uint16(b))
	// }

	// if b, err := wb.conn.ReadHoldingRegisters(chargeControlRegSerial, 2); err == nil {
	// 	fmt.Printf("Serial: %d\n", binary.LittleEndian.Uint32(b))
	// }
}
