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
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Ambibox is the Ambibox ambiCHARGE Home charger implementation.
type Ambibox struct {
	log         *util.Logger
	conn        *modbus.Connection
	mu          sync.Mutex
	current     float64
	inputBase   uint16
	holdingBase uint16
	input       func() ([]byte, error) // cached bulk read of the input register block
}

// input register offsets (relative to inputBase), 2 registers per value
const (
	ambiCurrentL1        = 4
	ambiCurrentL2        = 6
	ambiCurrentL3        = 8
	ambiVoltageL1        = 12
	ambiVoltageL2        = 14
	ambiVoltageL3        = 16
	ambiPowerAc          = 18  // int32
	ambiEnergyImport     = 26  // float
	ambiEnergyExport     = 28  // float
	ambiNumberPhases     = 36  // uint32
	ambiCapacity         = 48  // float
	ambiSoc              = 54  // float
	ambiSessionState     = 82  // uint32 (EvseState)
	ambiEvConnected      = 84  // bool
	ambiEnergyImportSess = 98  // float
	ambiReplugRequired   = 102 // bool
	ambiInputLength      = 104 // registers 0..103
)

// holding register offsets (relative to holdingBase), 2 registers per value
const (
	ambiTargetPower = 0 // int32 W (negative = charge)
	ambiWakeUp      = 2 // uint32 (1 = wake up)
)

// EvseState enum values (session state)
const (
	ambiStateSessionSetup = iota
	ambiStateAuthorization
	ambiStateChargeParameterDiscovery
	ambiStateCableCheck
	ambiStatePreCharge
	ambiStateChargeLoop
	ambiStatePostCharge
	ambiStatePaused
	ambiStateStopped
	ambiStateError
)

func init() {
	registry.AddCtx("ambibox", NewAmbiboxFromConfig)
}

// NewAmbiboxFromConfig creates an Ambibox charger from configuration
func NewAmbiboxFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Connector          int
	}{
		TcpSettings: modbus.TcpSettings{ID: 1},
		Connector:   1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Connector < 1 || cc.Connector > 10 {
		return nil, fmt.Errorf("invalid connector: %d", cc.Connector)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	return NewAmbibox(ctx, cc.TcpSettings, cc.Connector)
}

// NewAmbibox creates an Ambibox charger
func NewAmbibox(ctx context.Context, settings modbus.TcpSettings, connector int) (*Ambibox, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("ambibox")
	conn.Logger(log.TRACE)

	wb := &Ambibox{
		log:         log,
		conn:        conn,
		inputBase:   uint16(4000 + (connector-1)*200),
		holdingBase: uint16(3000 + (connector-1)*100),
	}

	// share a single bulk read of the input block across all decoders
	wb.input = util.Cached(func() ([]byte, error) {
		return wb.conn.ReadInputRegisters(wb.inputBase, ambiInputLength)
	}, time.Second)

	return wb, nil
}

// decode helpers - byte offset in the input block is registerOffset*2
func f32(b []byte, off int) float64 { return float64(encoding.Float32(b[off*2:])) }
func i32(b []byte, off int) int32   { return encoding.Int32(b[off*2:]) }
func u32(b []byte, off int) uint32  { return encoding.Uint32(b[off*2:]) }

// writeRegister writes a 32-bit value to a holding register (2 registers)
func (wb *Ambibox) writeRegister(offset uint16, value uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, value)
	_, err := wb.conn.WriteMultipleRegisters(wb.holdingBase+offset, 2, b)

	return err
}

// targetWatts converts a charging current (A) into the Ambibox targetPower
// setpoint (DCW), based on measured voltage and active phases (VA).
func (wb *Ambibox) targetWatts(current float64) float64 {
	phases := 3
	var v1, v2, v3 float64 = 230, 230, 230

	if b, err := wb.input(); err == nil {
		if p := u32(b, ambiNumberPhases); p >= 1 && p <= 3 {
			phases = int(p)
		}
		if v := f32(b, ambiVoltageL1); v > 0 {
			v1 = v
		}
		if v := f32(b, ambiVoltageL2); v > 0 {
			v2 = v
		}
		if v := f32(b, ambiVoltageL3); v > 0 {
			v3 = v
		}
	}

	sum := []float64{v1, v2, v3}
	var v float64
	for i := 0; i < phases; i++ {
		v += sum[i]
	}

	// negative = charge
	return -current * v
}

