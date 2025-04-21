package charger

// LICENSE

// Copyright (c) 2022 premultiply, andig

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
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"

	"github.com/lorenzodonini/ocpp-go/ocpp1.6/core"
)

// eSolutions eProWallbox charger implementation
type EProWallbox struct {
	conn    *modbus.Connection
	current uint32
	log     *util.Logger
}

const (
	eproRegSerial           = 40000 // Serial Number 8 registers (16 ASCII chars)
	eproReg1ph3p            = 40017 // 1p/3p mode, 1 register, UINT16 (RO)
	eproRegGeneralStatus    = 40101 // IEC 61851 Status, 1 register, UINT16
	eproRegOCPPStatus       = 40102 // OCPP Status, 1 register, UINT16
	eproRegUserCurrentLimit = 40103 // Current limit set by user (via app) in mA, 2 registers, UINT32
	eproRegHwCurrentLimit   = 40018 // Current limit by HW (rotary switch) in mA, 2 registers, UINT32
	eproRegOnOffInput       = 40406 // On/Off state, 1 registers, UINT16
	eproRegCurrentLimit     = 40407 // in mA
	// TODO: Fallback + Watchdog
	eproRegL1Vac = 40604 // L1 voltage in V, 2 registers, Float32 (followed by L2, L3)
	eproRegL1Iac = 40620 // L1 current in A, 2 registers, Float32 (followed by L2, L3)
)

func init() {
	registry.AddCtx("eprowallbox", NewEProWallboxFromConfig)
}

// NewEProWallboxFromConfig creates a eProWallbox charger from generic config
func NewEProWallboxFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEProWallbox(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
}

// NewEProWallbox creates eProWallbox charger
func NewEProWallbox(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("eprowallbox")
	conn.Logger(log.TRACE)

	wb := &EProWallbox{
		conn:    conn,
		current: 6000, // assume min current
		log:     log,
	}

	// keep-alive
	// go func() {
	// 	for range time.Tick(30 * time.Second) {
	// 		_, _ = wb.status()
	// 	}
	// }()

	return wb, err
}

func (wb *EProWallbox) getGeneralStatus() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(eproRegGeneralStatus, 1)
	if err != nil {
		return "F", err
	}
	s := binary.BigEndian.Uint16(b)
	statusDecodeMap := map[uint16]string{
		0: "A1",
		1: "A2",
		2: "B1",
		3: "B2",
		4: "C1",
		5: "C2",
		6: "D1",
		7: "D2",
		8: "E",
		9: "F",
	}
	if status, ok := statusDecodeMap[s]; ok {
		return status, nil
	} else {

		return "F", fmt.Errorf("invalid status value: %d", s)
	}
}

func (wb *EProWallbox) getOCPPStatus() (core.ChargePointStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(eproRegOCPPStatus, 1)

	if err != nil {
		return core.ChargePointStatusFaulted, err
	}

	s := binary.BigEndian.Uint16(b)
	statusDecodeMap := map[uint16]core.ChargePointStatus{
		0: core.ChargePointStatusAvailable,
		1: core.ChargePointStatusPreparing,
		2: core.ChargePointStatusCharging,
		3: core.ChargePointStatusSuspendedEVSE,
		4: core.ChargePointStatusSuspendedEV,
		5: core.ChargePointStatusFinishing,
		6: core.ChargePointStatusReserved,
		7: core.ChargePointStatusUnavailable,
		8: core.ChargePointStatusFaulted,
	}

	if status, ok := statusDecodeMap[s]; ok {
		wb.log.TRACE.Printf("OCPP Status: %s", status)
		return status, nil
	} else {
		wb.log.TRACE.Printf("OCPP Status: Unknown (%x)", s)
		return core.ChargePointStatusFaulted, fmt.Errorf("invalid status value: %d", s)
	}
}

