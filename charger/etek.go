package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// https://www.etek-electric.com/epc-ev-charge-controller/ekepc2-cs-ev-charging-station-controller
// https://www.etek-electric.com/ev-charging-station-knowledge/how-to-setting-rs485-communication-for-ekepc2-c-s-epc-controller

// ETEK EKEPC2 charger implementation
type Etek struct {
	*embed
	log  *util.Logger
	conn *modbus.Connection
}

const (
	etekRegInvalidMeterAddr = 0xffff // Indicates no external meter configured

	// Meter configuration registers
	etekRegMeterVoltageAAddr = 90 // 1# Meter A-phase voltage address
	etekRegMeterVoltageBAddr = 91 // 1# Meter B-phase voltage address
	etekRegMeterVoltageCAddr = 92 // 1# Meter C-phase voltage address
	etekRegMeterCurrentAddr  = 93 // 1# Total current address
	etekRegMeterPowerAddr    = 94 // 1# Total power address
	etekRegMeterEnergyAddr   = 95 // 1# Total energy address

	// Read registers
	etekRegStatus    = 141 // Current working status (0-19)
	etekRegCPVoltage = 153 // CP positive voltage
	etekRegPWMDuty   = 152 // Current output PWM duty cycle

	// Metering registers
	etekRegVoltageA = 159 // 1# meter A phase voltage
	etekRegVoltageB = 160 // 1# meter B phase voltage
	etekRegVoltageC = 161 // 1# meter C phase voltage
	etekRegCurrent  = 162 // 1# meter current
	etekRegPower    = 163 // 1# total power of the meter (W)
	etekRegEnergy   = 164 // 1# total energy (2 registers, 32-bit, Wh)

	// Write registers
	etekRegRemoteControl = 89  // Remote start/stop (0=invalid, 1=start, 2=stop)
	etekRegMaxCurrent    = 109 // Max Output Current PWM Duty cycle (*100)
)

func init() {
	registry.AddCtx("etek", NewEtekFromConfig)
}

//go:generate go tool decorate -f decorateEtek -b *Etek -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)"

// NewEtekFromConfig creates an ETEK EKEPC2 charger from generic config
func NewEtekFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewEtek(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
	if err != nil {
		return nil, err
	}

	// Check if external meter is configured
	// If register value is 65535 (0xffff), no external meter is configured
	var (
		currentPower func() (float64, error)
		totalEnergy  func() (float64, error)
		currents     func() (float64, float64, float64, error)
		voltages     func() (float64, float64, float64, error)
	)

	// Check power register (94)
	if b, err := wb.conn.ReadHoldingRegisters(etekRegMeterPowerAddr, 1); err == nil {
		if binary.BigEndian.Uint16(b) != etekRegInvalidMeterAddr {
			currentPower = wb.currentPower
		}
	}

	// Check energy register (95)
	if b, err := wb.conn.ReadHoldingRegisters(etekRegMeterEnergyAddr, 1); err == nil {
		if binary.BigEndian.Uint16(b) != etekRegInvalidMeterAddr {
			totalEnergy = wb.totalEnergy
		}
	}

	/* 	// Check current register (93)
	   	if b, err := wb.conn.ReadHoldingRegisters(etekRegMeterCurrentAddr, 1); err == nil {
	   		if binary.BigEndian.Uint16(b) != etekRegInvalidMeterAddr {
	   			currents = wb.currents
	   		}
	   	} */

	// Check voltage registers (90, 91, 92) - if any is configured, enable voltages
	if b, err := wb.conn.ReadHoldingRegisters(etekRegMeterVoltageAAddr, 3); err == nil {
		va := binary.BigEndian.Uint16(b[0:2])
		vb := binary.BigEndian.Uint16(b[2:4])
		vc := binary.BigEndian.Uint16(b[4:6])
		if va != etekRegInvalidMeterAddr || vb != etekRegInvalidMeterAddr || vc != etekRegInvalidMeterAddr {
			voltages = wb.voltages
		}
	}

	return decorateEtek(wb, currentPower, totalEnergy, currents, voltages), nil
}

