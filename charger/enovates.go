package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

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
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://github.com/enovates/enovates-modbus
// EMS over Modbus TCP (ModBusTCPEMSEnabled) must be enabled on the charger.

// Enovates charger implementation
type Enovates struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16 // offered EMS limit in mA
}

const (
	enovatesRegMaxAmp         = 51   // uint16 RO, max amp per phase
	enovatesRegCurrents       = 200  // 3x uint16 RO, mA
	enovatesRegVoltages       = 203  // 3x int16 RO, V
	enovatesRegPower          = 206  // int16 RO, W (active power total)
	enovatesRegEnergy         = 216  // int32 RO, Wh (active energy import total)
	enovatesRegMode3StateStr  = 301  // 2x register RO, IEC 61851 state, e.g. "C2"
	enovatesRegEMSLimit       = 400  // int16 RW, mA (-1 = no limit, 0 = disabled)
	enovatesRegToken          = 401  // 16x register RO, ASCII
	enovatesRegCurrentOffered = 417  // uint16 RO, mA
	enovatesRegSerial         = 5032 // 16x register RO, ASCII
	enovatesRegModel          = 5048 // 16x register RO, ASCII
	enovatesRegFirmware       = 5064 // 16x register RO, ASCII
)

func init() {
	registry.AddCtx("enovates", NewEnovatesFromConfig)
}

// NewEnovatesFromConfig creates an Enovates charger from generic config
func NewEnovatesFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEnovates(ctx, cc.URI, cc.ID)
}

// NewEnovates creates an Enovates charger
func NewEnovates(ctx context.Context, uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("enovates")
	conn.Logger(log.TRACE)

	wb := &Enovates{
		log:     log,
		conn:    conn,
		current: 6000,
	}

	// verify connection and EMS register availability
	if _, err := wb.conn.ReadHoldingRegisters(enovatesRegMode3StateStr, 2); err != nil {
		return nil, err
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Enovates) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegMode3StateStr, 2)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string(b))
}

// Enabled implements the api.Charger interface
func (wb *Enovates) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegEMSLimit, 1)
	if err != nil {
		return false, err
	}

	// 0 = charging disabled, anything else (positive limit or -1 = no limit) = enabled
	return int16(binary.BigEndian.Uint16(b)) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Enovates) Enable(enable bool) error {
	var curr uint16
	if enable {
		curr = wb.current
	}

	return wb.setCurrent(curr)
}

// setCurrent writes the EMS limit in mA without modifying the stored current value
func (wb *Enovates) setCurrent(current uint16) error {
	_, err := wb.conn.WriteSingleRegister(enovatesRegEMSLimit, current)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Enovates) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Enovates)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Enovates) MaxCurrentMillis(current float64) error {
	curr := uint16(current * 1e3)

	err := wb.setCurrent(curr)
	if err == nil {
		wb.current = curr
	}

	return err
}

var _ api.CurrentGetter = (*Enovates)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *Enovates) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegEMSLimit, 1)
	if err != nil {
		return 0, err
	}

	// -1 signals no EMS limit, fall back to the hardware maximum
	if limit := int16(binary.BigEndian.Uint16(b)); limit >= 0 {
		return float64(limit) / 1e3, nil
	}

	b, err = wb.conn.ReadHoldingRegisters(enovatesRegMaxAmp, 1)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint16ToFloat64(b), nil
}

var _ api.Meter = (*Enovates)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Enovates) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegPower, 1)
	if err != nil {
		return 0, err
	}

	return rs485.RTUInt16ToFloat64(b), nil
}

var _ api.MeterEnergy = (*Enovates)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Enovates) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUInt32ToFloat64(b) / 1e3, nil
}

var _ api.PhaseCurrents = (*Enovates)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Enovates) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegCurrents, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	return float64(binary.BigEndian.Uint16(b[0:2])) / 1e3,
		float64(binary.BigEndian.Uint16(b[2:4])) / 1e3,
		float64(binary.BigEndian.Uint16(b[4:6])) / 1e3, nil
}

var _ api.PhaseVoltages = (*Enovates)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Enovates) Voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegVoltages, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	return float64(int16(binary.BigEndian.Uint16(b[0:2]))),
		float64(int16(binary.BigEndian.Uint16(b[2:4]))),
		float64(int16(binary.BigEndian.Uint16(b[4:6]))), nil
}

var _ api.Identifier = (*Enovates)(nil)

// Identify implements the api.Identifier interface
func (wb *Enovates) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(enovatesRegToken, 16)
	if err != nil {
		return "", err
	}

	return trimModbusString(b), nil
}

var _ api.Diagnosis = (*Enovates)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Enovates) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(enovatesRegSerial, 16); err == nil {
		fmt.Printf("\tSerial:\t%s\n", trimModbusString(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(enovatesRegModel, 16); err == nil {
		fmt.Printf("\tModel:\t%s\n", trimModbusString(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(enovatesRegFirmware, 16); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", trimModbusString(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(enovatesRegEMSLimit, 1); err == nil {
		fmt.Printf("\tEMS Limit:\t%dmA\n", int16(binary.BigEndian.Uint16(b)))
	}
	if b, err := wb.conn.ReadHoldingRegisters(enovatesRegCurrentOffered, 1); err == nil {
		fmt.Printf("\tCurrent Offered:\t%dmA\n", binary.BigEndian.Uint16(b))
	}
}

// trimModbusString decodes a NUL-padded ASCII modbus string
func trimModbusString(b []byte) string {
	return strings.TrimRight(string(b), "\x00")
}
