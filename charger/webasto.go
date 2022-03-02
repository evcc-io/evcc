package charger

// LICENSE

// Copyright (c) 2022 premultiply

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

// The modbus server must be enabled
// The setting 'Modbus Slave Register Address Set' must be set to 'TQ-DM100'
// The setting 'Modbus TCP Server Port Number' must be set to 502

import (
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// WebastoLive charger implementation
type WebastoLive struct {
	conn    *modbus.Connection
	current uint16
}

const (
	// all holding type registers
	wblRegChargePointState = 1000 // State of the charging device
	wblRegChargeState      = 1001 // Charging
	wblRegEVSEState        = 1002 // State of the charging station
	wblRegCableState       = 1004 // State of the charging cable
	wblRegEVSEError        = 1006 // Error code of the charging station
	wblRegCurrentL1        = 1008 // Charging current L1
	wblRegCurrentL2        = 1010 // Charging current L2
	wblRegCurrentL3        = 1012 // Charging current L3
	wblRegActivePower      = 1020 // Electric Power that can be changed to f.e. mechanical, chemical, thermic power
	//wblRegActivePowerL1        = 1024 // Active power L1
	//wblRegActivePowerL2        = 1028 // Active power L2
	//wblRegActivePowerL3        = 1032 // Active power L3
	wblRegEnergyMeter = 1036 // Meter reading of the charging station
	//wblRegMaxCurrent           = 1100 // Maximal charging current UINT of the hardware (EVSE, cable, EV)
	//wblRegMinimumCurrentLimit  = 1102 // Minimal charging current of the hardware (EVSE, cable, EV)
	//wblRegMaxCurrentFromEVSE   = 1104 // Maximal charging current of the charging station
	//wblRegMaxCurrentFromCable  = 1106 // Maximal charging current of the cable
	//wblRegMaxCurrentFromEV     = 1108 // Maximal charging current of the EV
	//wblRegUserPriority         = 1200 // Priorities of the user 0: not defined 1: high priority - 10: low priority
	//wblRegEVBatteryState       = 1300 // Returns an estimate of the SoC
	//wblRegEVBatteryCapacity    = 1302 // Returns an estimate of the EV Battery Capacity
	//wblRegScheduleType         = 1400 // Type/information of traveling 0: energy that has to be charged, 1: Specification of the desired battery charge (Needs: state of the battery)
	//wblRegRequiredEnergy       = 1402 // Desired energy
	//wblRegRequiredBatteryState = 1404 // Desired state of the battery
	//wblRegScheduledTime        = 1408 // Departure time
	//wblRegScheduledDate        = 1412 // Departure date
	//wblRegChargedEnergy        = 1502 // Sum of charged energy for the current session
	//wblRegStartTime            = 1504 // Start time of charging process
	//wblRegChargingTime         = 1508 // Duration since beginning of charge
	//wblRegEndTime              = 1512 // End time of charging process
	//wblRegUserId               = 1600 // 24 Bytes long User ID (OCPP IdTag) from the current session
	wblRegSmartVehicleDetected = 1620 // ISO15118 Smart Vehicle detected Returns 1 if an EV currently connected is a smart vehicle, or 0 if no EV connected or it is not a smart vehicle,
	wblRegSafeCurrent          = 2000 // Max. charge current under communication failure
	wblRegComTimeout           = 2002 // Communication timeout
	wblRegChargePower          = 5000 // Charge power
	wblRegChargeCurrent        = 5004 // Charge current
	wblRegLifeBit              = 6000 // Communication monitoring 0/1 Toggle-Bit EM writes 1, Live deletes it and puts it on 0.
)

var wblRegCurrents = []uint16{wblRegCurrentL1, wblRegCurrentL2, wblRegCurrentL3}

func init() {
	registry.Add("webasto", NewWebastoLiveFromConfig)
}

// NewWebastoLiveFromConfig creates a WebastoLive charger from generic config
func NewWebastoLiveFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWebastoLive(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewWebastoLive creates WebastoLive charger
func NewWebastoLive(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("webast")
	conn.Logger(log.TRACE)

	wb := &WebastoLive{
		conn:    conn,
		current: 6, // assume min current
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *WebastoLive) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(wblRegChargePointState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	sb := binary.BigEndian.Uint16(b)

	switch sb {
	case 0, 8:
		return api.StatusA, nil
	case 1, 2, 4, 5, 6, 9:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 7:
		return api.StatusE, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", sb)
	}
}

// Enabled implements the api.Charger interface
func (wb *WebastoLive) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wblRegChargeCurrent, 1)
	if err != nil {
		return false, err
	}

	cur := binary.BigEndian.Uint16(b)

	enabled := cur != 0
	if enabled {
		wb.current = cur
	}

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *WebastoLive) Enable(enable bool) error {
	var cur uint16
	if enable {
		cur = wb.current
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(wblRegChargeCurrent, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *WebastoLive) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	cur := uint16(current)

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(wblRegChargeCurrent, 1, b)
	if err == nil {
		wb.current = cur
	}

	return err
}

var _ api.Meter = (*WebastoLive)(nil)

// CurrentPower implements the api.Meter interface
func (wb *WebastoLive) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wblRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.MeterEnergy = (*WebastoLive)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WebastoLive) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wblRegEnergyMeter, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) * 100, nil
}

var _ api.MeterCurrent = (*WebastoLive)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *WebastoLive) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range wblRegCurrents {
		b, err := wb.conn.ReadHoldingRegisters(regCurrent, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(binary.BigEndian.Uint16(b))/1e3)
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Diagnosis = (*WebastoLive)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *WebastoLive) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(wblRegChargePointState, 1); err == nil {
		fmt.Printf("ChargePointState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegChargeState, 1); err == nil {
		fmt.Printf("ChargeState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegEVSEState, 1); err == nil {
		fmt.Printf("EVSEState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegCableState, 1); err == nil {
		fmt.Printf("CableState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegEVSEError, 1); err == nil {
		fmt.Printf("EVSEError:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegSmartVehicleDetected, 2); err == nil {
		fmt.Printf("SmartVehicleDetected:\t%d\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegSafeCurrent, 1); err == nil {
		fmt.Printf("SafeCurrent:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegComTimeout, 1); err == nil {
		fmt.Printf("ComTimeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegChargePower, 1); err == nil {
		fmt.Printf("ChargePower:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(wblRegChargeCurrent, 1); err == nil {
		fmt.Printf("ChargeCurrent:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
