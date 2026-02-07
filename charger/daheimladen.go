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
)

// DaheimLaden charger implementation
type DaheimLaden struct {
	log    *util.Logger
	conn   *modbus.Connection
	curr   uint16
	phases uint16
}

const (
	dlRegChargingState   = 0   // Uint16 RO ENUM
	dlRegConnectorState  = 2   // Uint16 RO ENUM
	dlRegCurrents        = 6   // 3xUint16 plus placeholder RO 0.1A
	dlRegActivePower     = 12  // Uint32 RO 1W
	dlRegTotalEnergy     = 28  // Uint32 RO 0.1KWh
	dlRegEvseMaxCurrent  = 32  // Uint16 RO 0.1A
	dlRegCableMaxCurrent = 36  // Uint16 RO 0.1A
	dlRegStationId       = 38  // Chr[16] RO UTF16
	dlRegChargedEnergy   = 72  // Uint16 RO 0.1kWh
	dlRegChargingTime    = 78  // Uint32 RO 1s
	dlRegSafeCurrent     = 87  // Uint16 WR 0.1A
	dlRegCommTimeout     = 89  // Uint16 WR 1s
	dlRegCurrentLimit    = 91  // Uint16 WR 0.1A
	dlRegChargeControl   = 93  // Uint16 WR ENUM
	dlRegChargeCmd       = 95  // Uint16 WR ENUM
	dlRegVoltages        = 109 // 3xUint16 plus placeholder RO 0.1V

	// PRO only
	dlRegPhaseSwitchState   = 184
	dlRegPhaseSwitchControl = 186
	dlRegPhaseSwitchAction  = 188
)

func init() {
	registry.AddCtx("daheimladen", NewDaheimLadenFromConfig)
}

//go:generate go tool decorate -f decorateDaheimLaden -b *DaheimLaden -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewDaheimLadenFromConfig creates a DaheimLaden charger from generic config
func NewDaheimLadenFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Phases1p3p         bool
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewDaheimLaden(ctx, cc.URI, cc.ID, cc.Phases1p3p)
}

// NewDaheimLaden creates DaheimLaden charger
func NewDaheimLaden(ctx context.Context, uri string, id uint8, phases bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("daheimladen")
	conn.Logger(log.TRACE)

	wb := &DaheimLaden{
		log:    log,
		conn:   conn,
		curr:   60, // assume min current
		phases: 3,  // assume 3p
	}

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr > 0 {
		wb.curr = curr
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(dlRegCommTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(ctx, time.Duration(u)*time.Second/2)
	}

	var phases1p3p func(int) error
	var phasesG func() (int, error)
	if phases {
		phases1p3p = wb.phases1p3p
		phasesG = wb.getPhases
	}

	return decorateDaheimLaden(wb, phases1p3p, phasesG), nil
}

func (wb *DaheimLaden) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if _, err := wb.conn.ReadHoldingRegisters(dlRegSafeCurrent, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *DaheimLaden) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(dlRegCurrentLimit, 1, b)

	return err
}

func (wb *DaheimLaden) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// Status implements the api.Charger interface
func (wb *DaheimLaden) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegChargingState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	switch s {
	case 1: // Standby (A)
		return api.StatusA, nil
	case 2: // Connect (B1)
		return api.StatusB, nil
	case 3: // Start-up State (B2)
		return api.StatusB, nil
	case 4: // Charging (C)
		power, err := wb.CurrentPower()
		if power == 0 {
			return api.StatusB, err
		}
		return api.StatusC, err
	case 5: // Start-UP Fail (B2)
		return api.StatusB, nil
	case 6: // Session Terminated by EVSE
		return api.StatusB, nil
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *DaheimLaden) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr > 0, err
}

// Enable implements the api.Charger interface
func (wb *DaheimLaden) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	// break to avoid too fast commands
	time.Sleep(time.Second)

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *DaheimLaden) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*DaheimLaden)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *DaheimLaden) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	curr := uint16(current * 10)

	err := wb.setCurrent(curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}

var _ api.Meter = (*DaheimLaden)(nil)

// CurrentPower implements the api.Meter interface
func (wb *DaheimLaden) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*DaheimLaden)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *DaheimLaden) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

// getPhaseValues returns 3 sequential register values
func (wb *DaheimLaden) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 5)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		// 16-bit registers for currents/voltages spaced by 1 empty register
		res[i] = float64(binary.BigEndian.Uint16(b[4*i:])) / 10
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*DaheimLaden)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *DaheimLaden) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(dlRegCurrents)
}

var _ api.PhaseVoltages = (*DaheimLaden)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *DaheimLaden) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(dlRegVoltages)
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *DaheimLaden) phases1p3p(phases int) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(phases))

	_, err := wb.conn.WriteMultipleRegisters(dlRegPhaseSwitchControl, 1, b)
	if err == nil {
		wb.phases = uint16(phases)
	}
	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *DaheimLaden) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(dlRegPhaseSwitchAction, 1)
	if err != nil {
		return 0, err
	}

	// 0=standby, 1=changing, 2=changed
	if binary.BigEndian.Uint16(b) != 1 {
		b, err := wb.conn.ReadHoldingRegisters(dlRegPhaseSwitchState, 1)
		if err != nil {
			return 0, err
		}

		wb.phases = binary.BigEndian.Uint16(b)
	}

	return int(wb.phases), nil
}

var _ api.Diagnosis = (*DaheimLaden)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *DaheimLaden) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargingState, 1); err == nil {
		fmt.Printf("\tCharging Station State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegEvseMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCableMaxCurrent, 1); err == nil {
		fmt.Printf("\tCable Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegStationId, 16); err == nil {
		s, _ := utf16BEBytesAsString(b)
		fmt.Printf("\tStation ID:\t%s\n", s)
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegCurrentLimit, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargeControl, 1); err == nil {
		fmt.Printf("\tCharge Control:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(dlRegChargeCmd, 1); err == nil {
		fmt.Printf("\tCharge Command:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
