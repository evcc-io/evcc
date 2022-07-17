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

// PrachtAlpha charger implementation
type PrachtAlpha struct {
	conn    *modbus.Connection
	vehicle uint16
	curr    uint16
}

const (
	prachtTotalCurrent = 40003 - 40001 // total limit of all connectors
	prachtMaxCurrent   = 40004 - 40001 // +1 for second connector
	prachtStatus       = 30107 - 30001 // +1 for second connector
)

func init() {
	registry.Add("pracht-alpha", NewPrachtAlphaFromConfig)
}

// https://www.prachtenergy.com/wp-content/uploads/2021/03/instruction-manual-charging-station-pracht-alphaXT-eng-1.pdf

// NewPrachtAlphaFromConfig creates a PrachtAlpha charger from generic config
func NewPrachtAlphaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Connector       uint16
		modbus.Settings `mapstructure:",squash"`
		Timeout         time.Duration
	}{
		Connector: 1,
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPrachtAlpha(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID, cc.Timeout, cc.Connector)
}

// NewPrachtAlpha creates PrachtAlpha charger
func NewPrachtAlpha(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, timeout time.Duration, vehicle uint16) (api.Charger, error) {
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
		conn:    conn,
		vehicle: vehicle,
		curr:    6,
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
	reg := wb.register(prachtStatus)
	b, err := wb.conn.ReadInputRegisters(reg, 1)
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
		return api.StatusF, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *PrachtAlpha) Enabled() (bool, error) {
	reg := wb.register(prachtMaxCurrent)

	b, err := wb.conn.ReadInputRegisters(reg, 1)
	if err != nil {
		return false, err
	}

	// get total current (https://github.com/evcc-io/evcc/issues/3738)
	tot, err := wb.conn.ReadInputRegisters(prachtTotalCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0 && binary.BigEndian.Uint16(tot) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *PrachtAlpha) Enable(enable bool) error {
	var curr uint16
	if enable {
		curr = wb.curr
	}

	return wb.setCurrent(curr)
}

func (wb *PrachtAlpha) setCurrent(current uint16) error {
	reg := wb.register(prachtMaxCurrent)
	_, err := wb.conn.WriteSingleRegister(reg, current)

	// set total current (https://github.com/evcc-io/evcc/issues/3738)
	if err == nil {
		_, err = wb.conn.WriteSingleRegister(prachtTotalCurrent, current)
	}

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
