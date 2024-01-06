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
	"encoding/hex"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Amperfied charger implementation
type Amperfied struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
	wakeup  bool
}

const (
	ampRegChargingState  = 5    // Input
	ampRegCurrents       = 6    // Input 6,7,8
	ampRegTemperature    = 9    // Input
	ampRegVoltages       = 10   // Input 10,11,12
	ampRegPower          = 14   // Input
	ampRegEnergy         = 17   // Input
	ampRegTimeoutConfig  = 257  // Holding
	ampRegRemoteLock     = 259  // Holding
	ampRegAmpsConfig     = 261  // Holding
	ampRegFailSafeConfig = 262  // Holding
	ampRegRfidUID        = 2002 // Input
)

func init() {
	registry.Add("amperfied", NewAmperfiedFromConfig)
}

// NewAmperfiedFromConfig creates a Amperfied charger from generic config
func NewAmperfiedFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAmperfied(cc.URI, cc.ID)
}

// NewAmperfied creates Amperfied charger
func NewAmperfied(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("amperfied")
	conn.Logger(log.TRACE)

	wb := &Amperfied{
		log:     log,
		conn:    conn,
		current: 60, // assume min current
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(ampRegTimeoutConfig, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Millisecond / 2)
	}

	return wb, nil
}

func (wb *Amperfied) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.Status(); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *Amperfied) set(reg, val uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)

	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

// Status implements the api.Charger interface
func (wb *Amperfied) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(ampRegChargingState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	sb := binary.BigEndian.Uint16(b)

	if sb != 10 {
		wb.wakeup = false
	}

	switch sb {
	case 2, 3:
		return api.StatusA, nil
	case 4, 5:
		return api.StatusB, nil
	case 6, 7:
		return api.StatusC, nil
	case 8:
		return api.StatusD, nil
	case 9:
		return api.StatusE, nil
	case 10:
		// ensure RemoteLock is disabled after wake-up
		b, err := wb.conn.ReadHoldingRegisters(ampRegRemoteLock, 1)
		if err != nil {
			return api.StatusNone, err
		}

		// unlock
		if binary.BigEndian.Uint16(b) != 1 {
			if err := wb.set(ampRegRemoteLock, 1); err != nil {
				return api.StatusNone, err
			}
		}

		// keep status B2 during wakeup
		if wb.wakeup {
			return api.StatusB, nil
		}

		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", sb)
	}
}

// Enabled implements the api.Charger interface
func (wb *Amperfied) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(ampRegAmpsConfig, 1)
	if err != nil {
		return false, err
	}

	cur := binary.BigEndian.Uint16(b)

	enabled := cur != 0
	if enabled {
		wb.current = cur
	}

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *Amperfied) Enable(enable bool) error {
	var cur uint16
	if enable {
		cur = wb.current
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(ampRegAmpsConfig, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Amperfied) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Amperfied)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Amperfied) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	cur := uint16(10 * current)

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(ampRegAmpsConfig, 1, b)
	if err == nil {
		wb.current = cur
	}

	return err
}

var _ api.Meter = (*Amperfied)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Amperfied) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(ampRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.MeterEnergy = (*Amperfied)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Amperfied) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(ampRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// getPhaseValues returns 3 sequential register values
func (wb *Amperfied) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Amperfied)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Amperfied) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(ampRegCurrents, 10)
}

var _ api.PhaseVoltages = (*Amperfied)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Amperfied) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(ampRegVoltages, 1)
}

var _ api.Identifier = (*Amperfied)(nil)

// identify implements the api.Identifier interface
func (wb *Amperfied) Identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(ampRegRfidUID, 6)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

var _ api.Diagnosis = (*Amperfied)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Amperfied) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(ampRegTemperature, 1); err == nil {
		fmt.Printf("Temperature:\t%.1fC\n", float64(int16(binary.BigEndian.Uint16(b)))/10)
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegTimeoutConfig, 1); err == nil {
		fmt.Printf("Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegRemoteLock, 1); err == nil {
		fmt.Printf("Remote Lock:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegFailSafeConfig, 1); err == nil {
		fmt.Printf("FailSafe:\t%d\n", binary.BigEndian.Uint16(b))
	}
}

var _ api.Resurrector = (*Amperfied)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *Amperfied) WakeUp() error {
	// force status F by locking
	if err := wb.set(ampRegRemoteLock, 0); err == nil {
		// Takes at least ~10 sec to return to normal operation
		// after locking even if unlocking immediately.
		wb.wakeup = true
	}

	// return to normal operation by unlocking after ~10 sec
	return wb.set(ampRegRemoteLock, 1)
}
