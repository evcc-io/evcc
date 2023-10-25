package charger

// LICENSE

// Copyright (c) 2022 premultiply, andig

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

// ABB charger implementation
type ABB struct {
	conn *modbus.Connection
	curr uint32
}

const (
	abbRegSerial     = 0x4000 // Serial Number 4 unsigned RO available
	abbRegFirmware   = 0x4004 // Firmware version 2 unsigned RO available
	abbRegMaxRated   = 0x4006 // Max rated current 2 unsigned RO available
	abbRegErrorCode  = 0x4008 // Error Code 2 unsigned RO available
	abbRegSocketLock = 0x400A // Socket Lock State 2 unsigned RO available
	abbRegStatus     = 0x400C // Charging state 2 unsigned RO available
	abbRegGetCurrent = 0x400E // Current charging current limit 2 0.001 A unsigned RO
	abbRegCurrents   = 0x4010 // Charging current phases 6 0.001 A unsigned RO available
	abbRegVoltages   = 0x4016 // Voltage phases 6 0.1 V unsigned RO available
	abbRegPower      = 0x401C // Active power 2 1 W unsigned RO available
	abbRegEnergy     = 0x401E // Energy delivered in charging session 2 1 Wh unsigned RO available
	abbRegSetCurrent = 0x4100 // Set charging current limit 2 0.001 A unsigned WO available
	// abbRegSession    = 0x4105 // Start/Stop Charging Session 1 unsigned WO available
	// abbRegPhases     = 0x4102 // Set charging phase 1 unsigned WO Not supported
)

func init() {
	registry.Add("abb", NewABBFromConfig)
}

// NewABBFromConfig creates a ABB charger from generic config
func NewABBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewABB(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewABB creates ABB charger
func NewABB(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("abb")
	conn.Logger(log.TRACE)

	wb := &ABB{
		conn: conn,
		curr: 6000, // assume min current
	}

	// keep-alive
	go func() {
		for range time.Tick(30 * time.Second) {
			_, _ = wb.status()
		}
	}()

	return wb, err
}

func (wb *ABB) status() (byte, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegStatus, 2)
	if err != nil {
		return 0, err
	}

	return b[2] & 0x7f, nil
}

// Status implements the api.Charger interface
func (wb *ABB) Status() (api.ChargeStatus, error) {
	s, err := wb.status()
	if err != nil {
		return api.StatusNone, err
	}

	switch s {
	case 0: // State A: Idle
		return api.StatusA, nil
	case 1: // State B1: EV Plug in, pending authorization
		return api.StatusB, nil
	case 2: // State B2: EV Plug in, EVSE ready for charging(PWM)
		return api.StatusB, nil
	case 3: // State C1: EV Ready for charge, S2 closed(no PWM)
		return api.StatusB, nil
	case 4: // State C2: Charging Contact closed, energy delivering
		return api.StatusC, nil
	case 5: // Other: Session stopped
		b, err := wb.conn.ReadHoldingRegisters(abbRegSocketLock, 2)
		if err != nil {
			return api.StatusNone, err
		}
		if binary.BigEndian.Uint32(b) >= 0x0101 {
			return api.StatusB, nil
		}
		return api.StatusA, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %0x", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *ABB) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegGetCurrent, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *ABB) Enable(enable bool) error {
	var current uint32
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// setCurrent writes the current limit in mA
func (wb *ABB) setCurrent(current uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, current)

	_, err := wb.conn.WriteMultipleRegisters(abbRegSetCurrent, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *ABB) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*ABB)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *ABB) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	wb.curr = uint32(current * 1e3)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*ABB)(nil)

// CurrentPower implements the api.Meter interface
func (wb *ABB) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.ChargeRater = (*ABB)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *ABB) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

// getPhaseValues returns 3 sequential register values
func (wb *ABB) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*ABB)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *ABB) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(abbRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*ABB)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *ABB) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(abbRegVoltages, 10)
}

// var _ api.PhaseSwitcher = (*ABB)(nil)

// // Phases1p3p implements the api.PhaseSwitcher interface
// func (wb *ABB) Phases1p3p(phases int) error {
// 	var b uint16 = 1
// 	if phases != 1 {
// 		b = 2 // 3p
// 	}

// 	_, err := wb.conn.WriteSingleRegister(abbRegPhases, b)
// 	return err
// }

var _ api.Diagnosis = (*ABB)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ABB) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(abbRegSerial, 4); err == nil {
		fmt.Printf("\tSerial:\t%x\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegFirmware, 2); err == nil {
		fmt.Printf("\tFirmware:\t%d.%d.%d\n", b[0], b[1], b[2])
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegMaxRated, 2); err == nil {
		fmt.Printf("\tMax rated current:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegGetCurrent, 2); err == nil {
		fmt.Printf("\tCharging current limit:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegSocketLock, 2); err == nil {
		fmt.Printf("\tSocket lock state:\t%x\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegStatus, 2); err == nil {
		fmt.Printf("\tStatus:\t%x\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegErrorCode, 2); err == nil {
		fmt.Printf("\tError code:\t%x\n", binary.BigEndian.Uint32(b))
	}
}
