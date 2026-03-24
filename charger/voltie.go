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
	"github.com/evcc-io/evcc/util/sponsor"
)

// Voltie charger implementation
// https://voltie.eu
// Modbus API documentation v1.02

const (
	voltieRegChargerID       = 0x0000 // R, INT16, Voltie Charger ID
	voltieRegFirmware        = 0x0001 // R, INT16, FW version
	voltieRegStatus          = 0x000A // R, INT16, EVSE_STATE
	voltieRegAutoStart       = 0x000B // R/W, INT16, Auto Start enabled
	voltieRegChargingEnabled = 0x000C // R/W, INT16, Charging enabled
	voltieRegCharging        = 0x000D // R, INT16, Charging (0=no charging, 1=charging)
	voltieRegPhases          = 0x000E // R, INT16, Number of phases in use
	voltieRegStopReason      = 0x0012 // R, INT16, Charge stop reason
	voltieRegCurrentLimit    = 0x0014 // R/W, INT16, Software current limit [mA]

	voltieRegVoltages       = 0x2000 // R, INT32, Phase L1 voltage [mV]
	voltieRegCurrents       = 0x2006 // R, INT32, Phase L1 charging current [mA]
	voltieRegChargeDuration = 0x200C // R, INT32, Charge duration [s]
	voltieRegChargedEnergy  = 0x200E // R, INT32, Charged energy in current session [Ws]
	voltieRegChargingPower  = 0x2010 // R, INT32, Charging power [W]
)

// Voltie is an api.Charger implementation for Voltie wallboxes
type Voltie struct {
	conn *modbus.Connection
}

func init() {
	registry.AddCtx("voltie", NewVoltieFromConfig)
}

// NewVoltieFromConfig creates a Voltie charger from generic config
func NewVoltieFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVoltie(ctx, cc.URI, cc.ID)
}

// NewVoltie creates a Voltie charger
func NewVoltie(ctx context.Context, uri string, slaveID uint8) (*Voltie, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("voltie")
	conn.Logger(log.TRACE)

	wb := &Voltie{
		conn: conn,
	}

	// Disable auto start
	if _, err := wb.conn.WriteSingleRegister(voltieRegAutoStart, 0); err != nil {
		return nil, err
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Voltie) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	status := binary.BigEndian.Uint16(b)

	// EVSE states:
	// 0x01: vehicle in state A – not connected
	// 0x02: vehicle in state B – connected, ready
	// 0x03: vehicle in state C – charging
	// 0x04: vehicle in state D – charging, ventilation required
	// 0x0D: vehicle in state E – vehicle error
	// 0x05-0x0C, 0x0E-0x11: internal error states
	// 0xFF: charger disabled, not functioning

	switch status {
	case 0x01:
		return api.StatusA, nil
	case 0x02:
		return api.StatusB, nil
	case 0x03, 0x04:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *Voltie) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegChargingEnabled, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Voltie) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(voltieRegChargingEnabled, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Voltie) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Voltie)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Voltie) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	_, err := wb.conn.WriteSingleRegister(voltieRegCurrentLimit, uint16(current*1000))
	return err
}

var _ api.Meter = (*Voltie)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Voltie) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegChargingPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.ChargeRater = (*Voltie)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Voltie) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 3.6e6, nil // Ws to kWh
}

var _ api.PhaseCurrents = (*Voltie)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Voltie) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegCurrents, 6)

	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 1e3 // mA to A
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*Voltie)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Voltie) Voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(voltieRegVoltages, 6)

	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 1e3 // mV to V
	}

	return res[0], res[1], res[2], nil
}

var _ api.Diagnosis = (*Voltie)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Voltie) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegChargerID, 1); err == nil {
		fmt.Printf("\tCharger ID:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegFirmware, 1); err == nil {
		fmt.Printf("\tFirmware:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegStatus, 1); err == nil {
		fmt.Printf("\tStatus:\t\t0x%04X\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegPhases, 1); err == nil {
		fmt.Printf("\tPhases:\t\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(voltieRegStopReason, 1); err == nil {
		fmt.Printf("\tStop reason:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