// Status implements the api.Charger interface
func (wb *EProWallbox) Status() (api.ChargeStatus, error) {
	ocppStatus, err := wb.getOCPPStatus()
	if err != nil {
		return api.StatusNone, err
	}

	// Same decoding as for OCPP charger implementation
	switch ocppStatus {
	case
		core.ChargePointStatusAvailable,   // "Available"
		core.ChargePointStatusUnavailable: // "Unavailable"
		return api.StatusA, nil
	case
		core.ChargePointStatusPreparing,     // "Preparing"
		core.ChargePointStatusSuspendedEVSE, // "SuspendedEVSE"
		core.ChargePointStatusSuspendedEV,   // "SuspendedEV"
		core.ChargePointStatusFinishing:     // "Finishing"
		return api.StatusB, nil
	case
		core.ChargePointStatusCharging: // "Charging"
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", ocppStatus)
	}
}

// Enabled implements the api.Charger interface
func (wb *EProWallbox) Enabled() (bool, error) {
	ocppStatus, err := wb.getOCPPStatus()

	// try with OCPP status implementation
	if err == nil {
		switch ocppStatus {
		case
			core.ChargePointStatusSuspendedEVSE:
			return false, nil
		case
			core.ChargePointStatusCharging,
			core.ChargePointStatusSuspendedEV:
			return true, nil
		}
	}

	return true, nil
}

// Enable implements the api.Charger interface
func (wb *EProWallbox) Enable(enable bool) error {
	var current uint32
	wb.log.TRACE.Printf("Called Set Enable: %t", enable)

	if enable {
		current = wb.current
	}

	return wb.setCurrent(current)
}

// setCurrent writes the current limit in mA
func (wb *EProWallbox) setCurrent(current uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, current)

	// current is written independent of limit
	// disabling charger results in 0A
	_, err := wb.conn.WriteMultipleRegisters(eproRegCurrentLimit, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *EProWallbox) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EProWallbox)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *EProWallbox) MaxCurrentMillis(current float64) error {

	wb.log.TRACE.Printf("Called Set MaxCurrentMillis: %f", current)
	wb.current = uint32(current * 1e3)

	if (wb.current < 6000) || (wb.current > 32000) {
		return fmt.Errorf("invalid current %d", wb.current)
	}
	return wb.setCurrent(wb.current)
}

// var _ api.Meter = (*ABB)(nil)

// // CurrentPower implements the api.Meter interface
// func (wb *EProWallbox) CurrentPower() (float64, error) {
// 	return 30, nil

// 	// b, err := wb.conn.ReadHoldingRegisters(abbRegPower, 2)
// 	// if err != nil {
// 	// 	return 0, err
// 	// }

// 	// return float64(binary.BigEndian.Uint32(b)), err
// }

// var _ api.ChargeRater = (*ABB)(nil)

// // ChargedEnergy implements the api.MeterEnergy interface
// func (wb *EProWallbox) ChargedEnergy() (float64, error) {

// 	return 30, nil

// 	// 	b, err := wb.conn.ReadHoldingRegisters(abbRegEnergy, 2)
// 	// 	if err != nil {
// 	// 		return 0, err
// 	// 	}

// 	// return float64(binary.BigEndian.Uint32(b)) / 1e3, err
// }

// getPhaseValues returns 3 sequential register values
func (wb *EProWallbox) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {

	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		bits := binary.BigEndian.Uint32(b) // falls Big-Endian – sonst LittleEndian
		res[i] = float64(math.Float32frombits(bits)) / divider
	}
	wb.log.TRACE.Printf("getPhaseValues: %d %f %f %f", reg, res[0], res[1], res[2])

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*EProWallbox)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *EProWallbox) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(eproRegL1Iac, 1)
}

var _ api.PhaseVoltages = (*EProWallbox)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *EProWallbox) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(eproRegL1Vac, 1)
}

var _ api.Diagnosis = (*EProWallbox)(nil)

