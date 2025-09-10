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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// eSolutions eProWallbox charger implementation
type EProWallbox struct {
	conn *modbus.Connection
	log  *util.Logger
}

const (
	eproRegStatus         = 40101 // IEC 61851 Status, 1 register, UINT16
	eproRegEnable         = 40406 // On/Off state, 1 registers, UINT16
	eproRegCurrentLimit   = 40407 // in mA
	eproRegResetWatchdog  = 40502
	eproRegVoltages       = 40604 // L1 voltage in V, 2 registers, Float32 (followed by L2, L3)
	eproRegCurrents       = 40620 // L1 current in A, 2 registers, Float32 (followed by L2, L3)
	eproRegPowers         = 40636 // L1 power in W, 2 registers, Float32 (followed by L2, L3)
	eproRegActiveEnergies = 40658 // L1 energy in Wh, 2 registers, Float32 (followed by L2, L3)
)

func init() {
	registry.AddCtx("eprowallbox", NewEProWallboxFromConfig)
}

// NewEProWallboxFromConfig creates a eProWallbox charger from generic config
func NewEProWallboxFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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
		conn: conn,
		log:  log,
	}

	go wb.heartbeat(ctx)

	return wb, nil
}

func (wb *EProWallbox) heartbeat(ctx context.Context) {
	for tick := time.Tick(10 * time.Second); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, 0x5555)
		if _, err := wb.conn.WriteMultipleRegisters(eproRegResetWatchdog, 1, b); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *EProWallbox) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(eproRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	switch s {
	case 0, 1: // A1, A2
		return api.StatusA, nil
	case 2, 3, 4, 6: // B1, B2, C1, D1
		return api.StatusB, nil
	case 5, 7: // C2, D2
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *EProWallbox) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(eproRegEnable, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *EProWallbox) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(eproRegEnable, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *EProWallbox) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*EProWallbox)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *EProWallbox) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(current*1e3))

	_, err := wb.conn.WriteMultipleRegisters(eproRegCurrentLimit, 2, b)

	return err
}

// getPhaseValues returns 3 sequential register values
func (wb *EProWallbox) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = rs485.RTUIeee754ToFloat64(b[4*i:]) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.Meter = (*EProWallbox)(nil)

// CurrentPower implements the api.Meter interface
func (wb *EProWallbox) CurrentPower() (float64, error) {
	l1, l2, l3, err := wb.getPhaseValues(eproRegPowers, 1)
	return l1 + l2 + l3, err
}

var _ api.MeterEnergy = (*EProWallbox)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *EProWallbox) TotalEnergy() (float64, error) {
	l1, l2, l3, err := wb.getPhaseValues(eproRegActiveEnergies, 1000)
	return -(l1 + l2 + l3), err
}

var _ api.PhaseCurrents = (*EProWallbox)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *EProWallbox) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(eproRegCurrents, 1)
}

var _ api.PhaseVoltages = (*EProWallbox)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *EProWallbox) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(eproRegVoltages, 1)
}
