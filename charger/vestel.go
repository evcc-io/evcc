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
	vestelRegSerial          = 100 // 25
	vestelRegBrand           = 190 // 10
	vestelRegModel           = 210 // 5
	vestelRegFirmware        = 230 // 50
	vestelRegChargeStatus    = 1001
	vestelRegCableStatus     = 1004
	vestelRegChargeTime      = 1508
	vestelRegMaxCurrent      = 5004
	vestelRegPower           = 1020
	vestelRegTotalEnergy     = 1036
	vestelRegSessionEnergy   = 1502
	vestelRegFailsafeTimeout = 2002
	vestelRegAlive           = 6000
	// vestelRegChargepointState = 1000
)

var (
	vestelRegCurrents = []uint16{1008, 1010, 1012} // non-continuous uint16 registers!
	vestelRegVoltages = []uint16{1014, 1016, 1018} // non-continuous uint16 registers!
)

// Vestel is an api.Charger implementation for Vestel/Hymes wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Vestel struct {
	log     *util.Logger
	conn    *modbus.Connection
	enabled bool
	current uint16
}

func init() {
	registry.Add("vestel", NewVestelFromConfig)
}

// NewVestelFromConfig creates a Vestel charger from generic config
func NewVestelFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVestel(cc.URI, cc.ID)
}

// NewVestel creates a Vestel charger
func NewVestel(uri string, id uint8) (*Vestel, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("vestel")
	conn.Logger(log.TRACE)

	wb := &Vestel{
		log:     log,
		conn:    conn,
		current: 6,
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadInputRegisters(vestelRegFailsafeTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	timeout := 10 * time.Second
	if u := binary.BigEndian.Uint16(b); u > 0 {
		timeout = time.Duration(u) * time.Second / 4
	}
	if timeout < 3*time.Second {
		timeout = 3 * time.Second
	}
	go wb.heartbeat(timeout)

	return wb, nil
}

func (wb *Vestel) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.WriteSingleRegister(vestelRegAlive, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Vestel) Status() (api.ChargeStatus, error) {
	res := api.StatusA

	b, err := wb.conn.ReadInputRegisters(vestelRegCableStatus, 1)
	if err == nil && binary.BigEndian.Uint16(b) >= 2 {
		res = api.StatusB

		b, err = wb.conn.ReadInputRegisters(vestelRegChargeStatus, 1)
		if err == nil && binary.BigEndian.Uint16(b) == 1 {
			res = api.StatusC
		}
	}

	return res, err
}

// Enabled implements the api.Charger interface
func (wb *Vestel) Enabled() (bool, error) {
	return verifyEnabled(wb, wb.enabled)

	// b, err := wb.conn.ReadHoldingRegisters(vestelRegMaxCurrent, 1)
	// if err != nil {
	// 	return false, err
	// }

	// return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Vestel) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.current
	}

	_, err := wb.conn.WriteSingleRegister(vestelRegMaxCurrent, u)
	if err == nil {
		wb.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Vestel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current)
	_, err := wb.conn.WriteSingleRegister(vestelRegMaxCurrent, u)
	if err == nil {
		wb.current = u
	}

	return err
}

var _ api.ChargeTimer = (*Vestel)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Vestel) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.Meter = (*Vestel)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Vestel) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*Vestel)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Vestel) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

var _ api.ChargeRater = (*Vestel)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Vestel) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegSessionEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

// getPhaseValues returns 3 sequential register values
func (wb *Vestel) getPhaseValues(regs []uint16, divider float64) (float64, float64, float64, error) {
	var res [3]float64
	for i, reg := range regs {
		b, err := wb.conn.ReadInputRegisters(reg, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = float64(binary.BigEndian.Uint16(b)) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Vestel)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Vestel) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(vestelRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*Vestel)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Vestel) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(vestelRegVoltages, 1)
}

var _ api.Diagnosis = (*Vestel)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Vestel) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(vestelRegBrand, 10); err == nil {
		fmt.Printf("Brand:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegModel, 5); err == nil {
		fmt.Printf("Model:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegSerial, 25); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegFirmware, 50); err == nil {
		fmt.Printf("Firmware:\t%s\n", b)
	}
}
