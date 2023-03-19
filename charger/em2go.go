package charger

// LICENSE

// Copyright (c) 2019-2023 andig

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
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://files2.elv.com/public/25/2522/252210/Internet/252210_modbus_tcp_register.pdf

// Em2Go charger implementation
type Em2Go struct {
	conn *modbus.Connection
}

const (
	em2GoRegStatus         = 0
	em2GoRegCurrents       = 6
	em2GoRegPower          = 12
	em2GoRegEnergy         = 28
	em2GoRegCurrent        = 32
	em2GoRegSerial         = 38
	em2GoRegChargedEnergy  = 72
	em2GoRegChargeDuration = 78
	em2GoRegChargeMode     = 93
	em2GoRegChargeCommand  = 95
)

func init() {
	registry.Add("em2go", NewEm2GoFromConfig)
}

// NewEm2GoFromConfig creates a Mennekes Em2Go charger from generic config
func NewEm2GoFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 0xff,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEm2Go(cc.URI, cc.ID)
}

// NewEm2Go creates Em2Go charger
func NewEm2Go(uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("em2go")
	conn.Logger(log.TRACE)

	wb := &Em2Go{
		conn: conn,
	}

	// set charge on command
	// b := make([]byte, 2)
	// _, err = wb.conn.WriteMultipleRegisters(em2GoRegChargeMode, 1,b)

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Em2Go) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 1:
		return api.StatusA, nil
	case 2, 3:
		return api.StatusB, nil
	case 4:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[1])
	}
}

// Enabled implements the api.Charger interface
func (wb *Em2Go) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargeCommand, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Em2Go) Enable(enable bool) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, map[bool]uint16{true: 1, false: 2}[enable])

	_, err := wb.conn.WriteMultipleRegisters(em2GoRegChargeCommand, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Em2Go) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Em2Go)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Em2Go) MaxCurrentMillis(current float64) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(10*current))

	_, err := wb.conn.WriteMultipleRegisters(em2GoRegCurrent, 1, b)
	return err
}

var _ api.Meter = (*Em2Go)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Em2Go) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b), nil
}

var _ api.MeterEnergy = (*Em2Go)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Em2Go) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b) / 10, nil
}

var _ api.PhaseCurrents = (*Em2Go)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Em2Go) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegCurrents, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10,
		float64(binary.BigEndian.Uint16(b[2:])) / 10,
		float64(binary.BigEndian.Uint16(b[4:])) / 10, nil
}

var _ api.ChargeRater = (*Em2Go)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Em2Go) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b) / 10, nil
}

var _ api.ChargeTimer = (*Em2Go)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Em2Go) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargeDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.Diagnosis = (*Em2Go)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Em2Go) Diagnose() {
	var serial []byte
	for reg := 0; reg < 8; reg++ {
		b, err := wb.conn.ReadHoldingRegisters(em2GoRegSerial+2*uint16(reg), 2)
		if err != nil {
			return
		}
		serial = append(serial, b...)
	}

	fmt.Printf("Serial: %s\n", string(serial))
}
