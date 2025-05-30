package charger

// LICENSE

// Copyright (c) 2022 premultiply, andig, fritz-net

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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// ABB charger implementation
type ABB struct {
	log         *util.Logger
	conn        *modbus.Connection
	lastCurrent uint32
	lastStatus  uint32
	settings    AbbSettings
}

type AbbSettings struct {
	autoStartStopSession bool `json:",omitempty" yaml:",omitempty"`
}

// https://library.e.abb.com/public/4124e0d39f614ba7b0a7a6f7a2ce1f99/ABB_Terra_AC_Charger_ModbusCommunication_v1.7.pdf
// https://library.e.abb.com/public/982c2befa2734d259e66d76fa4a7ba77/ABB_Terra_AC_Charger_ModbusCommunication_v1.11.pdf?x-sign=iLpQGyucvLxy9vxpJTfP6c8T5PD2xNxDRcfuyYcqp84qm7jSCqomJm0%2FlJ9jA%2BvU
const (
	// Read Only Registers
	abbRegSerial     = 0x4000 // Serial Number 4 unsigned RO available
	abbRegFirmware   = 0x4004 // Firmware version 2 unsigned RO available
	abbRegMaxRated   = 0x4006 // Max rated current 2 unsigned RO available // <- is set in the Terra Config App
	abbRegErrorCode  = 0x4008 // Error Code 2 unsigned RO available
	abbRegSocketLock = 0x400A // Socket Lock State 2 unsigned RO available
	abbRegStatus     = 0x400C // Charging state 2 unsigned RO available
	abbRegGetCurrent = 0x400E // Current charging current limit 2 0.001 A unsigned RO
	abbRegCurrents   = 0x4010 // Charging current phases 6 0.001 A unsigned RO available
	abbRegVoltages   = 0x4016 // Voltage phases 6 0.1 V unsigned RO available
	abbRegPower      = 0x401C // Active power 2 1 W unsigned RO available
	abbRegEnergy     = 0x401E // Energy delivered in charging session 2 1 Wh unsigned RO available
	//abbCommunicationTimeout = 0x4020 // Communication timeout 1 1s unsigned RO
	//abbRegCurrentModbus     = 0x4022 // current liomit set by Modbus 2 0.001A unsigned RO
	//abbRegCurrentFallback   = 0x4024 // Fallback current limit 2 1A unsigned RO

	// Write Only Registers
	abbRegSetCurrent = 0x4100 // Set charging current limit 2 0.001 A unsigned WO available (smalles value 6A according to docs/)
	// abbRegSetPhases               = 0x4102 // Set charging phase 1 unsigned WO Not supported // TODO not found in docs - check source
	// abbRegSetSocketLock           = 0x4103 // Set socket lock state 1 unsigned WO
	abbRegSetSession = 0x4105 // Start/Stop Charging Session 1 unsigned WO available
	// abbRegSetCommunicationTimeout = 0x4106 // Set communication timeout 1 1s unsigned WO -> RO 0x4020
	// abbRegSetCurrentFallback      = 0x4109 // Set (current) fallback limit 1 1A unsigned WO -> RO 0x4024; default is 0xFF; if configured the carger will not go into error mode on modbus connection loss
)

func init() {
	registry.AddCtx("abb", NewABBFromConfig)
}

// NewABBFromConfig creates a ABB charger from generic config
func NewABBFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	abbSettings := AbbSettings{
		autoStartStopSession: true,
	}

	return NewABB(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID, abbSettings)
}

// NewABB creates ABB charger
func NewABB(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, abbSettings AbbSettings) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("abb")
	conn.Logger(log.TRACE)

	wb := &ABB{
		log:         log,
		conn:        conn,
		lastCurrent: 6000, // assume min current
		settings:    abbSettings,
	}

	// keep-alive // TODO: sometimes it seems the abb class gets collected if there is no current needed -> WB stopps charge
	go func() {
		for range time.Tick(30 * time.Second) {
			_, _ = wb.status()
		}
	}()

	return wb, err
}

func (wb *ABB) status() (byte, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegStatus, 2)
	if err != nil {
		return 0, err
	}

	status := b[2] & 0x7f

	wb.lastStatus = uint32(status)
	wb.log.TRACE.Printf("status: %0x", wb.lastStatus)

	return b[2] & 0x7f, nil
}