// setTargetPhaseCurrent writes the targetPower setpoint for the given current
func (wb *Ambibox) setTargetPhaseCurrent(current float64) error {
	power := int32(math.Round(wb.targetWatts(current)))
	return wb.writeRegister(ambiTargetPower, uint32(power))
}

// Status implements the api.Charger interface
func (wb *Ambibox) Status() (api.ChargeStatus, error) {
	b, err := wb.input()
	if err != nil {
		return api.StatusNone, err
	}

	if u32(b, ambiEvConnected) == 0 {
		return api.StatusA, nil
	}

	if u32(b, ambiSessionState) == ambiStateChargeLoop {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

var _ api.StatusReasoner = (*Ambibox)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *Ambibox) StatusReason() (api.Reason, error) {
	b, err := wb.input()
	if err != nil {
		return api.ReasonUnknown, err
	}

	if u32(b, ambiReplugRequired) != 0 {
		return api.ReasonDisconnectRequired, nil
	}
	if u32(b, ambiSessionState) == ambiStateAuthorization {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *Ambibox) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.holdingBase+ambiTargetPower, 2)
	if err != nil {
		return false, err
	}

	return encoding.Int32(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Ambibox) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	var current float64
	if enable {
		current = wb.current
	}

	return wb.setTargetPhaseCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *Ambibox) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Ambibox)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Ambibox) MaxCurrentMillis(current float64) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()

	if err := wb.setTargetPhaseCurrent(current); err != nil {
		return err
	}

	wb.current = current
	return nil
}

var _ api.Meter = (*Ambibox)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Ambibox) CurrentPower() (float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, err
	}

	return float64(i32(b, ambiPowerAc)), nil
}

var _ api.MeterEnergy = (*Ambibox)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Ambibox) TotalEnergy() (float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, err
	}

	return f32(b, ambiEnergyImport) / 1e3, nil
}

var _ api.MeterReturnEnergy = (*Ambibox)(nil)

// ReturnEnergy implements the api.MeterReturnEnergy interface
func (wb *Ambibox) ReturnEnergy() (float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, err
	}

	return f32(b, ambiEnergyExport) / 1e3, nil
}

var _ api.ChargeRater = (*Ambibox)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Ambibox) ChargedEnergy() (float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, err
	}

	return f32(b, ambiEnergyImportSess) / 1e3, nil
}

var _ api.PhaseCurrents = (*Ambibox)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Ambibox) Currents() (float64, float64, float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, 0, 0, err
	}

	return f32(b, ambiCurrentL1), f32(b, ambiCurrentL2), f32(b, ambiCurrentL3), nil
}

var _ api.PhaseVoltages = (*Ambibox)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Ambibox) Voltages() (float64, float64, float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, 0, 0, err
	}

	return f32(b, ambiVoltageL1), f32(b, ambiVoltageL2), f32(b, ambiVoltageL3), nil
}

var _ api.Battery = (*Ambibox)(nil)

// Soc implements the api.Battery interface
func (wb *Ambibox) Soc() (float64, error) {
	b, err := wb.input()
	if err != nil {
		return 0, err
	}

	return f32(b, ambiSoc), nil
}

var _ api.BatteryCapacity = (*Ambibox)(nil)

// Capacity implements the api.BatteryCapacity interface
func (wb *Ambibox) Capacity() float64 {
	b, err := wb.input()
	if err != nil {
		return 0
	}

	return f32(b, ambiCapacity) / 1e3
}

var _ api.Resurrector = (*Ambibox)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *Ambibox) WakeUp() error {
	return wb.writeRegister(ambiWakeUp, 1)
}
