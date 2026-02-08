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
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Weidmüller charger implementation
type Weidmüller struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	wmRegCarStatus    = 301   // GD_ID_EVCC_CAR_STATE CHAR
	wmRegEvccStatus   = 302   // GD_ID_EVCC_EVSE_STATE UINT16
	wmRegPhases       = 318   // GD_ID_EVCC_PHASES_LLM UINT16
	wmRegVoltages     = 400   // GD_ID_CM_VOLTAGE_PHASE UINT32
	wmRegCurrents     = 406   // GD_ID_CM_CURRENT_PHASE UINT32
	wmRegActivePower  = 418   // GD_ID_CM_ACTIVE_POWER UINT32
	wmRegTotalEnergy  = 457   // GD_ID_CM_CONSUMED_ENERGY_TOTAL_WH UINT64
	wmRegCardId       = 1000  // GD_ID_RFID_TAG_UID CHAR[21]
	wmRegTimeout      = 11050 // GD_ID_LCM_TIMEOUT UINT32
	wmRegCurrentLimit = 11052 // GD_ID_LCM_ACTUAL_CURRENT_LIMIT UINT16 (A)

	wmHeartbeatInterval = 5 * time.Second
	wmTimeout           = 65535 // ms
)

func init() {
	registry.AddCtx("weidmüller", NewWeidmüllerFromConfig)
}

// NewWeidmüllerFromConfig creates a Weidmüller charger from generic config
func NewWeidmüllerFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWeidmüller(ctx, cc.URI, cc.ID)
}

//go:generate go tool decorate -f decorateWeidmüller -b *Weidmüller -r api.Charger -t "api.MeterEnergy,TotalEnergy,func() (float64, error)"

// NewWeidmüller creates Weidmüller charger
func NewWeidmüller(ctx context.Context, uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("weidmüller")
	conn.Logger(log.TRACE)

	wb := &Weidmüller{
		log:  log,
		conn: conn,
		curr: 6, // assume min current
	}

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr != 0 {
		wb.curr = curr
	}

	// failsafe
	go wb.heartbeat(ctx, wmHeartbeatInterval)

	// check presence of energy meter
	if b, err := wb.conn.ReadHoldingRegisters(wmRegTotalEnergy, 2); err == nil && binary.BigEndian.Uint32(b) > 0 {
		return decorateWeidmüller(wb, wb.totalEnergy), nil
	}

	return wb, nil
}

func (wb *Weidmüller) heartbeat(ctx context.Context, timeout time.Duration) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, wmTimeout)

	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if _, err := wb.conn.WriteMultipleRegisters(wmRegTimeout, 2, b); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *Weidmüller) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(wmRegCurrentLimit, 1, b)

	return err
}

func (wb *Weidmüller) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// getPhaseValues returns 3 sequential register values
func (wb *Weidmüller) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Uint32LswFirst(b[4*i:])) / 1e3
	}

	return res[0], res[1], res[2], nil
}

// Status implements the api.Charger interface
func (wb *Weidmüller) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegCarStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := string(b[1]); s {
	case "A", "B", "C":
		return api.ChargeStatus(s), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Weidmüller) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr != 0, err
}

// Enable implements the api.Charger interface
func (wb *Weidmüller) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *Weidmüller) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	wb.curr = uint16(current)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*Weidmüller)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Weidmüller) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32LswFirst(b)) / 1e3, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Weidmüller) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32LswFirst(b)) / 1e3, err
}

var _ api.PhaseCurrents = (*Weidmüller)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Weidmüller) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(wmRegCurrents)
}

var _ api.PhaseVoltages = (*Weidmüller)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Weidmüller) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(wmRegVoltages)
}

var _ api.Identifier = (*Weidmüller)(nil)

// Identify implements the api.Identifier interface
func (wb *Weidmüller) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(wmRegCardId, 11)
	if err != nil {
		return "", err
	}
	return bytesAsString(b), nil
}

var _ api.PhaseSwitcher = (*Weidmüller)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Weidmüller) Phases1p3p(phases int) error {
	b := make([]byte, 2)

	if phases == 3 {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(wmRegPhases, 1, b)

	return err
}
