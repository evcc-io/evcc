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

// https://blog.chargex.de/hubfs/Customer%20Support/ChargeX%20Modbus%20TCP%20documentation.pdf

// ChargeX charger implementation
type ChargeX struct {
	log       *util.Logger
	conn      *modbus.Connection
	connector uint16
	curr      float64
}

const (
	// Module specific base address (module X: 100 + module_index*12)
	chargexRegModuleBase     = 100 // 0x0064 Base address for module 0
	chargexRegModulePower    = 0   // PAC_X offset
	chargexRegModuleCurrent1 = 2   // IAC_SUM_1_X offset
	chargexRegModuleCurrent2 = 4   // IAC_SUM_2_X offset
	chargexRegModuleCurrent3 = 6   // IAC_SUM_3_X offset
	chargexRegModuleState    = 8   // States_CP_X offset

	// Holding registers (read/write)
	chargexRegTargetPower  = 504 // 0x01F8 PAC_Target_Power (W) - U32
	chargexRegChargingMode = 506 // 0x01FA Charging_Mode (0=Full, 1=Min, 2=NoRed) - U32
)

func init() {
	registry.AddCtx("chargex", NewChargeXFromConfig)
}

// NewChargeXFromConfig creates a ChargeX charger from generic config
func NewChargeXFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Connector          uint16
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 10,
		},
		Connector: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewChargeX(ctx, cc.URI, cc.ID, cc.Connector)
}

// NewChargeX creates ChargeX charger
func NewChargeX(ctx context.Context, uri string, id uint8, connector uint16) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("chargex")
	conn.Logger(log.TRACE)

	wb := &ChargeX{
		log:       log,
		conn:      conn,
		connector: connector,
		curr:      6, // assume min current
	}

	// Initialize charging mode to 0 (Full control)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, 0)
	if _, err := conn.WriteMultipleRegisters(chargexRegChargingMode, 2, b); err != nil {
		return nil, fmt.Errorf("failed to initialize charging mode: %w", err)
	}

	return wb, nil
}

// moduleReg returns the register address for a module-specific register
func (wb *ChargeX) moduleReg(offset uint16) uint16 {
	// connector is 1-indexed, convert to 0-indexed module_index
	return chargexRegModuleBase + ((wb.connector - 1) * 12) + offset
}

// setCurrent writes the current limit in Amperes
func (wb *ChargeX) setCurrent(current float64) error {
	// Read module state to determine charging mode (1p or 3p)
	b, err := wb.conn.ReadHoldingRegisters(wb.moduleReg(chargexRegModuleState), 2)
	if err != nil {
		return err
	}

	state := binary.BigEndian.Uint32(b)
	// Bit 1: ChMode - 0=single phase, 1=3 phase
	phases := 3
	if (state & (1 << 1)) == 0 {
		phases = 1
	}

	b = make([]byte, 4)
	targetPower := uint32(230 * current * float64(phases))
	binary.BigEndian.PutUint32(b, targetPower)

	wb.log.DEBUG.Printf("set charge power: %dW (%.1fA, %dp)", targetPower, current, phases)

	_, err = wb.conn.WriteMultipleRegisters(chargexRegTargetPower, 2, b)
	return err
}

// Status implements the api.Charger interface
func (wb *ChargeX) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.moduleReg(chargexRegModuleState), 2)
	if err != nil {
		return api.StatusNone, err
	}

	state := binary.BigEndian.Uint32(b)

	// Bit 0: Charging (1=charging, 0=not charging)
	// Bit 1: ChMode (0=single phase, 1=3 phase)
	// Bit 2: Auth (1=authorized, 0=not authorized)
	// Bit 3: EV (1=vehicle connected, 0=not connected)
	// Bit 4: Req (1=requesting charge, 0=not requesting)
	// Bit 5: Battery Full (1=battery full, 0=not full)

	connected := (state & (1 << 3)) != 0
	charging := (state & (1 << 0)) != 0

	switch {
	case charging:
		return api.StatusC, nil // Charging
	case connected:
		return api.StatusB, nil // Vehicle connected
	default:
		return api.StatusA, nil // No vehicle
	}
}

var _ api.StatusReasoner = (*ChargeX)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *ChargeX) StatusReason() (api.Reason, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.moduleReg(chargexRegModuleState), 2)
	if err != nil {
		return api.ReasonUnknown, err
	}

	// Check if not authorized
	if (binary.BigEndian.Uint32(b) & (1 << 2)) == 0 {
		return api.ReasonWaitingForAuthorization, nil
	}

	return api.ReasonUnknown, nil
}

// Enabled implements the api.Charger interface
func (wb *ChargeX) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargexRegTargetPower, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *ChargeX) Enable(enable bool) error {
	var current float64
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *ChargeX) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*ChargeX)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *ChargeX) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	err := wb.setCurrent(current)
	if err == nil {
		wb.curr = current
	}

	return err
}

var _ api.Meter = (*ChargeX)(nil)

// CurrentPower implements the api.Meter interface
func (wb *ChargeX) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.moduleReg(chargexRegModulePower), 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.PhaseCurrents = (*ChargeX)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *ChargeX) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.moduleReg(chargexRegModuleCurrent1), 6)
	if err != nil {
		return 0, 0, 0, err
	}

	// Values are in mA, convert to A
	l1 := float64(binary.BigEndian.Uint32(b[0:4])) / 1e3
	l2 := float64(binary.BigEndian.Uint32(b[4:8])) / 1e3
	l3 := float64(binary.BigEndian.Uint32(b[8:12])) / 1e3

	return l1, l2, l3, nil
}
