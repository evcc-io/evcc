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
	"strings"

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
	solaxRegSerialNumber   = 0x0600 // 7x string
	solaxRegDeviceMode     = 0x060D // uint16
	solaxRegCommandControl = 0x0627 // uint16
	solaxRegMaxCurrent     = 0x0628 // uint16 0.01A
	solaxRegPhaseSwitch    = 0xA105 // uint16

	// input (FC 0x04)
	solaxRegVoltages           = 0x0000 // 3x uint16 0.01V
	solaxRegCurrents           = 0x0004 // 3x uint16 0.01A
	solaxRegActivePower        = 0x000B // uint16 1W
	solaxRegTotalEnergy        = 0x0010 // uint32 0.1kWh
	solaxRegState              = 0x001D // uint16
	solaxRegFaultCode          = 0x001E // 2x uint32
	solaxRegFirmwareVersion    = 0x0025 // uint16 Vx.xx
	solaxRegConnectionStrength = 0x0027 // uint16 1%
	solaxRegLockState          = 0x002D // uint16
	solaxRegPhases             = 0xA02A // uint16

	// commands
	solaxCmdStop  = 3
	solaxCmdStart = 4

	// modes
	solaxModeStop = 0
	solaxModeFast = 1
	solaxModeECO  = 2

	// minimum firmware version for phase switching support
	solaxFirmwarePhaseSwitching = 905
)

func init() {
	registry.AddCtx("solax", NewSolaxG1FromConfig)
	registry.AddCtx("solax-g2", NewSolaxG2FromConfig)
}

//go:generate go tool decorate -f decorateSolax -b *Solax -r api.Charger -t api.PhaseSwitcher,api.PhaseGetter

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

	var phases1p3p func(int) error
	var phasesG func() (int, error)

	if b, err := wb.conn.ReadInputRegisters(solaxRegFirmwareVersion, 1); err == nil {
		v := encoding.Uint16(b)
		if !wb.isLegacyHw && v >= solaxFirmwarePhaseSwitching {
			phases1p3p = wb.phases1p3p
			phasesG = wb.getPhases
		}
	}

	return decorateSolax(wb, phases1p3p, phasesG), nil
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
		1,  // "Preparing"
		3,  // "Finishing"
		7,  // "SuspendedEV"
		8,  // "SuspendedEVSE"
		11, // "StartDelay"
		12, // "ChargPause"
		13, // "Stopping"
		17: // "PhaseSwitching"
		return api.StatusB, nil
	case 2: // "Charging"
		return api.StatusC, nil
	case 4: // "Fault"
		return api.StatusE, nil
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

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Solax) phases1p3p(phases int) error {
	u := uint16(1)
	if phases == 3 {
		u = 2
	}

	_, err := wb.conn.WriteSingleRegister(solaxRegPhaseSwitch, u)

	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *Solax) getPhases() (int, error) {
	b, err := wb.conn.ReadInputRegisters(solaxRegPhases, 1)
	if err != nil {
		return 0, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 1:
		return 1, nil
	case 2:
		return 3, nil
	default:
		return 0, nil
	}
}

var _ api.Diagnosis = (*Solax)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Solax) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(solaxRegSerialNumber, 7); err == nil {
		fmt.Printf("\tSerial Number:\t%s\n", bytesAsString(b))
	}
	if b, err := wb.conn.ReadInputRegisters(solaxRegFirmwareVersion, 1); err == nil {
		v := encoding.Uint16(b)
		fmt.Printf("\tFirmware Version:\tV%d.%02d\n", v/100, v%100)
	}
	if b, err := wb.conn.ReadInputRegisters(solaxRegConnectionStrength, 1); err == nil {
		fmt.Printf("\tConnection Strength (RSSI):\t%d%%\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(solaxRegFaultCode, 2); err == nil {
		code := binary.BigEndian.Uint32(b)
		fmt.Printf("\tFault Code:\t0x%08X", code)

		// Collect all set bits
		var setBits []string
		for bitIndex := range 32 {
			if (code & (1 << bitIndex)) != 0 { // Check if the bit is set
				setBits = append(setBits, fmt.Sprintf("%d", bitIndex+1)) // Add the 1-based bit number
			}
		}
		if len(setBits) > 0 {
			fmt.Printf(", Set Bits: %s\n", strings.Join(setBits, ","))
		} else {
			fmt.Printf(", Set Bits: None\n")
		}
	}
	if b, err := wb.conn.ReadInputRegisters(solaxRegLockState, 1); err == nil {
		switch state := encoding.Uint16(b); state {
		case 0:
			fmt.Printf("\tLock State:\tUnlocked (%d)\n", state)
		case 1:
			fmt.Printf("\tLock State:\tLocked (%d)\n", state)
		default:
			fmt.Printf("\tLock State:\tUnknown (%d)\n", state)
		}
	}
	if b, err := wb.conn.ReadHoldingRegisters(solaxRegDeviceMode, 1); err == nil {
		switch state := encoding.Uint16(b); state {
		case solaxModeStop:
			fmt.Printf("\tDevice Mode:\tStop (%d)\n", state)
		case solaxModeFast:
			fmt.Printf("\tDevice Mode:\tFast (%d)\n", state)
		case solaxModeECO:
			fmt.Printf("\tDevice Mode:\tECO (%d)\n", state)
		default:
			fmt.Printf("\tDevice Mode:\tUnknown (%d)\n", state)
		}
	}
}
