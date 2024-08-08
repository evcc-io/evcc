package charger

// LICENSE

// Copyright (c) 2024 premultiply

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
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// smartEVSE is an api.Charger implementation
type smartEVSE struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	smartEVSERegSerial             = 0x0000 // 5 regs
	smartEVSERegFirmware           = 0x0005
	smartEVSERegExternalLock       = 0x0010
	smartEVSERegI2Cerrors          = 0x0011
	smartEVSERegLockLock           = 0x0015
	smartEVSERegUnlockLock         = 0x0016
	smartEVSERegDisconnectCP       = 0x0017
	smartEVSERegMaxCurrentAdv      = 0x0102
	smartEVSERegChargingState      = 0x0103
	smartEVSERegTemp               = 0x0104 // 1 °C uint16
	smartEVSERegCurrents           = 0x0105 // 1/256 A * 3 uint16
	smartEVSERegSessionEnergy      = 0x0108 // 1/256 kWh uint16
	smartEVSERegVoltages           = 0x0109 // 1/256 V * 3 uint16
	smartEVSERegEnergy             = 0x010d // 1/256 kWh uint32s
	smartEVSERegMaxCurrent         = 0x0201 // Lbyte max current 1 A, Hbyte 1s max current 1 A
	smartEVSERegSettings           = 0x0204 // bits: 7: x, 6: x, 5: x, 4: CP_AUTO_DISCONNECT, 3: MISUSE_LOCKPORT_AS_CP_DISCONNECT, 2: DCL_MUST_BE_PRESENT, 1: LOCK_STATE, 0: PHASES
	smartEVSERegCPDisconnectTime   = 0x0208 // CP interruption time 1 ms uint16
	smartEVSERegTimeoutBeforeCPDis = 0x0209 // time the board waits before it disconnects CP 1 ms uint16

	smartEVSEConfAutoCPDisconnect             = 0x10
	smartEVSEConfMisuseLockPortAsCPDisconnect = 0x8
	smartEVSEConfDCLMustBePresent             = 0x4
	smartEVSEConfLockState                    = 0x2
	smartEVSEConfPhases                       = 0x1
)

func init() {
	registry.Add("smartevse", NewsmartEVSEFromConfig)
}

// NewsmartEVSEFromConfig creates a new smartEVSE ModbusTCP charger
func NewsmartEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewsmartEVSE(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewsmartEVSE creates a new charger
func NewsmartEVSE(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*smartEVSE, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("smartevse")
	conn.Logger(log.TRACE)

	wb := &smartEVSE{
		log:  log,
		conn: conn,
		curr: 6,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *smartEVSE) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(smartEVSERegChargingState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	status := binary.BigEndian.Uint16(b) & 0xff

	switch status {
	case 0:
		return api.StatusNone, nil
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 4:
		return api.StatusD, nil
	case 5:
		return api.StatusE, nil
	case 6:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status returned: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *smartEVSE) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(smartEVSERegMaxCurrentAdv, 1)
	if err != nil {
		return false, err
	}

	return (binary.BigEndian.Uint16(b) >> 8) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *smartEVSE) Enable(enable bool) error {
	var u uint16
	if enable {
		u = ((wb.curr << 8) | wb.curr)
	}

	_, err := wb.conn.WriteSingleRegister(smartEVSERegMaxCurrent, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *smartEVSE) MaxCurrent(current int64) error {
	curr := uint16(current)

	_, err := wb.conn.WriteSingleRegister(smartEVSERegMaxCurrent, ((curr << 8) | curr))
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*smartEVSE)(nil)

// CurrentPower implements the api.Meter interface
func (wb *smartEVSE) CurrentPower() (float64, error) {
	v1, v2, v3, err := wb.Voltages()
	if err != nil {
		return 0, err
	}
	i1, i2, i3, err := wb.Currents()
	if err != nil {
		return 0, err
	}

	return v1*i1 + v2*i2 + v3*i3, nil
}

var _ api.MeterEnergy = (*smartEVSE)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *smartEVSE) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(smartEVSERegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64Swapped(b) / 256, err
}

var _ api.ChargeRater = (*smartEVSE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *smartEVSE) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(smartEVSERegSessionEnergy, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 256, err
}

// getPhaseValues returns 3 sequential register values
func (wb *smartEVSE) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*smartEVSE)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *smartEVSE) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(smartEVSERegCurrents, 256)
}

var _ api.PhaseVoltages = (*smartEVSE)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *smartEVSE) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(smartEVSERegVoltages, 256)
}

var _ api.PhaseSwitcher = (*smartEVSE)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *smartEVSE) Phases1p3p(phases int) error {
	b, err := wb.conn.ReadHoldingRegisters(smartEVSERegSettings, 1)
	if err != nil {
		return err
	}

	settings := binary.BigEndian.Uint16(b) &^ smartEVSEConfPhases // clear bit 0

	if phases == 3 {
		settings |= 1 // set bit 0 (smartEVSEConfPhases)
	}

	if _, err := wb.conn.WriteSingleRegister(smartEVSERegSettings, settings); err != nil {
		return err
	}

	// we need to stop charging quickly for the setting to take effect
	return wb.Enable(false)
}

var _ api.Diagnosis = (*smartEVSE)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *smartEVSE) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(smartEVSERegSerial, 5); err == nil {
		fmt.Printf("\tSerial: %s\n", strings.TrimLeft(strconv.Itoa(int(binary.BigEndian.Uint32(b))), "0"))
	}
	if b, err := wb.conn.ReadInputRegisters(smartEVSERegFirmware, 1); err == nil {
		fmt.Printf("\tFirmware: %d.%d.%d\n", b[0]>>4, b[0]&0x0f, b[1])
	}

	if b, err := wb.conn.ReadInputRegisters(smartEVSERegTemp, 1); err == nil {
		fmt.Printf("\tBoard Temp: %d°C\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadInputRegisters(smartEVSERegTemp, 1); err == nil {
		opt := binary.BigEndian.Uint16(b)
		fmt.Printf("\tOptions: 32/16A: %d DCL: %d\n", opt&0x2, opt&0x1)
	}

	if b, err := wb.conn.ReadHoldingRegisters(smartEVSERegSettings, 1); err == nil {
		settings := binary.BigEndian.Uint16(b)
		phasenum := 1
		if settings&smartEVSEConfPhases == 1 {
			phasenum = 3
		}
		fmt.Printf("\tSettings:\n\t\tPhases: %d\n\t\tLockState: %t\n\t\tDCLMustbePresent: %t\n\t\tLockPortDrivingCPRelais: %t\n\t\tCPInterruptAuto: %t\n", phasenum, settings&0x2 != 0, settings&0x4 != 0, settings&0x8 != 0, settings&0x10 != 0)
	}

	if b, err := wb.conn.ReadHoldingRegisters(smartEVSERegTimeoutBeforeCPDis, 1); err == nil {
		timeoutms := binary.BigEndian.Uint16(b)
		fmt.Printf("\tTimeout before starting CP interruption: %dms\n", timeoutms)
	}

	if b, err := wb.conn.ReadHoldingRegisters(smartEVSERegCPDisconnectTime, 1); err == nil {
		cptime := binary.BigEndian.Uint16(b)
		fmt.Printf("\tCP interruption time: %dms\n", cptime)
	}
}
