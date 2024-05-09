package charger

// LICENSE

// Copyright (c) 2023 andig

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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// smartEVSE is an api.Charger implementation
type smartEVSE struct {
	log        *util.Logger
	conn       *modbus.Connection
	curr       uint16
	en         bool
	cphwonlock bool
	cpwakeauto bool
	oldfw      bool
}

const (
	smartEVSERegExternalLock       = 0x0010
	smartEVSERegI2Cerrors          = 0x0011
	smartEVSERegLockLock           = 0x0015
	smartEVSERegUnlockLock         = 0x0016
	smartEVSERegDisconnectCP       = 0x0017
	smartEVSERegChargingState      = 0x0103
	smartEVSERegMaxCurrentAdv      = 0x0102
	smartEVSERegCurrents           = 0x0105 // 3 regs, 1/256A
	smartEVSERegTemp               = 0x0104 // 1 reg
	smartEVSERegSerial             = 0x0000 // 5 regs
	smartEVSERegFirmware           = 0x0005
	smartEVSERegEnergy             = 0x010d // 1/256 KWh
	smartEVSERegVoltages           = 0x0109 // 3 regs 1/256 V
	smartEVSERegSessionEnergy      = 0x010f // 3 regs 1/256 KWh
	smartEVSERegMaxCurrent         = 0x0201 //L byte max current A, Hbyte 1s max curren
	smartEVSERegSettings           = 0x0204 //bits: 7: x, 6: x, 5: x, 4: CP_AUTO_DISCONNECT, 3: MISUSE_LOCKPORT_AS_CP_DISCONNECT, 2: DCL_MUST_BE_PRESENT, 1: LOCK_STATE, 0: PHASES
	smartEVSERegCPDisconnectTime   = 0x0208 // CP interruption time
	smartEVSERegTimeoutBeforeCPDis = 0x0209 // time the board waits before it disconnects CP
)

func init() {
	registry.Add("smartevse", NewsmartEVSEFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratesmartEVSE -b *smartEVSE -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewsmartEVSEFromConfig creates a new smartEVSE ModbusTCP charger
func NewsmartEVSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		CpHwOnLock      bool
		CpWakeAuto      bool
	}{
		Settings: modbus.Settings{
			ID: 1, // default
		},
		CpHwOnLock: false,
		CpWakeAuto: false,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewsmartEVSE(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID, cc.CpHwOnLock, cc.CpWakeAuto)
	if err != nil {
		return nil, err
	}

	// features
	var (
		currentPower, chargedEnergy, totalEnergy func() (float64, error)
		currents                                 func() (float64, float64, float64, error)
		voltages                                 func() (float64, float64, float64, error)
		phases                                   func(int) error
	)

	currentPower = wb.currentPower
	chargedEnergy = wb.chargedEnergy
	totalEnergy = wb.totalEnergy
	currents = wb.currents
	voltages = wb.voltages
	phases = wb.phases1p3p

	if b, err := wb.conn.ReadInputRegisters(smartEVSERegFirmware, 1); err == nil {
		fwversion := binary.BigEndian.Uint16(b)
		wb.log.INFO.Printf("board uses firmware version: 0x%x", fwversion)
		if fwversion <= 0x0290e {
			wb.oldfw = true
			if wb.cpwakeauto {
				wb.log.WARN.Println("your board's FW version does not support a timeout based CP interrupt (CpWakeAuto), thus this setting will be ignored")
			}
		}
	}

	return decoratesmartEVSE(wb, currentPower, chargedEnergy, totalEnergy, currents, voltages, phases), nil
}

// NewsmartEVSE creates a new charger
func NewsmartEVSE(uri string, device string, comset string, baudrate int, slaveID uint8, cphwonlock bool, cpwakeauto bool) (*smartEVSE, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("sevse")
	conn.Logger(log.TRACE)

	wb := &smartEVSE{
		log:  log,
		conn: conn,
	}

	wb.cphwonlock = cphwonlock
	wb.cpwakeauto = cpwakeauto

	return wb, err
}

// Status implements the api.Charger interface
func (wb *smartEVSE) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(smartEVSERegChargingState, 1)
	if err != nil {
		return api.StatusNone, err
	}
	wb.log.DEBUG.Println("read state:", b[1])
	status := binary.BigEndian.Uint16(b)
	errorcode := status & 0xff00 >> 8
	status = status & 0xff

	switch errorcode {
	case 1:
		wb.log.ERROR.Println("Temperature High")
	case 2:
		wb.log.ERROR.Println("Stuck Relay")
	case 4:
		wb.log.ERROR.Println("Ground fault")
	case 8:
		wb.log.ERROR.Println("MaxCP too low")
	case 16:
		wb.log.ERROR.Println("MinCP too high")
	case 32:
		wb.log.ERROR.Println("DCL triggered")
	case 64:
		wb.log.ERROR.Println("I2C init failed")
	case 128:
		b, _ = wb.conn.ReadInputRegisters(smartEVSERegI2Cerrors, 1)
		wb.log.ERROR.Printf("%d I2C comm errors detected", binary.BigEndian.Uint16(b))
	default:
	}

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
		wb.en = false
		return false, err
	}
	current := (binary.BigEndian.Uint16(b) & 0xff00) >> 8
	wb.en = current > 0
	if wb.en {
		wb.curr = current
		wb.log.DEBUG.Printf("Enabled with current: %dA ", current)
	} else {
		wb.log.DEBUG.Println("Disabled with current: 0A ")
	}

	return wb.en, nil
}

