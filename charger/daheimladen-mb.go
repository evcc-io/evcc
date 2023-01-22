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
	"unicode/utf16"
	"unicode/utf8"

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
	dhlChargingState   = 0  // Uint16 RO ENUM
	dhlConnectorState  = 2  // Uint16 RO ENUM
	dhlActivePower     = 12 // Uint32 RO 1W
	dhlTotalEnergy     = 28 // Uint32 RO 0.1KWh
	dhlEvseMaxCurrent  = 32 // Uint16 RO 0.1A
	dhlCableMaxCurrent = 36 // Uint16 RO 0.1A
	dhlStationId       = 38 // Chr[16] RO UTF16
	dhlCardId          = 54 // Chr[16] RO UTF16
	dhlChargedEnergy   = 72 // Uint16 RO 0.1kWh
	dhlChargingTime    = 78 // Uint32 RO 1s
	dhlSafeCurrent     = 87 // Uint16 WR 0.1A
	dhlCommTimeout     = 89 // Uint16 WR 1s
	dhlCurrentLimit    = 91 // Uint16 WR 0.1A
	dhlChargeControl   = 93 // Uint16 WR ENUM
	dhlChargeCmd       = 95 // Uint16 WR ENUM
)

var dhlCurrents = []uint16{6, 8, 10}      // Uint16 RO 0.1A
var dhlVoltages = []uint16{109, 111, 113} // Uint16 RO 0.1V

func init() {
	registry.Add("dhl", NewDaheimLadenModbusFromConfig)
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

	log := util.NewLogger("dhl")
	conn.Logger(log.TRACE)

	wb := &DaheimLadenModbus{
		conn: conn,
		curr: 60, // assume min current
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *DaheimLadenModbus) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlChargingState, 1)
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
	case 4: // Charging (C)
		enabled, err := wb.Enabled()
		if !enabled {
			return api.StatusB, err
		}
		return api.StatusC, nil
	case 6: // Session Terminated by EVSE
		return api.StatusB, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *DaheimLadenModbus) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlCurrentLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *DaheimLadenModbus) Enable(enable bool) error {
	var current uint16
	var cmd uint16 = 2 // stop ession

	if enable {
		current = wb.curr
		cmd = 1 // start session
	}
	_ = wb.setCurrent(current)

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cmd)

	_, err := wb.conn.WriteMultipleRegisters(dhlChargeCmd, 1, b)
	return err
}

// setCurrent writes the current limit in coarse 1A steps
func (wb *DaheimLadenModbus) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(dhlCurrentLimit, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *DaheimLadenModbus) MaxCurrent(current int64) error {
	//return wb.MaxCurrentMillis(float64(current))
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	return wb.setCurrent(uint16(current * 10))
}

/*
var _ api.ChargerEx = (*DaheimLadenModbus)(nil)

// maxCurrentMillis implements the api.ChargerEx interface

	func (wb *DaheimLadenModbus) MaxCurrentMillis(current float64) error {
		if current < 6 {
			return fmt.Errorf("invalid current %.1g", current)
		}

		return wb.setCurrent(uint16(current * 10))
	}
*/
var _ api.Meter = (*DaheimLadenModbus)(nil)

// CurrentPower implements the api.Meter interface
func (wb *DaheimLadenModbus) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

/*
var _ api.ChargeTimer = (*DaheimLadenModbus)(nil)

// ChargingTime implements the api.ChargeTimer interface

	func (wb *DaheimLadenModbus) ChargingTime() (time.Duration, error) {
		b, err := wb.conn.ReadHoldingRegisters(dhlChargingTime, 2)
		if err != nil {
			return 0, err
		}

		return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
	}

var _ api.ChargeRater = (*DaheimLadenModbus)(nil)

// ChargedEnergy implements the api.MeterEnergy interface

	func (wb *DaheimLadenModbus) ChargedEnergy() (float64, error) {
		b, err := wb.conn.ReadHoldingRegisters(dhlChargedEnergy, 1)
		if err != nil {
			return 0, err
		}

		return float64(binary.BigEndian.Uint16(b)) / 10, err
	}
*/
var _ api.MeterEnergy = (*DaheimLadenModbus)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *DaheimLadenModbus) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

var _ api.PhaseCurrents = (*DaheimLadenModbus)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *DaheimLadenModbus) Currents() (float64, float64, float64, error) {
	var i []float64
	for _, regCurrent := range dhlCurrents {
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
	for _, regVoltage := range dhlVoltages {
		b, err := wb.conn.ReadHoldingRegisters(regVoltage, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		u = append(u, float64(binary.BigEndian.Uint16(b))/10)
	}

	return u[0], u[1], u[2], nil
}

// UTF16BytesToString converts UTF-16 encoded bytes, in big or little endian byte order,
// to a UTF-8 encoded string.
func UTF16BytesToString(b []byte, o binary.ByteOrder) string {
	utf := make([]uint16, (len(b)+(2-1))/2)
	for i := 0; i+(2-1) < len(b); i += 2 {
		utf[i/2] = o.Uint16(b[i:])
	}
	if len(b)/2 < len(utf) {
		utf[len(utf)-1] = utf8.RuneError
	}
	return string(utf16.Decode(utf))
}

/*
var _ api.Identifier = (*DaheimLadenModbus)(nil)

// identify implements the api.Identifier interface

	func (wb *DaheimLadenModbus) Identify() (string, error) {
		b, err := wb.conn.ReadHoldingRegisters(dhlCardId, 16)
		if err != nil {
			return "", err
		}

		return UTF16BytesToString(b, binary.BigEndian), nil
	}
*/
var _ api.Diagnosis = (*DaheimLadenModbus)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *DaheimLadenModbus) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(dhlChargingState, 1); err == nil {
		fmt.Printf("\tCharging Station State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlEvseMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlCableMaxCurrent, 1); err == nil {
		fmt.Printf("\tCable Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlStationId, 16); err == nil {
		fmt.Printf("\tStation ID:\t%s\n", UTF16BytesToString(b, binary.BigEndian))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlCardId, 16); err == nil {
		fmt.Printf("\tCard ID:\t%s\n", UTF16BytesToString(b, binary.BigEndian))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlCurrentLimit, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlChargeControl, 1); err == nil {
		fmt.Printf("\tCharge Control:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlChargeCmd, 1); err == nil {
		fmt.Printf("\tCharge Command:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