func (wb *ABB) socketStatus() (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegSocketLock, 2)
	if err != nil {
		return 0, err
	}

	/**
	 * Value | Description                                           | Socket     | Non-socket
	 * 0x0000 | No cable is plugged.                                 | Applicable | Not applicable
	 * 0x0001 | Cable is connected to the charging station unlocked. | Applicable | Applicable
	 * 0x0011 | Cable is connected to the charging station locked.   | Applicable | Not applicable
	 * 0x0101 | Cable is connected to the charging station and the
	 * 		    electric vehicle, unlocked in charging station.      | Applicable | Applicable
	 * 0x0111 | Cable is connected to the charging station and the
	 * 		    electric vehicle, locked in charging station.        | Applicable | Not applicable
	 */

	/*
	 * this table results in:
	 * bit 0 = cabled is plugged
	 * bit 1 = cable is locked in charging station
	 * bit 2 = cable is connected to vehicle
	 */

	switch b := binary.BigEndian.Uint32(b); b {
	case 0x0000: // No cable is plugged
		wb.log.TRACE.Println("socket: No cable is plugged (0x0000)")
		return b, nil
	case 0x0001: // Cable is connected to the charging station unlocked
		wb.log.TRACE.Println("socket: Cable is connected to the charging station unlocked (0x0001)")
		return b, nil
	case 0x0011: // Cable is connected to the charging station locked
		wb.log.TRACE.Println("socket: Cable is connected to the charging station locked (0x0011)")
		return b, nil
	case 0x0101: // Cable is connected to the charging station and the electric vehicle, unlocked in charging station
		wb.log.TRACE.Println("socket: Cable is connected to the charging station and the electric vehicle, unlocked in charging station (0x0101)")
		return b, nil
	case 0x0111: // Cable is connected to the charging station and the electric vehicle, locked in charging station
		wb.log.TRACE.Println("socket: Cable is connected to the charging station and the electric vehicle, locked in charging station (0x0111)")
		return b, nil
	default:
		return b, fmt.Errorf("socket: invalid status %0x", b)
	}
}