// Enable implements the api.Charger interface
func (wb *smartEVSE) Enable(enable bool) error {
	var u uint16
	if enable {
		u = ((wb.curr << 8) | wb.curr)
		wb.log.DEBUG.Printf("Enable by setting current: %dA", wb.curr)
	} else {
		u = 0
		wb.log.DEBUG.Printf("Disable by setting current: %dA", u)
	}

	_, err := wb.conn.WriteSingleRegister(smartEVSERegMaxCurrent, u)
	wb.en = err == nil && enable

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *smartEVSE) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*smartEVSE)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *smartEVSE) MaxCurrentMillis(current float64) error {
	i := uint16(current)
	_, err := wb.conn.WriteSingleRegister(smartEVSERegMaxCurrent, (i<<8 | i))
	wb.log.DEBUG.Printf("Set current to: %dA ", i)
	if err == nil {
		wb.curr = i
	}
	return err
}

// currentPower implements the api.Meter interface
func (wb *smartEVSE) currentPower() (float64, error) {
	var v1, v2, v3, i1, i2, i3 float64

	v1, v2, v3, errv := wb.voltages()
	i1, i2, i3, errc := wb.currents()

	if errv != nil || errc != nil {
		return 0, errv
	}

	res := v1*i1 + v2*i2 + v3*i3

	return res, nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *smartEVSE) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(smartEVSERegEnergy, 2)
	if err != nil {
		return 0, err
	}
	total := float64(uint32(b[2])<<24|
		uint32(b[3])<<16|
		uint32(b[0])<<8|
		uint32(b[1])) / 256.0
	wb.log.DEBUG.Printf("Total charged energy: %.3fKWh", total)
	return total, nil
}

func (wb *smartEVSE) tripple(dbgstr string, regbase uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(regbase, 3)

	if err != nil {
		return 0, 0, 0, err
	}

	x := float64(binary.BigEndian.Uint16(b[0:])) / 256.0
	y := float64(binary.BigEndian.Uint16(b[2:])) / 256.0
	z := float64(binary.BigEndian.Uint16(b[4:])) / 256.0
	wb.log.DEBUG.Printf("%s %3.1f %3.1f %3.1f", dbgstr, x, y, z)
	return x, y, z, err
}

// chargedEnergy implements the api.ChargeRater interface
func (wb *smartEVSE) chargedEnergy() (float64, error) {
	e1, e2, e3, err := wb.tripple("Session Energy[KWh]: ", smartEVSERegSessionEnergy)
	return (e1 + e2 + e3), err
}

// currents implements the api.PhaseCurrents interface
func (wb *smartEVSE) currents() (float64, float64, float64, error) {
	return wb.tripple("Currents [A]: ", smartEVSERegCurrents)
}

func (wb *smartEVSE) voltages() (float64, float64, float64, error) {
	return wb.tripple("Voltages[V]: ", smartEVSERegVoltages)
}

