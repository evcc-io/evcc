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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	etrelRegSerial          = 100
	etrelRegBrand           = 190
	etrelRegModel           = 210
	etrelRegFirmware        = 230
	etrelRegChargeStatus    = 1001
	etrelRegCableStatus     = 1004
	etrelRegChargeTime      = 1508
	etrelRegMaxCurrent      = 5004
	etrelRegPower           = 1020
	etrelRegTotalEnergy     = 1036
	etrelRegSessionEnergy   = 1502
	etrelRegFailsafeTimeout = 2002
	etrelRegAlive           = 6000
)

var etrelRegCurrents = []uint16{1008, 1010, 1012}

// Etrel is an api.ChargeController implementation for Etrel/Sonnen wallboxes
type Etrel struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
}

func init() {
	registry.Add("etrel", NewEtrelFromConfig)
}

// NewEtrelFromConfig creates a Vestel charger from generic config
func NewEtrelFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEtrel(cc.URI, cc.ID)
}

// NewEtrel creates a Vestel charger
func NewEtrel(uri string, id uint8) (*Etrel, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("etrel")
	conn.Logger(log.TRACE)

	wb := &Etrel{
		log:     log,
		conn:    conn,
		current: 6,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Etrel) Status() (api.ChargeStatus, error) {
	res := api.StatusA

	b, err := wb.conn.ReadInputRegisters(etrelRegCableStatus, 1)
	if err == nil && binary.BigEndian.Uint16(b) > 0 {
		res = api.StatusB

		b, err = wb.conn.ReadInputRegisters(etrelRegChargeStatus, 1)
		if err == nil && binary.BigEndian.Uint16(b) > 0 {
			res = api.StatusC
		}
	}

	return res, err
}

// Enabled implements the api.Charger interface
func (wb *Etrel) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(etrelRegMaxCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Etrel) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.current
	}

	_, err := wb.conn.WriteSingleRegister(etrelRegMaxCurrent, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Etrel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current)
	_, err := wb.conn.WriteSingleRegister(etrelRegMaxCurrent, u)
	if err == nil {
		wb.current = u
	}

	return err
}

var _ api.ChargeTimer = (*Etrel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Etrel) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	secs := binary.BigEndian.Uint32(b)
	return time.Duration(time.Duration(secs) * time.Second), nil
}

var _ api.Meter = (*Etrel)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Etrel) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*Etrel)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Etrel) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

var _ api.ChargeRater = (*Etrel)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Etrel) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(etrelRegSessionEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

var _ api.MeterCurrent = (*Etrel)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Etrel) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range etrelRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(binary.BigEndian.Uint16(b))/1e3)
	}

	return currents[0], currents[1], currents[2], nil
}

// Diagnose implements the Diagnosis interface
func (wb *Etrel) Diagnose() {
	// if b, err := wb.conn.ReadInputRegisters(etrelRegBrand, 10); err == nil {
	// 	fmt.Printf("Brand:\t%s\n", b)
	// }
	// if b, err := wb.conn.ReadInputRegisters(etrelRegModel, 5); err == nil {
	// 	fmt.Printf("Model:\t%s\n", b)
	// }
	// if b, err := wb.conn.ReadInputRegisters(etrelRegSerial, 25); err == nil {
	// 	fmt.Printf("Serial:\t%s\n", b)
	// }
	// if b, err := wb.conn.ReadInputRegisters(etrelRegFirmware, 50); err == nil {
	// 	fmt.Printf("Firmware:\t%s\n", b)
	// }
}
