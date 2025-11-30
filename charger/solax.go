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
	"github.com/volkszaehler/mbmd/encoding"
)

// Solax charger implementation
type Solax struct {
	log        *util.Logger
	conn       *modbus.Connection
	isLegacyHw bool
}

const (
	// holding (FC 0x03, 0x06, 0x10)
	solaxRegDeviceMode      = 0x060D // uint16
	solaxRegStartChargeMode = 0x0610 // uint16
	solaxRegPhases          = 0x0625 // uint16
	solaxRegCommandControl  = 0x0627 // uint16
	solaxRegMaxCurrent      = 0x0628 // uint16 0.01A

	// input (FC 0x04)
	solaxRegVoltages    = 0x0000 // 3x uint16 0.01V
	solaxRegCurrents    = 0x0004 // 3x uint16 0.01A
	solaxRegActivePower = 0x000B // uint16 1W
	solaxRegTotalEnergy = 0x0010 // uint32s 0.1kWh
	solaxRegState       = 0x001D // uint16

	solaxCmdStop  = 3
	solaxCmdStart = 4

	solaxModeStop = 0
	solaxModeFast = 1
)

func init() {
	registry.AddCtx("solax", NewSolaxG1FromConfig)
	registry.AddCtx("solax-g2", NewSolaxG2FromConfig)
}

func NewSolaxG1FromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	return NewSolaxFromConfig(ctx, other, true)
}

func NewSolaxG2FromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	return NewSolaxFromConfig(ctx, other, false)
}

// NewSolaxFromConfig creates a Solax charger from generic config
func NewSolaxFromConfig(ctx context.Context, other map[string]any, isLegacyHw bool) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSolax(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID, isLegacyHw)
}

// NewSolax creates Solax charger
func NewSolax(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, id uint8, isLegacyHw bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("solax")
	conn.Logger(log.TRACE)

	wb := &Solax{
		log:        log,
		conn:       conn,
		isLegacyHw: isLegacyHw,
	}

	return wb, err
}

// getPhaseValues returns 3 sequential register values
func (wb *Solax) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / 100
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Solax) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := encoding.Uint16(b); s {
	case
		0, // "Available"
		5: // "Unavailable"
		return api.StatusA, nil
	case
		1, // "Preparing"
		8, // "SuspendedEVSE"
		7, // "SuspendedEV"
		3: // "Finishing"
		return api.StatusB, nil
	case 2: // "Charging"
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Solax) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(solaxRegDeviceMode, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != solaxModeStop, nil
}

// Enable implements the api.Charger interface
func (wb *Solax) Enable(enable bool) error {
	var cmd uint16 = solaxCmdStop
	if enable {
		cmd = solaxCmdStart
	}

	_, err := wb.conn.WriteSingleRegister(solaxRegCommandControl, cmd)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Solax) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Solax)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Solax) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	_, err := wb.conn.WriteSingleRegister(solaxRegMaxCurrent, uint16(current*100))

	return err
}

var _ api.Meter = (*Solax)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Solax) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegActivePower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), err
}

var _ api.MeterEnergy = (*Solax)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Solax) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	if wb.isLegacyHw {
		return float64(binary.BigEndian.Uint32(b)) / 10, err
	}

	return float64(encoding.Uint32LswFirst(b)) / 10, err
}

var _ api.PhaseCurrents = (*Solax)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Solax) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(solaxRegCurrents)
}

var _ api.PhaseVoltages = (*Solax)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Solax) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(solaxRegVoltages)
}

/* https://github.com/evcc-io/evcc/pull/14108
var _ api.PhaseSwitcher = (*Solax)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Solax) Phases1p3p(phases int) error {
	var u uint16

	if phases == 1 {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(solaxRegPhases, u)

	return err
}
*/
