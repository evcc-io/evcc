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
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// FoxESS EV Charger, Modbus TCP Protocol 1.6
// https://github.com/evcc-io/evcc/discussions/26218

// FoxESSEVC charger implementation
type FoxESSEVC struct {
	implement.Caps
	conn *modbus.Connection
}

const (
	// read-only registers (0x03)
	foxRegStatus        = 0x1003 // EVC status
	foxRegVoltages      = 0x1008 // A/B/C phase voltage, 3 registers, 0.1V
	foxRegCurrents      = 0x100B // A/B/C phase current, 3 registers, 0.1A
	foxRegPower         = 0x100E // active power, 0.1kW
	foxRegPhaseSequence = 0x1010 // current phase sequence
	foxRegCurrentEnergy = 0x1016 // session energy, uint32, 0.1kWh
	foxRegTotalEnergy   = 0x1018 // total energy, uint32, 0.1kWh
	foxRegRFID          = 0x101C // last RFID card, uint32

	// read/write registers (write with 0x10)
	foxRegWorkMode     = 0x3000 // work mode
	foxRegMaxCurrent   = 0x3001 // max charging current, 0.1A
	foxRegTimeValidity = 0x3005 // command validity window, seconds

	// write-only registers (write with 0x06)
	foxRegChargingControl = 0x4001 // start/stop charging
	foxRegPhaseSwitching  = 0x4002 // phase sequence switching (requires PBOX)

	foxWorkModeControlled = 0 // external command required
	foxChargingStart      = 1
	foxChargingStop       = 2
	foxTimeValidity       = 60 // maximum command validity window in seconds
)

func init() {
	registry.AddCtx("modbus-foxess", NewFoxESSEVCFromConfig)
}

// NewFoxESSEVCFromConfig creates a FoxESS EV charger from generic config
func NewFoxESSEVCFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Pbox               bool
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewFoxESSEVC(ctx, cc.URI, cc.ID, cc.Pbox)
}

// NewFoxESSEVC creates a FoxESS EV charger
func NewFoxESSEVC(ctx context.Context, uri string, slaveID uint8, pbox bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("modbus-foxess")
	conn.Logger(log.TRACE)

	wb := &FoxESSEVC{
		Caps: implement.New(),
		conn: conn,
	}

	// take control of the charger and keep the command window at its maximum
	if err := wb.writeReg(foxRegWorkMode, foxWorkModeControlled); err != nil {
		return nil, fmt.Errorf("work mode: %w", err)
	}
	if err := wb.writeReg(foxRegTimeValidity, foxTimeValidity); err != nil {
		return nil, fmt.Errorf("time validity: %w", err)
	}

	// phase switching is only available with an external phase-cutting box
	if pbox {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))
	}

	return wb, nil
}

// writeReg writes a single read/write register (0x10)
func (wb *FoxESSEVC) writeReg(reg, val uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)

	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

// readUint32 reads two consecutive registers as a big-endian uint32
func (wb *FoxESSEVC) readUint32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

// getPhaseValues returns 3 sequential register values scaled by divider
func (wb *FoxESSEVC) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *FoxESSEVC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 0: // idle
		return api.StatusA, nil
	case 1, 4, 5: // connect, pause, finish
		return api.StatusB, nil
	case 2, 3: // start, charging
		return api.StatusC, nil
	default: // fault, locked, reserved
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *FoxESSEVC) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegStatus, 1)
	if err != nil {
		return false, err
	}

	// enabled when charger is starting (2) or charging (3)
	s := binary.BigEndian.Uint16(b)

	return s == 2 || s == 3, nil
}

// Enable implements the api.Charger interface
func (wb *FoxESSEVC) Enable(enable bool) error {
	val := uint16(foxChargingStop)
	if enable {
		val = foxChargingStart
	}

	_, err := wb.conn.WriteSingleRegister(foxRegChargingControl, val)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *FoxESSEVC) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*FoxESSEVC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *FoxESSEVC) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	return wb.writeReg(foxRegMaxCurrent, uint16(10*current))
}

var _ api.Meter = (*FoxESSEVC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *FoxESSEVC) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegPower, 1)
	if err != nil {
		return 0, err
	}

	// 0.1kW -> W
	return float64(binary.BigEndian.Uint16(b)) * 100, nil
}

var _ api.ChargeRater = (*FoxESSEVC)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *FoxESSEVC) ChargedEnergy() (float64, error) {
	energy, err := wb.readUint32(foxRegCurrentEnergy)
	if err != nil {
		return 0, err
	}

	// 0.1kWh -> kWh
	return float64(energy) / 10, nil
}

var _ api.MeterEnergy = (*FoxESSEVC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *FoxESSEVC) TotalEnergy() (float64, error) {
	energy, err := wb.readUint32(foxRegTotalEnergy)
	if err != nil {
		return 0, err
	}

	// 0.1kWh -> kWh
	return float64(energy) / 10, nil
}

var _ api.PhaseCurrents = (*FoxESSEVC)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *FoxESSEVC) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(foxRegCurrents, 10)
}

var _ api.PhaseVoltages = (*FoxESSEVC)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *FoxESSEVC) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(foxRegVoltages, 10)
}

var _ api.Identifier = (*FoxESSEVC)(nil)

// Identify implements the api.Identifier interface
func (wb *FoxESSEVC) Identify() (string, error) {
	id, err := wb.readUint32(foxRegRFID)
	if err != nil {
		return "", err
	}

	if id == 0 {
		return "", nil
	}

	return fmt.Sprintf("%08X", id), nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *FoxESSEVC) phases1p3p(phases int) error {
	// 0: three-phase, 1: single-phase (L2)
	val := uint16(0)
	if phases == 1 {
		val = 1
	}

	_, err := wb.conn.WriteSingleRegister(foxRegPhaseSwitching, val)

	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *FoxESSEVC) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(foxRegPhaseSequence, 1)
	if err != nil {
		return 0, err
	}

	// 0: three-phase output, 1: L2, 2: L3
	if binary.BigEndian.Uint16(b) == 0 {
		return 3, nil
	}

	return 1, nil
}
