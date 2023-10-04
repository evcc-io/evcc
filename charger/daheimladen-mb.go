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

// DaheimLadenMB charger implementation
type DaheimLadenMB struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	dlRegChargingState   = 0   // Uint16 RO ENUM
	dlRegConnectorState  = 2   // Uint16 RO ENUM
	dlRegCurrents        = 5   // 3xUint32 RO 0.1A
	dlRegActivePower     = 12  // Uint32 RO 1W
	dlRegTotalEnergy     = 28  // Uint32 RO 0.1KWh
	dlRegEvseMaxCurrent  = 32  // Uint16 RO 0.1A
	dlRegCableMaxCurrent = 36  // Uint16 RO 0.1A
	dlRegStationId       = 38  // Chr[16] RO UTF16
	dlRegCardId          = 54  // Chr[16] RO UTF16
	dlRegChargedEnergy   = 72  // Uint16 RO 0.1kWh
	dlRegChargingTime    = 78  // Uint32 RO 1s
	dlRegSafeCurrent     = 87  // Uint16 WR 0.1A
	dlRegCommTimeout     = 89  // Uint16 WR 1s
	dlRegCurrentLimit    = 91  // Uint16 WR 0.1A
	dlRegChargeControl   = 93  // Uint16 WR ENUM
	dlRegChargeCmd       = 95  // Uint16 WR ENUM
	dlRegVoltages        = 108 // 3xUint32 RO 0.1V
)

func init() {
	registry.Add("daheimladen-mb", NewDaheimLadenMBFromConfig)
}

// NewDaheimLadenMBFromConfig creates a DaheimLadenMB charger from generic config
func NewDaheimLadenMBFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLadenMB(cc.URI, cc.ID)
}

// NewDaheimLadenMB creates DaheimLadenMB charger
func NewDaheimLadenMB(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("daheimladen-mb")
	conn.Logger(log.TRACE)

	wb := &DaheimLadenMB{
		log:  log,
		conn: conn,
		curr: 60, // assume min current
	}

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr > 0 {
		wb.curr = curr
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(dlRegCommTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Second / 2)
	}

	return wb, err
}

func (wb *DaheimLadenMB) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.ReadHoldingRegisters(dlRegSafeCurrent, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *DaheimLadenMB) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(dlRegCurrentLimit, 1, b)

	return err
}

func (wb *DaheimLadenMB) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
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

// Status implements the api.Charger interface
func (wb *DaheimLadenMB) Status() (api.ChargeStatus, error) {
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
	case 4: // Charging (C)
		enabled, err := wb.Enabled()
		if !enabled {
			return api.StatusB, err
		}
		return api.StatusC, nil
	case 5: // Start-UP Fail (B2)
		return api.StatusB, nil
	case 6: // Session Terminated by EVSE
		return api.StatusB, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *DaheimLadenMB) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr > 0, err
}

// Enable implements the api.Charger interface
func (wb *DaheimLadenMB) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *DaheimLadenMB) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	err := wb.setCurrent(wb.curr)
	if err == nil {
		wb.curr = uint16(current * 10)
	}

	return err
}

var _ api.Meter = (*DaheimLadenMB)(nil)

// CurrentPower implements the api.Meter interface
func (wb *DaheimLadenMB) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*DaheimLadenMB)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *DaheimLadenMB) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

// getPhaseValues returns 3 sequential register values
func (wb *DaheimLadenMB) getPhaseValues(reg uint16) (float64, float64, float64, error) {
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

var _ api.PhaseCurrents = (*DaheimLadenMB)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *DaheimLadenMB) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(dlRegCurrents)
}

var _ api.PhaseVoltages = (*DaheimLadenMB)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *DaheimLadenMB) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(dlRegVoltages)
}

var _ api.Identifier = (*DaheimLadenMB)(nil)

// Identify implements the api.Identifier interface
func (wb *DaheimLadenMB) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegCardId, 16)
	if err != nil {
		return "", err
	}
	return utf16BytesToString(b, binary.BigEndian), nil
}

var _ api.Diagnosis = (*DaheimLadenMB)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *DaheimLadenMB) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargingState, 1); err == nil {
		fmt.Printf("\tCharging Station State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegEvseMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCableMaxCurrent, 1); err == nil {
		fmt.Printf("\tCable Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegStationId, 16); err == nil {
		fmt.Printf("\tStation ID:\t%s\n", utf16BytesToString(b, binary.BigEndian))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCardId, 16); err == nil {
		fmt.Printf("\tCard ID:\t%s\n", utf16BytesToString(b, binary.BigEndian))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCurrentLimit, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargeControl, 1); err == nil {
		fmt.Printf("\tCharge Control:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargeCmd, 1); err == nil {
		fmt.Printf("\tCharge Command:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
