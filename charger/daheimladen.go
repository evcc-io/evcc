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
	"unicode/utf16"
	"unicode/utf8"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// DaheimLaden charger implementation
type DaheimLaden struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	dhlRegChargingState   = 0   // Uint16 RO ENUM
	dhlRegConnectorState  = 2   // Uint16 RO ENUM
	dhlRegCurrents        = 5   // 3xUint32 RO 0.1A
	dhlRegActivePower     = 12  // Uint32 RO 1W
	dhlRegTotalEnergy     = 28  // Uint32 RO 0.1KWh
	dhlRegEvseMaxCurrent  = 32  // Uint16 RO 0.1A
	dhlRegCableMaxCurrent = 36  // Uint16 RO 0.1A
	dhlRegStationId       = 38  // Chr[16] RO UTF16
	dhlRegCardId          = 54  // Chr[16] RO UTF16
	dhlRegChargedEnergy   = 72  // Uint16 RO 0.1kWh
	dhlRegChargingTime    = 78  // Uint32 RO 1s
	dhlRegSafeCurrent     = 87  // Uint16 WR 0.1A
	dhlRegCommTimeout     = 89  // Uint16 WR 1s
	dhlRegCurrentLimit    = 91  // Uint16 WR 0.1A
	dhlRegChargeControl   = 93  // Uint16 WR ENUM
	dhlRegChargeCmd       = 95  // Uint16 WR ENUM
	dhlRegVoltages        = 108 // 3xUint32 RO 0.1V

	dhlCmdStartSession = 1
	dhlCmdStopSession  = 2
)

func init() {
	registry.Add("daheimladen", NewDaheimLadenFromConfig)
}

// NewDaheimLadenFromConfig creates a DaheimLaden charger from generic config
func NewDaheimLadenFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLaden(cc.URI, cc.ID)
}

// NewDaheimLaden creates DaheimLaden charger
func NewDaheimLaden(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("dhl")
	conn.Logger(log.TRACE)

	wb := &DaheimLaden{
		log:  log,
		conn: conn,
		curr: 60, // assume min current
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(dhlRegCommTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	go wb.heartbeat(time.Duration(binary.BigEndian.Uint16(b)/2) * time.Second)

	return wb, err
}

func (wb *DaheimLaden) heartbeat(timeout time.Duration) {
	for range time.NewTicker(timeout).C {
		if _, err := wb.conn.ReadHoldingRegisters(dhlRegSafeCurrent, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *DaheimLaden) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlRegChargingState, 1)
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
func (wb *DaheimLaden) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlRegCurrentLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *DaheimLaden) Enable(enable bool) error {
	var current uint16
	var cmd uint16 = dhlCmdStopSession // stop session

	if enable {
		current = wb.curr
		cmd = dhlCmdStartSession // start (new) session
	}
	_ = wb.setCurrent(current)

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cmd)

	_, err := wb.conn.WriteMultipleRegisters(dhlRegChargeCmd, 1, b)
	return err
}

// setCurrent writes the current limit in coarse 1A steps
func (wb *DaheimLaden) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(dhlRegCurrentLimit, 1, b)
	if err == nil {
		wb.curr = current
	}
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *DaheimLaden) MaxCurrent(current int64) error {
	//return wb.MaxCurrentMillis(float64(current))
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	return wb.setCurrent(uint16(current * 10))
}

/*
var _ api.ChargerEx = (*DaheimLaden)(nil)

// maxCurrentMillis implements the api.ChargerEx interface

	func (wb *DaheimLaden) MaxCurrentMillis(current float64) error {
		if current < 6 {
			return fmt.Errorf("invalid current %.1g", current)
		}

		return wb.setCurrent(uint16(current * 10))
	}
*/

var _ api.Meter = (*DaheimLaden)(nil)

// CurrentPower implements the api.Meter interface
func (wb *DaheimLaden) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

/*
var _ api.ChargeTimer = (*DaheimLaden)(nil)

// ChargingTime implements the api.ChargeTimer interface

	func (wb *DaheimLaden) ChargingTime() (time.Duration, error) {
		b, err := wb.conn.ReadHoldingRegisters(dhlChargingTime, 2)
		if err != nil {
			return 0, err
		}

		return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
	}
*/

/*
var _ api.ChargeRater = (*DaheimLaden)(nil)

// ChargedEnergy implements the api.MeterEnergy interface

	func (wb *DaheimLaden) ChargedEnergy() (float64, error) {
		b, err := wb.conn.ReadHoldingRegisters(dhlChargedEnergy, 1)
		if err != nil {
			return 0, err
		}

		return float64(binary.BigEndian.Uint16(b)) / 10, err
	}
*/

var _ api.MeterEnergy = (*DaheimLaden)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *DaheimLaden) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dhlRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

// getPhaseValues returns 3 sequential register values
func (wb *DaheimLaden) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 10
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*DaheimLaden)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *DaheimLaden) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(dhlRegCurrents)
}

var _ api.PhaseVoltages = (*DaheimLaden)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *DaheimLaden) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(dhlRegVoltages)
}

// utf16BytesToString converts UTF-16 encoded bytes, in big or little endian byte order,
// to a UTF-8 encoded string.
func utf16BytesToString(b []byte, o binary.ByteOrder) string {
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
var _ api.Identifier = (*DaheimLaden)(nil)

// identify implements the api.Identifier interface

	func (wb *DaheimLaden) Identify() (string, error) {
		b, err := wb.conn.ReadHoldingRegisters(dhlCardId, 16)
		if err != nil {
			return "", err
		}

		return UTF16BytesToString(b, binary.BigEndian), nil
	}
*/

var _ api.Diagnosis = (*DaheimLaden)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *DaheimLaden) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegChargingState, 1); err == nil {
		fmt.Printf("\tCharging Station State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegEvseMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegCableMaxCurrent, 1); err == nil {
		fmt.Printf("\tCable Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegStationId, 16); err == nil {
		fmt.Printf("\tStation ID:\t%s\n", utf16BytesToString(b, binary.BigEndian))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegCardId, 16); err == nil {
		fmt.Printf("\tCard ID:\t%s\n", utf16BytesToString(b, binary.BigEndian))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegCurrentLimit, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegChargeControl, 1); err == nil {
		fmt.Printf("\tCharge Control:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dhlRegChargeCmd, 1); err == nil {
		fmt.Printf("\tCharge Command:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