// api.Identifier not implemented

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *smartEVSE) phases1p3p(phases int) error {
	var b uint16

	if phases == 3 {
		b = 1
	} else {
		b = 0
	}
	wb.log.DEBUG.Printf("requested switching to %d phases", phases)
	cfg, err := wb.conn.ReadHoldingRegisters(smartEVSERegSettings, 1)
	if err == nil {
		_, err1 := wb.conn.WriteSingleRegister(smartEVSERegSettings, (binary.BigEndian.Uint16(cfg)&0xfffe)|b)
		if wb.en && err1 == nil {
			wb.conn.WriteSingleRegister(smartEVSERegMaxCurrent, (0<<8 | wb.curr)) //we need to stop charging quickly for the setting to take effect
		}
	}

	return err
}

var _ api.Diagnosis = (*smartEVSE)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *smartEVSE) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(smartEVSERegSerial, 5); err == nil {
		fmt.Printf("\tSerial:\t%s\n", strings.TrimLeft(strconv.Itoa(int(binary.BigEndian.Uint32(b))), "0"))
	}
	if b, err := wb.conn.ReadInputRegisters(smartEVSERegFirmware, 1); err == nil {
		fmt.Printf("\tFirmware:\t%d.%d.%d\n", b[0]>>4, b[0]&0x0f, b[1])
	}

	if b, err := wb.conn.ReadInputRegisters(smartEVSERegTemp, 1); err == nil {
		fmt.Printf("\tBoard Temp:\t%dC\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadInputRegisters(smartEVSERegTemp, 1); err == nil {
		opt := binary.BigEndian.Uint16(b)
		fmt.Printf("\tOptions:\t 32/16A:%d DCL:%d\n", opt&0x2, opt&0x1)
	}

	if b, err := wb.conn.ReadHoldingRegisters(smartEVSERegSettings, 1); err == nil {
		settings := binary.BigEndian.Uint16(b)
		phasenum := 1
		if settings&0x1 == 1 {
			phasenum = 3
		}
		fmt.Printf("\tSettings:\n\t\tPhases:%d\n\t\tLockState:%t\n\t\tDCLMustbePresent:%t\n\t\tLockPortDrivingCPRelais:%t\n\t\tCPInterruptAuto:%t\n", phasenum, settings&0x2 != 0, settings&0x4 != 0, settings&0x8 != 0, settings&0x10 != 0)
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

var _ api.Resurrector = (*smartEVSE)(nil)

// WakeUp implements the Resurrector interface
func (wb *smartEVSE) WakeUp() error {
	if wb.cphwonlock {
		if wb.oldfw { //this includes that cp auto mode does not exist
			wb.log.DEBUG.Printf("requested switching to wakeup by interrupting CP")
			cfg, err := wb.conn.ReadHoldingRegisters(smartEVSERegSettings, 1)
			if err == nil {
				wb.conn.WriteSingleRegister(smartEVSERegSettings, (binary.BigEndian.Uint16(cfg)&0xfffd)|1) //set Lockbit
				wb.conn.ReadInputRegisters(smartEVSERegExternalLock, 1)                                    //active lock setting (or relais in our case) by reading
				time.Sleep(3 * time.Second)
				wb.conn.WriteSingleRegister(smartEVSERegSettings, (binary.BigEndian.Uint16(cfg) & 0xfffd)) //reset Lockbit
				wb.conn.ReadInputRegisters(smartEVSERegExternalLock, 1)                                    //active lock setting (or relais in our case) by reading
			}
			return err
		} else if !wb.cpwakeauto {
			wb.log.DEBUG.Printf("requested switching to wakeup by interrupting CP")
			dcp, err := wb.conn.ReadInputRegisters(smartEVSERegDisconnectCP, 1)
			if (binary.BigEndian.Uint16(dcp) == 0xffff) && err == nil {
				wb.log.WARN.Println("warning CpHWOnLock misconfiguration detected: board reports it has no CP hardware configured")
			}
		} else {
			wb.log.DEBUG.Printf("skipping wakeup by CP interrupt as your HW is configured to do it by itself")
			cfg, err := wb.conn.ReadHoldingRegisters(smartEVSERegSettings, 1)
			if (binary.BigEndian.Uint16(cfg)&0x10) == 0 && err == nil {
				wb.log.WARN.Println("warning CpWakeAuto misconfiguration detected: board hast automatic CP mode disabled")
			}
			if (binary.BigEndian.Uint16(cfg)&0x8) == 0 && err == nil {
				wb.log.WARN.Println("warning CpHWOnLock misconfiguration detected: board reports it has no CP hardware configured")
			}
		}
	} else {
		wb.log.DEBUG.Printf("skipping wakeup by CP interrupt as respective HW has not been configured")
	}
	return nil
}