// NewEtek creates an ETEK EKEPC2 charger
func NewEtek(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*Etek, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("etek")
	conn.Logger(log.TRACE)

	wb := &Etek{
		log:  log,
		conn: conn,
	}

	return wb, nil
}

// getStatus reads the current working status from register 141
func (wb *Etek) getStatus() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(etekRegStatus, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// getCPVoltage reads the CP positive voltage from register 153
func (wb *Etek) getCPVoltage() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(etekRegCPVoltage, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// Status implements the api.Charger interface
func (wb *Etek) Status() (api.ChargeStatus, error) {
	status, err := wb.getStatus()
	if err != nil {
		return api.StatusNone, err
	}

	// Status mapping based on EKEPC2 documentation and user feedback:
	// 0: Initialization
	// 1: Ready (no vehicle connected)
	// 2: Fault
	// 3,4: Connected (vehicle connected, not charging)
	// 5: Charging
	// 19: Emergency stop (when disabled via register 89)
	//
	// When status is 19 (emergency stop), we need to check CP voltage
	// to determine if a vehicle is connected (voltage > 300 means connected)

	switch status {
	case 1:
		// Ready - no vehicle connected
		return api.StatusA, nil
	case 3, 4:
		// Vehicle connected, not charging
		return api.StatusB, nil
	case 5:
		// Charging
		return api.StatusC, nil
	case 19:
		// Emergency stop - check CP voltage to determine actual status
		voltage, err := wb.getCPVoltage()
		if err != nil {
			return api.StatusNone, err
		}
		if voltage < 300 {
			// No vehicle connected
			return api.StatusA, nil
		}
		// Vehicle connected but not charging (disabled)
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown status: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *Etek) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(etekRegRemoteControl, 1)
	if err != nil {
		return false, err
	}

	// Register 89: 0=invalid/disabled, 1=start/enabled, 2=stop/disabled
	return binary.BigEndian.Uint16(b) == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Etek) Enable(enable bool) error {
	var value uint16
	if enable {
		value = 1 // Start charging
	} else {
		value = 2 // Stop charging
	}

	_, err := wb.conn.WriteSingleRegister(etekRegRemoteControl, value)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Etek) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Etek)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Etek) MaxCurrentMillis(current float64) error {
	// The PWM value is calculated as: current (A) * 167
	// According to the documentation and user feedback,
	// the value should be scaled by 167 to get the correct PWM duty cycle

	if current < 6 {
		return fmt.Errorf("current %.1fA is below minimum of 6A", current)
	}

	// Scale current to PWM duty cycle value
	pwmValue := uint16(current * 167)

	_, err := wb.conn.WriteSingleRegister(etekRegMaxCurrent, pwmValue)
	return err
}

// currentPower implements the api.Meter interface
func (wb *Etek) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(etekRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Etek) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(etekRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil // Convert to kWh
}

/* // currents implements the api.PhaseCurrents interface
func (wb *Etek) currents() (float64, float64, float64, error) {
	// Note: EKEPC2 only provides total current, not per-phase
	b, err := wb.conn.ReadHoldingRegisters(etekRegCurrent, 1)
	if err != nil {
		return 0, 0, 0, err
	}

	current := float64(binary.BigEndian.Uint16(b)) / 10.0 // Assuming 0.1A resolution

	// For single-phase charging, return current on L1 only
	// For 3-phase, distribute evenly (this is an approximation)
	// The actual implementation depends on the pole selection in register 127
	return current, 0, 0, nil
} */

// voltages implements the api.PhaseVoltages interface
func (wb *Etek) voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(etekRegVoltageA, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	vA := float64(binary.BigEndian.Uint16(b[0:2]))
	vB := float64(binary.BigEndian.Uint16(b[2:4]))
	vC := float64(binary.BigEndian.Uint16(b[4:6]))

	return vA, vB, vC, nil
}