func (wb *EProWallbox) Diagnose() {

	// use description from modbus communication map pdf from Free2Move
	var decoderOcppStatus = map[uint16]string{
		0: "Available (A)", 1: "Preparing (B)", 2: "Charging (C)",
		3: "SuspendedEV (D)", 4: "SuspendedEVSE (E)", 5: "Finishing (F)",
		6: "Reserved (G)", 7: "Unavailable (H)", 8: "Faulted (I)",
	}

	var decoderDeviceType = map[uint16]string{
		0: "AC 1-phase",
		1: "AC 3-phase",
	}

	var decoderGeneralStatus = map[uint16]string{
		0: "A1", 1: "A2", 2: "B1", 3: "B2", 4: "C1", 5: "C2", 6: "D1", 7: "D2", 8: "E", 9: "F",
	}

	var decoderEnabledDisabled = map[uint16]string{0: "Disabled", 1: "Enabled"}

	var decoderTempDerating = map[uint16]string{
		0: "Disabled",
		1: "Enabled (limits when temperature > 85°C)",
	}

	var decoderMeterUsed = map[uint16]string{0: "External", 1: "Internal"}

	var decoderBaudRate = map[uint16]string{
		0: "9600", 1: "19200", 2: "38400", 3: "57600", 4: "115200",
	}

	decode := func(m map[uint16]string, value uint16) string {
		if decoded, ok := m[value]; ok {
			return decoded
		}
		return fmt.Sprintf("Unknown (%d)", value)
	}

	readASCII := func(address, size uint16, label string) {
		b, _ := wb.conn.ReadHoldingRegisters(address, size)
		fmt.Printf("%-25s %s\n", label+":", string(b))
	}

	readUint16Dec := func(address uint16, label string, decoder map[uint16]string) {
		b, _ := wb.conn.ReadHoldingRegisters(address, 1)
		val := binary.BigEndian.Uint16(b)
		if decoder != nil {
			fmt.Printf("%-25s %s (%d)\n", label+":", decode(decoder, val), val)
		} else {
			fmt.Printf("%-25s %d\n", label+":", val)
		}
	}

	readUint16 := func(address uint16, label string, unit string) {
		b, _ := wb.conn.ReadHoldingRegisters(address, 1)
		val := binary.BigEndian.Uint16(b)
		if unit != "" {
			fmt.Printf("%-25s %d %s\n", label+":", val, unit)
		} else {
			fmt.Printf("%-25s %d\n", label+":", val)
		}
	}

	readUint32 := func(address uint16, label string, unit string) {
		b, _ := wb.conn.ReadHoldingRegisters(address, 2)
		val := binary.BigEndian.Uint32(b)
		if unit != "" {
			fmt.Printf("%-25s %d %s\n", label+":", val, unit)
		} else {
			fmt.Printf("%-25s %d\n", label+":", val)
		}
	}

	fmt.Println("eProWallbox diagnostics:")

	// Identification
	readASCII(40000, 8, "Serial Number")
	readASCII(40008, 4, "Software Version")
	readASCII(40012, 4, "Modbus Version")

	// Device Info
	readUint16(40016, "HW Type", "")
	readUint16Dec(40017, "Device Type", decoderDeviceType)
	readUint32(40018, "HW Current Limit", "mA")

	// Status
	readUint16(40100, "Actual Error", "")
	readUint16Dec(40101, "General Status", decoderGeneralStatus)
	readUint16Dec(40102, "OCPP Status", decoderOcppStatus)
	readUint32(40103, "User Current Limit", "mA")
	readUint16Dec(40106, "Load Unbalance Limiting", decoderEnabledDisabled)
	readUint16Dec(40107, "DPM Limiting", decoderEnabledDisabled)
	readUint16Dec(40108, "Temp Derating Status", decoderTempDerating)

	// Communication
	readUint16(40200, "RTU Address", "")
	readUint16(40201, "New RTU Address", "")
	readUint16Dec(40210, "Baud Rate", decoderBaudRate)
	readUint16Dec(40211, "New Baud Rate", decoderBaudRate)

	// Config
	readUint16Dec(40400, "DPM Enabled", decoderEnabledDisabled)
	readUint16Dec(40401, "MID Meter Enabled", decoderEnabledDisabled)
	readUint16Dec(40402, "Load Unbalance Enabled", decoderEnabledDisabled)
	readUint32(40403, "Load Unbalance Current", "mA")
	readUint16(40406, "On/Off Input", "")
	readUint32(40407, "Current Limit", "mA")
	readUint32(40409, "Fallback Current", "mA")
	readUint16Dec(40411, "Meter Used", decoderMeterUsed)
	readUint16(40412, "Watchdog Enabled", "")
	readUint16(40413, "Watchdog Timer", "s")
}
