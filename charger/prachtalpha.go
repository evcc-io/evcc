package charger

// LICENSE

// Copyright (c) 2019-2023 andig, premultiply

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

// PrachtAlpha charger implementation
type PrachtAlpha struct {
	conn              *modbus.Connection
	vehicle           uint16
	curr              uint16
	currentWorkaround bool
}

const (
	prachtTotalCurrent = 40003 - 40001 //   2 total limit of all connectors
	prachtConnCurrent  = 40004 - 40001 //   3 (+1 for second connector)
	prachtRfidCount    = 30075 - 30001 //  74
	prachtFirmwareRfid = 30101 - 30001 // 100
	prachtFirmwareMain = 30102 - 30001 // 101
	prachtEnclTemp     = 30104 - 30001 // 103
	prachtConnStatus   = 30107 - 30001 // 106 (+1 for second connector)
)

func init() {
	registry.Add("pracht-alpha", NewPrachtAlphaFromConfig)
}

// NewPrachtAlphaFromConfig creates a PrachtAlpha charger from generic config
func NewPrachtAlphaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Connector         uint16
		modbus.Settings   `mapstructure:",squash"`
		Timeout           time.Duration
		Currentworkaround bool
	}{
		Connector: 1,
		Settings: modbus.Settings{
			ID: 1,
		},
		Currentworkaround: true,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPrachtAlpha(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID, cc.Timeout, cc.Connector, cc.Currentworkaround)
}

// NewPrachtAlpha creates PrachtAlpha charger
func NewPrachtAlpha(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, timeout time.Duration, vehicle uint16, currentWorkaround bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	if timeout > 0 {
		conn.Timeout(timeout)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("pracht")
	conn.Logger(log.TRACE)

	wb := &PrachtAlpha{
		conn:              conn,
		vehicle:           vehicle,
		curr:              6,
		currentWorkaround: currentWorkaround,
	}

	return wb, err
}

func (wb *PrachtAlpha) register(reg int) uint16 {
	res := uint16(reg)
	if wb.vehicle > 1 {
		res++
	}
	return res
}

// Status implements the api.Charger interface
func (wb *PrachtAlpha) Status() (api.ChargeStatus, error) {
	reg := wb.register(prachtConnStatus)
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch u := binary.BigEndian.Uint16(b); u {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2, 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *PrachtAlpha) Enabled() (bool, error) {
	reg := wb.register(prachtConnCurrent)

	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return false, err
	}

	result := binary.BigEndian.Uint16(b) > 0

	if wb.currentWorkaround {
		// get total current (https://github.com/evcc-io/evcc/issues/3738)
		t, err := wb.conn.ReadHoldingRegisters(prachtTotalCurrent, 1)
		if err != nil {
			return false, err
		}

		result = result && binary.BigEndian.Uint16(t) > 0
	}

	return result, nil
}

// Enable implements the api.Charger interface
func (wb *PrachtAlpha) Enable(enable bool) error {
	var curr uint16 = 1
	if enable {
		curr = wb.curr
	}

	return wb.setCurrent(curr)
}

func (wb *PrachtAlpha) setTotalCurrent(current uint16) error {
	var err error = nil

	if wb.currentWorkaround {
		// set total current (https://github.com/evcc-io/evcc/issues/3738)
		_, err = wb.conn.WriteSingleRegister(prachtTotalCurrent, current)
	}

	return err
}

func (wb *PrachtAlpha) setConnectorCurrent(current uint16) error {
	reg := wb.register(prachtConnCurrent)
	_, err := wb.conn.WriteSingleRegister(reg, current)

	return err
}

func (wb *PrachtAlpha) setCurrent(current uint16) error {
	var err error

	// order of calls is important, since it is not possible to
	// increase connector current > total current (it will be revoked)
	err = wb.setTotalCurrent(current)
	if err != nil {
		return err
	}

	err = wb.setConnectorCurrent(current)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PrachtAlpha) MaxCurrent(current int64) error {
	err := wb.setCurrent(uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}
	return err
}

var _ api.Diagnosis = (*PrachtAlpha)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *PrachtAlpha) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(prachtTotalCurrent, 1); err == nil {
		fmt.Printf("\tTotal Current Limit:\t\t%dA\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wb.register(prachtConnCurrent), 1); err == nil {
		fmt.Printf("\tConn %d Current Limit:\t\t%dA\n", wb.vehicle, binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(prachtRfidCount, 1); err == nil {
		fmt.Printf("\tRFID cards learned:\t\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(prachtFirmwareRfid, 1); err == nil {
		fmt.Printf("\tVersion RFID board:\t\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(prachtFirmwareMain, 1); err == nil {
		fmt.Printf("\tVersion Main board:\t\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(prachtEnclTemp, 1); err == nil {
		fmt.Printf("\tEnclosure Temperature:\t\t%.2f°C\n", float64(binary.BigEndian.Uint16(b)-72)*0.4244)
	}
	if b, err := wb.conn.ReadHoldingRegisters(wb.register(prachtConnStatus), 1); err == nil {
		fmt.Printf("\tConnection Status (vehicle %d):\t%d\n", wb.vehicle, binary.BigEndian.Uint16(b))
	}
}
