package charger

// LICENSE

// Copyright (c) 2024 evcc

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

// Details on the Peblar modbus server obtained from: https://developer.peblar.com/modbus-api

import (
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Peblar charger implementation
type Peblar struct {
	conn    *modbus.Connection
	curr    uint32
	enabled bool
	phases  uint16
}

const (
	// Meter addresses
	peblarRegEnergyTotal   = 30000
	peblarRegSessionEnergy = 30004
	peblarRegPowerPhase1   = 30008
	peblarRegPowerPhase2   = 30010
	peblarRegPowerPhase3   = 30012
	peblarRegPowerTotal    = 30014
	peblarRegVoltages      = 30016
	peblarRegCurrents      = 30022

	// Config addresses
	peblarRegSerialNumber  = 30050
	peblarRegProductNumber = 30062
	peblarRegFwIdentifier  = 30074
	peblarRegPhaseCount    = 30092
	peblarRegIndepRelay    = 30093

	// Control addresses
	peblarRegCurrentLimitSource = 30112
	peblarRegCurrentLimitActual = 30113
	peblarRegModbusCurrentLimit = 40000
	peblarRegForce1Phase        = 40002

	// Diagnostic addresses
	peblarRegCpState = 30110
)

func init() {
	registry.Add("peblar", NewPeblarFromConfig)
}

//go:generate decorate -f decoratePeblar -b *Peblar -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewPeblarFromConfig creates a Peblar charger from generic config
func NewPeblarFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPeblar(cc.URI, cc.ID)
}

// NewPeblar creates Peblar charger
func NewPeblar(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	// Register contains the physically connected phases
	b, err := conn.ReadInputRegisters(peblarRegPhaseCount, 1)
	if err != nil {
		return nil, err
	}

	wb := &Peblar{
		conn:   conn,
		curr:   6000,                       // assume min current
		phases: binary.BigEndian.Uint16(b), // required for retrieving the right amount of voltage/current registers
	}

	b, err = conn.ReadInputRegisters(peblarRegIndepRelay, 1)
	if err != nil {
		return nil, err
	}

	var phasesS func(int) error
	var phasesG func() (int, error)

	if binary.BigEndian.Uint16(b) == 1 {
		phasesS = wb.phases1p3p
		phasesG = wb.getPhases
	}

	return decoratePeblar(wb, phasesS, phasesG), err
}

// Status implements the api.Charger interface
func (wb *Peblar) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(peblarRegCpState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := rune(encoding.Uint16(b)); s {
	case 'A', 'B', 'C':
		return api.ChargeStatus(s), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Peblar) Enabled() (bool, error) {
	return verifyEnabled(wb, wb.enabled)
}

// Enable implements the api.Charger interface
func (wb *Peblar) Enable(enable bool) error {
	var current uint32
	if enable {
		current = wb.curr
	}

	err := wb.setCurrent(current)
	if err == nil {
		wb.enabled = enable
	}

	return err
}

// setCurrent writes the current limit in mA
func (wb *Peblar) setCurrent(current uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, current)

	_, err := wb.conn.WriteMultipleRegisters(peblarRegModbusCurrentLimit, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Peblar) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Peblar)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *Peblar) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	curr := uint32(current * 1e3)

	err := wb.setCurrent(curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*Peblar)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Peblar) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(peblarRegPowerTotal, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.ChargeRater = (*Peblar)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Peblar) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(peblarRegSessionEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int64(b)) / 1e3, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Peblar) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(peblarRegEnergyTotal, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int64(b)) / 1e3, nil
}

// getPhaseValues returns 1..3 sequential register values
func (wb *Peblar) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, wb.phases*2)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < int(wb.phases); i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Peblar)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Peblar) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(peblarRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*Peblar)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Peblar) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(peblarRegVoltages, 1)
}

// phases1p3p implements the api.PhaseSwitcher interface via the decorator
func (wb *Peblar) phases1p3p(phases int) error {
	var b uint16
	if phases == 1 {
		b = 1
	}

	_, err := wb.conn.WriteSingleRegister(peblarRegForce1Phase, b)
	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *Peblar) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(peblarRegForce1Phase, 1)
	if err != nil {
		return 0, err
	}

	phases := 3
	if binary.BigEndian.Uint16(b) == 1 {
		phases = 1
	}

	return phases, nil
}

var _ api.Diagnosis = (*Peblar)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Peblar) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(peblarRegSerialNumber, 12); err == nil {
		fmt.Printf("\tSN:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(peblarRegProductNumber, 12); err == nil {
		fmt.Printf("\tPN:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(peblarRegFwIdentifier, 12); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", b)
	}
}