// Status implements the api.Charger interface
func (wb *ABB) Status() (api.ChargeStatus, error) {
	s, err := wb.status()
	if err != nil {
		return api.StatusNone, err
	}

	if _, err := wb.socketStatus(); err != nil {
		return api.StatusNone, err
	}

	switch s {
	case 0: // State A: Idle
		wb.log.TRACE.Printf("status: State A: Idle (0x%02x)", s)
		return api.StatusA, nil
	case 1: // State B1: EV Plug in, pending authorization
		wb.log.TRACE.Printf("status: State B1: EV Plug in, pending authorization (0x%02x)", s)
		return api.StatusB, nil
	case 2: // State B2: EV Plug in, EVSE ready for charging(PWM)
		wb.log.TRACE.Printf("status: State B2: EV Plug in, EVSE ready for charging (0x%02x)", s)
		return api.StatusB, nil
	case 3: // State C1: EV Ready for charge, S2 closed(no PWM)
		wb.log.TRACE.Printf("status: State C1: EV Ready for charge, S2 closed (0x%02x)", s)
		return api.StatusB, nil
	case 4: // State C2: Charging Contact closed, energy delivering
		wb.log.TRACE.Printf("status: State C2: Charging Contact closed, energy delivering (0x%02x)", s)
		return api.StatusC, nil
	case 5: // Other: Session stopped
		wb.log.TRACE.Printf("status: Other: Session stopped (0x%02x)", s)
		//b, err := wb.conn.ReadHoldingRegisters(abbRegSocketLock, 2)
		b, err := wb.socketStatus()
		if err != nil {
			return api.StatusNone, err
		}
		if b >= 0x0101 {
			return api.StatusB, nil
		}
		return api.StatusA, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %0x", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *ABB) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegGetCurrent, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *ABB) Enable(enable bool) error { // TODO: the issue is that this is only called on mode switch and not on start of evcc so wb stays disabled if it was not enabled by app or rfid
	var current uint32 = 0 // values lower then 6A pause the session
	if enable {
		current = wb.lastCurrent
	}

	wb.log.WARN.Printf("enable: %t; current: %dmA; autoStartStopSession: %t", enable, current, wb.settings.autoStartStopSession)

	// we should also start the charging session if not already -> abbRegSetSession
	if /*wb.settings.autoStartStopSession != nil &&*/ wb.settings.autoStartStopSession {
		b := 0x01 // this would stop the session

		s, err := wb.status()
		if err != nil {
			return fmt.Errorf("getting status: %w", err)
		}

		if s == 0x05 && // 0x05 = session stopped
			enable {
			wb.log.INFO.Printf("session currently stopped -> starting new session; current: %dmA", current)
			b = 0x00 // start session
		} else if s == 0x04 && // 0x04 = energy delivering // TODO do we also want other states to stop session?
			!enable {
			b = 0x01 // stop session
		} else {
			// unknown state
			wb.log.WARN.Printf("unknown session state: %0x, not changing session; requested enable flag: %t", s, enable)
		}

		wb.log.TRACE.Printf("set session: %d; current: %dmA", b, current)
		if _, err := wb.conn.WriteSingleRegister(abbRegSetSession, uint16(b)); err != nil {
			return fmt.Errorf("setting session: %w", err)
		}
	}

	return wb.setCurrent(current)
}

// setCurrent writes the current limit in mA
func (wb *ABB) setCurrent(current uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, current)

	// if last status is 0x05 trigger enable
	if wb.lastStatus == 0x05 && current > 0 {
		wb.log.WARN.Printf("trying to set current %dmA while last status is 0x05 (session stopped), enabling charger", current)
		wb.Enable(true)
	}

	/*
	 * In addition to that charging session will enter Pause state when the current limit is less than
	 * 6A. After that when current limit is set above 6A, then charging session will be resumed. The
	 * choice of 6A is derived from IEC 61851-1
	 */

	wb.log.TRACE.Printf("set current: %dmA; lastStatus: %0x", current, wb.lastStatus)

	_, err := wb.conn.WriteMultipleRegisters(abbRegSetCurrent, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *ABB) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*ABB)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *ABB) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1fA", current) // its actually not invalid but would pause charging if current is less then 6A
	}

	// read max rated current
	b, err := wb.conn.ReadHoldingRegisters(abbRegMaxRated, 2)
	if err != nil {
		return err
	}
	maxRatedCurrentinA := float64(binary.BigEndian.Uint32(b)) / 1e3 // convert mA to A
	wb.log.TRACE.Printf("max rated current: %.0fA", maxRatedCurrentinA*1e3)
	if current > maxRatedCurrentinA {
		return fmt.Errorf("current %.1fA exceeds max rated current %.0fA", current, maxRatedCurrentinA)
	}

	wb.lastCurrent = uint32(current * 1e3)

	return wb.setCurrent(wb.lastCurrent)
}

var _ api.Meter = (*ABB)(nil)

// CurrentPower implements the api.Meter interface
func (wb *ABB) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.ChargeRater = (*ABB)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *ABB) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(abbRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	wb.log.TRACE.Printf("charged energy: %dWh", binary.BigEndian.Uint32(b))

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

// getPhaseValues returns 3 sequential register values
func (wb *ABB) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*ABB)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *ABB) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(abbRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*ABB)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *ABB) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(abbRegVoltages, 10)
}

// var _ api.PhaseSwitcher = (*ABB)(nil)

// // Phases1p3p implements the api.PhaseSwitcher interface
// func (wb *ABB) Phases1p3p(phases int) error {
// 	var b uint16 = 1
// 	if phases != 1 {
// 		b = 2 // 3p
// 	}

// 	_, err := wb.conn.WriteSingleRegister(abbRegPhases, b)
// 	return err
// }

var _ api.Diagnosis = (*ABB)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ABB) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(abbRegSerial, 4); err == nil {
		fmt.Printf("\tSerial:\t%x\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegFirmware, 2); err == nil {
		fmt.Printf("\tFirmware:\t%d.%d.%d\n", b[0], b[1], b[2])
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegMaxRated, 2); err == nil {
		fmt.Printf("\tMax rated current:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegGetCurrent, 2); err == nil {
		fmt.Printf("\tCharging current limit:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegSocketLock, 2); err == nil {
		fmt.Printf("\tSocket lock state:\t%x\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegStatus, 2); err == nil {
		fmt.Printf("\tStatus:\t%x\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(abbRegErrorCode, 2); err == nil {
		fmt.Printf("\tError code:\t%x\n", binary.BigEndian.Uint32(b))
	}
}
