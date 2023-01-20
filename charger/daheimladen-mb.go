package charger

// LICENSE

// Copyright (c) 2023 premultiply

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
)

// DaheimLadenModbus charger implementation
type DaheimLadenModbus struct {
	conn *modbus.Connection
	curr uint16
}

const (
	dlRegChargingState  = 0  // Uint16 RO
	dlRegConnectorState = 2  // Uint16 RO
	dlRegActivePower    = 12 // Uint32 RO 1W
	dlRegTotalEnergy    = 28 // Uint32 RO 0.1KWh
	dlRegCardId         = 54 // Uint16 RO MAX 32
	dlRegChargedEnergy  = 72 // Uint16 RO 0.1kWh
	dlRegChargingTime   = 78 // Uint32 RO 1s
	dlRegCurrentLimit   = 91 // Uint16 WR 1A
)

var dlRegCurrents = []uint16{6, 8, 10}      // Uint16 RO 0.1A
var dlRegVoltages = []uint16{109, 111, 113} // Uint16 RO 0.1V

func init() {
	registry.Add("DaheimLadenModbus", NewDaheimLadenModbusFromConfig)
}

// NewDaheimLadenModbusFromConfig creates a DaheimLadenModbus charger from generic config
func NewDaheimLadenModbusFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLadenModbus(cc.URI, cc.ID)
}

// NewDaheimLadenModbus creates DaheimLadenModbus charger
func NewDaheimLadenModbus(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("DaheimLadenModbus")
	conn.Logger(log.TRACE)

	wb := &DaheimLadenModbus{
		conn: conn,
		curr: 6, // assume min current
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *DaheimLadenModbus) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegChargingState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	switch s {
	case 1: // Standby (A)
		return api.StatusA, nil
	case 2: // Connect (B1)
		return api.StatusB, nil
	case 3: // Start-up State (B2)
		return api.StatusB, nil
	case 4: // Charging (C2)
		return api.StatusC, nil
	case 6: // Charging End (C1)
		return api.StatusC, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *DaheimLadenModbus) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegCurrentLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *DaheimLadenModbus) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// setCurrent writes the current limit in coarse 1A steps
func (wb *DaheimLadenModbus) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(dlRegCurrentLimit, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *DaheimLadenModbus) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	wb.curr = uint16(current)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*DaheimLadenModbus)(nil)

// CurrentPower implements the api.Meter interface
func (wb *DaheimLadenModbus) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.ChargeTimer = (*DaheimLadenModbus)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *DaheimLadenModbus) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegChargingTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.ChargeRater = (*DaheimLadenModbus)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *DaheimLadenModbus) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegChargedEnergy, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, err
}

var _ api.PhaseCurrents = (*DaheimLadenModbus)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *DaheimLadenModbus) Currents() (float64, float64, float64, error) {
	var i []float64
	for _, regCurrent := range dlRegCurrents {
		b, err := wb.conn.ReadHoldingRegisters(regCurrent, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		i = append(i, float64(binary.BigEndian.Uint16(b))/10)
	}

	return i[0], i[1], i[2], nil
}

var _ api.PhaseVoltages = (*DaheimLadenModbus)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *DaheimLadenModbus) Voltages() (float64, float64, float64, error) {
	var u []float64
	for _, regVoltage := range dlRegVoltages {
		b, err := wb.conn.ReadHoldingRegisters(regVoltage, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		u = append(u, float64(binary.BigEndian.Uint16(b))/10)
	}

	return u[0], u[1], u[2], nil
}

var _ api.Diagnosis = (*DaheimLadenModbus)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *DaheimLadenModbus) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargingState, 1); err == nil {
		fmt.Printf("\tCharging Station State:\t%d\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", b)
	}
}
