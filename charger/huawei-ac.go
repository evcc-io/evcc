package charger

// LICENSE

// Copyright (c) 2023 premultiply

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
	"encoding/binary"
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

type HuaweiACCharger struct {
	conn *modbus.Connection
	curr float64
	lp   loadpoint.API
}

const (
	huaweiRegVoltages = 0x1000 // uint32 * 3
	huaweiRegCurrents = 0x1006 // uint32 * 3
	huaweiRegPower    = 0x100c // uint32
	huaweiRegStatus   = 0x100e // uint32
	huaweiRegMaxPower = 0x2000 // uint32
	huaweiRegControl  = 0x2006 // uint16
)

func init() {
	registry.Add("huawei-ac", NewHuaweiACChargerFromConfig)
}

// NewHuaweiACChargerFromConfig creates a HuaweiACCharger charger from generic config
func NewHuaweiACChargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHuaweiACCharger(cc.URI, cc.ID)
}

// NewHuaweiACCharger creates a HuaweiACCharger charger
func NewHuaweiACCharger(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("huawei-ac")
	conn.Logger(log.TRACE)

	wb := &HuaweiACCharger{
		conn: conn,
		curr: 6,
	}

	return wb, err
}

func (wb *HuaweiACCharger) setCurrent(current float64) error {
	var phases int
	// get (expectedly) active phases from loadpoint
	if wb.lp != nil {
		phases = wb.lp.GetPhases()
	}
	if phases == 0 {
		phases = 3
	}

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(math.Trunc(230.0*current*float64(phases)*10.0)))

	_, err := wb.conn.WriteMultipleRegisters(huaweiRegMaxPower, 2, b)

	return err
}

// Status implements the api.Charger interface
func (wb *HuaweiACCharger) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(huaweiRegControl, 1)
	if err != nil {
		return api.StatusNone, err
	}

	//ToDo: Real status
	switch u := binary.BigEndian.Uint16(b); u {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *HuaweiACCharger) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(huaweiRegMaxPower, 2)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint32(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *HuaweiACCharger) Enable(enable bool) error {
	var curr float64
	if enable {
		curr = wb.curr
	}

	return wb.setCurrent(curr)
}

// MaxCurrent implements the api.Charger interface
func (wb *HuaweiACCharger) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*HuaweiACCharger)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *HuaweiACCharger) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	err := wb.setCurrent(current)
	if err == nil {
		wb.curr = current
	}

	return err
}

var _ api.Meter = (*HuaweiACCharger)(nil)

// currentPower implements the api.Meter interface
func (wb *HuaweiACCharger) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(huaweiRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.PhaseCurrents = (*HuaweiACCharger)(nil)

// currents implements the api.PhaseCurrents interface
func (wb *HuaweiACCharger) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(huaweiRegCurrents)
}

var _ api.PhaseVoltages = (*HuaweiACCharger)(nil)

// voltages implements the api.PhaseVoltages interface
func (wb *HuaweiACCharger) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(huaweiRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *HuaweiACCharger) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 10
	}

	return res[0], res[1], res[2], nil
}
