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

// https://api.library.loxone.com/downloader/file/2425/Modbus%20Protocol%20RAEDIAN%20AC%20Wallbox%20v0.3.pdf

// Raedian charger implementation
type Raedian struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint32 // mA
}

const (
	raedianRegStatus        = 0x800C // uint32 RO ENUM
	raedianRegCurrents      = 0x8010 // uint32 RO mA
	raedianRegVoltages      = 0x8016 // uint32 RO 0.1V
	raedianRegPower         = 0x801C // uint32 RO W
	raedianRegChargedEnergy = 0x801E // uint32 RO Wh
	raedianRegMaxCurrent    = 0x8100 // uint32 WR mA
	raedianRegPhases        = 0x8102 // uint16 WO
)

func init() {
	registry.AddCtx("raedian", NewRaedianFromConfig)
}

// NewRaedianFromConfig creates a Raedian charger from generic config
func NewRaedianFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewRaedian(ctx, cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.Protocol(), cc.ID)
}

// NewRaedian creates Raedian charger
func NewRaedian(ctx context.Context, uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (*Raedian, error) {
	conn, err := modbus.NewConnection(ctx, uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("raedian")
	conn.Logger(log.TRACE)

	wb := &Raedian{
		log:  log,
		conn: conn,
		curr: 6000, // assume min current
	}

	if b, err := wb.conn.ReadHoldingRegisters(raedianRegMaxCurrent, 2); err == nil {
		if cur := binary.BigEndian.Uint32(b); cur >= 6000 {
			wb.curr = cur
		}
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Raedian) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegStatus, 2)
	if err != nil {
		return api.StatusNone, err
	}

	// Register layout: A3 A2 A1 A0 (big-endian)
	// A1 (b[2]): Bit7 = current limit flag, Bits 6~0 = charging state
	state := b[2] & 0x7F
	switch state {
	case 0x00:
		return api.StatusA, nil
	case 0x01, 0x02:
		return api.StatusB, nil
	case 0x03, 0x04:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %#x", state)
	}
}

var _ api.StatusReasoner = (*Raedian)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *Raedian) StatusReason() (api.Reason, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegStatus, 2)
	if err != nil {
		return api.ReasonUnknown, err
	}

	// A1 (b[2]) Bits 6~0: 0x01 = State B1, pending authorization
	if b[2]&0x7F == 0x01 {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *Raedian) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegMaxCurrent, 2)
	if err != nil {
		return false, err
	}

	cur := binary.BigEndian.Uint32(b)
	if cur != 0 {
		wb.curr = cur
	}

	return cur != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Raedian) Enable(enable bool) error {
	var cur uint32
	if enable {
		cur = wb.curr
	}

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(raedianRegMaxCurrent, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Raedian) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Raedian)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Raedian) MaxCurrentMillis(current float64) error {
	curr := uint32(current * 1000) // convert A to mA

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, curr)

	_, err := wb.conn.WriteMultipleRegisters(raedianRegMaxCurrent, 2, b)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*Raedian)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Raedian) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.ChargeRater = (*Raedian)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Raedian) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1000, nil
}

// getPhaseValues reads 3 sequential uint32 registers in a single Modbus operation
func (wb *Raedian) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
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

var _ api.PhaseCurrents = (*Raedian)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Raedian) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(raedianRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*Raedian)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Raedian) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(raedianRegVoltages, 10)
}

var _ api.PhaseSwitcher = (*Raedian)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Raedian) Phases1p3p(phases int) error {
	u := uint16(0x01)
	if phases == 3 {
		u = 0x02
	}

	_, err := wb.conn.WriteSingleRegister(raedianRegPhases, u)
	return err
}
