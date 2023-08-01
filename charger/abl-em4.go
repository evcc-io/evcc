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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	// per-unit registers
	abl4Offset = 0x100

	abl4RegCurrents   = 0x3001 // 0.1A (3x)
	abl4RegVoltages   = 0x3007 // 0.1V (3x)
	abl4RegPower      = 0x300d // 1W
	abl4RegEnergy     = 0x300f // 0.01kWh
	abl4RegStatus     = 0x3031
	abl4RegMaxCurrent = 0x3032 // 0.1A
)

// AblEm4 is an api.Charger implementation for ABL eM4 controller
type AblEm4 struct {
	log     *util.Logger
	conn    *modbus.Connection
	base    uint16
	current uint16
}

func init() {
	registry.Add("abl-em4", NewAblEm4FromConfig)
}

// NewAblEm4FromConfig creates an ABL eM4 charger from generic config
func NewAblEm4FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Connector          uint16
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 255, // default
		},
		Connector: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAblEm4(cc.URI, cc.ID, cc.Connector)
}

// NewAblEm4 creates an ABL eM4 charger
func NewAblEm4(uri string, id uint8, connector uint16) (*AblEm4, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("abl-em4")
	conn.Logger(log.TRACE)

	wb := &AblEm4{
		log:     log,
		conn:    conn,
		current: 60, // assume min current
	}

	wb.base = (connector - 1) * abl4Offset

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr > 0 {
		wb.current = curr
	}

	return wb, nil
}

func (wb *AblEm4) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(wb.base+abl4RegMaxCurrent, 1, b)

	return err
}

func (wb *AblEm4) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+abl4RegMaxCurrent, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// Status implements the api.Charger interface
func (wb *AblEm4) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+abl4RegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	r := rune(b[1]>>4-0x0A) + 'A'

	switch r {
	case 'A', 'B', 'C':
		return api.ChargeStatus(r), nil
	default:
		status, ok := ablStatus[b[1]]
		if !ok {
			status = string(r)
		}

		return api.StatusNone, fmt.Errorf("invalid status: %s", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *AblEm4) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr > 0, err
}

// Enable implements the api.Charger interface
func (wb *AblEm4) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.current
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *AblEm4) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*AblEm4)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *AblEm4) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	wb.current = uint16(current * 10)

	return wb.setCurrent(wb.current)
}

var _ api.Meter = (*AblEm4)(nil)

// currentPower implements the api.Meter interface
func (wb *AblEm4) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+abl4RegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.MeterEnergy = (*AblEm4)(nil)

// totalEnergy implements the api.MeterEnergy interface
func (wb *AblEm4) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+abl4RegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) * 0.01, nil
}

// getPhaseValues returns 3 sequential register values
func (wb *AblEm4) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3*2)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) * 0.1
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*AblEm4)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *AblEm4) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(wb.base + abl4RegCurrents)
}

var _ api.PhaseVoltages = (*AblEm4)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *AblEm4) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(wb.base + abl4RegVoltages)
}
