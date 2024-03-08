package charger

// LICENSE

// Copyright (c) 2019-2023 andig, premultiply

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
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Schneider charger implementation
type Schneider struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	schneiderRegEvState             = 1
	schneiderRegOcppStatus          = 150
	schneiderRegEvPresence          = 1150
	schneiderRegCurrents            = 2999
	schneiderRegVoltages            = 3027
	schneiderRegPower               = 3059
	schneiderRegEnergy              = 3203
	schneiderRegLifebit             = 4000
	schneiderRegSetCommand          = 4001
	schneiderRegCommandStatus       = 4002
	schneiderRegSetPoint            = 4004
	schneiderRegChargingTime        = 4007
	schneiderRegSessionChargingTime = 4009
	schneiderRegLastStopCause       = 4011

	schneiderDisabled = uint16(0)
)

func init() {
	registry.Add("schneider-v3", NewSchneiderV3FromConfig)
}

// https://download.schneider-electric.com/files?p_enDocType=Other+technical+guide&p_File_Name=GEX1969300-04.pdf&p_Doc_Ref=GEX1969300

// NewSchneiderV3FromConfig creates a Schneider charger from generic config
func NewSchneiderV3FromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSchneiderV3(cc.URI, cc.ID)
}

// NewSchneiderV3 creates Schneider charger
func NewSchneiderV3(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("schneider")
	conn.Logger(log.TRACE)

	wb := &Schneider{
		log:  log,
		conn: conn,
		curr: 6,
	}

	// get initial state from charger
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegSetPoint, 1)
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if u := encoding.Uint16(b); u > wb.curr {
		wb.curr = u
	}

	// heartbeat
	b, err = wb.conn.ReadHoldingRegisters(schneiderRegLifebit, 1)
	if err != nil {
		return nil, fmt.Errorf("heartbeat timeout: %w", err)
	}
	if u := encoding.Uint16(b); u != 2 {
		go wb.heartbeat(2 * time.Second)
	}

	return wb, nil
}

func (wb *Schneider) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.WriteSingleRegister(schneiderRegLifebit, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Schneider) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegEvState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := encoding.Uint16(b)

	switch s {
	case 0, 1, 2, 6:
		return api.StatusA, nil
	case 3, 4, 5, 7:
		return api.StatusB, nil
	case 8, 9:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Schneider) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegSetPoint, 1)
	if err != nil {
		return false, err
	}

	return encoding.Uint16(b) != schneiderDisabled, nil
}

// Enable implements the api.Charger interface
func (wb *Schneider) Enable(enable bool) error {
	u := schneiderDisabled
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(schneiderRegSetPoint, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Schneider) MaxCurrent(current int64) error {
	_, err := wb.conn.WriteSingleRegister(schneiderRegSetPoint, uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *Schneider) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32LswFirst(b)) * 1e3, nil
}

var _ api.MeterEnergy = (*Schneider)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Schneider) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint64LswFirst(b)) / 1e3, nil
}

var _ api.PhaseCurrents = (*Schneider)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Schneider) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(schneiderRegCurrents)
}

var _ api.PhaseVoltages = (*Schneider)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Schneider) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(schneiderRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *Schneider) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Float32LswFirst(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

var _ api.ChargeTimer = (*Schneider)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Schneider) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(schneiderRegSessionChargingTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint16(b)) * time.Second, nil
}

var _ api.Diagnosis = (*Schneider)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Schneider) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegEvState, 1); err == nil {
		fmt.Printf("\tevState:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegOcppStatus, 1); err == nil {
		fmt.Printf("\tOCPP Status:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegEvPresence, 1); err == nil {
		fmt.Printf("\tevPresence:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegLifebit, 1); err == nil {
		fmt.Printf("\tLifebit:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegSetCommand, 1); err == nil {
		fmt.Printf("\tSet command:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegCommandStatus, 2); err == nil {
		fmt.Printf("\tCommand status:\t\t%d\n", encoding.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegSetPoint, 1); err == nil {
		fmt.Printf("\tSet Point:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(schneiderRegLastStopCause, 1); err == nil {
		fmt.Printf("\tLast stop cause:\t%d\n", encoding.Uint16(b))
	}
}
