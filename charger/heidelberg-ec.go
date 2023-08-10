package charger

// LICENSE

// Copyright (c) 2019-2022 andig
// Copyright (c) 2022 premultiply

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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// HeidelbergEC charger implementation
type HeidelbergEC struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
	wakeup  bool
}

const (
	hecRegVehicleStatus  = 5   // Input
	hecRegCurrents       = 6   // Input 6,7,8
	hecRegTemperature    = 9   // Input
	hecRegVoltages       = 10  // Input 10,11,12
	hecRegPower          = 14  // Input
	hecRegEnergy         = 17  // Input
	hecRegTimeoutConfig  = 257 // Holding
	hecRegStandbyConfig  = 258 // Holding
	hecRegRemoteLock     = 259 // Holding
	hecRegAmpsConfig     = 261 // Holding
	hecRegFailSafeConfig = 262 // Holding

	hecStandbyDisabled = 4 // disable standby
)

func init() {
	registry.Add("heidelberg", NewHeidelbergECFromConfig)
}

// https://wallbox.heidelberg.com/wp-content/uploads/2021/05/EC_ModBus_register_table_20210222.pdf (newer)
// https://cdn.shopify.com/s/files/1/0101/2409/9669/files/heidelberg-energy-control-modbus.pdf (older)

// NewHeidelbergECFromConfig creates a HeidelbergEC charger from generic config
func NewHeidelbergECFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHeidelbergEC(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewHeidelbergEC creates HeidelbergEC charger
func NewHeidelbergEC(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("heidel")
	conn.Logger(log.TRACE)

	wb := &HeidelbergEC{
		log:     log,
		conn:    conn,
		current: 60, // assume min current
	}

	// disable standby to prevent comm loss
	if err := wb.set(hecRegStandbyConfig, hecStandbyDisabled); err != nil {
		return nil, err
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(hecRegTimeoutConfig, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Millisecond / 2)
	}

	return wb, nil
}

func (wb *HeidelbergEC) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.Status(); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *HeidelbergEC) set(reg, val uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)

	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

// Status implements the api.Charger interface
func (wb *HeidelbergEC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(hecRegVehicleStatus, 1)
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
		b, err := wb.conn.ReadHoldingRegisters(hecRegRemoteLock, 1)
		if err != nil {
			return api.StatusNone, err
		}

		// unlock
		if binary.BigEndian.Uint16(b) != 1 {
			if err := wb.set(hecRegRemoteLock, 1); err != nil {
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
func (wb *HeidelbergEC) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(hecRegAmpsConfig, 1)
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
func (wb *HeidelbergEC) Enable(enable bool) error {
	var cur uint16
	if enable {
		cur = wb.current
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(hecRegAmpsConfig, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *HeidelbergEC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*HeidelbergEC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *HeidelbergEC) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	cur := uint16(10 * current)

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, cur)

	_, err := wb.conn.WriteMultipleRegisters(hecRegAmpsConfig, 1, b)
	if err == nil {
		wb.current = cur
	}

	return err
}

var _ api.Meter = (*HeidelbergEC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *HeidelbergEC) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(hecRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.MeterEnergy = (*HeidelbergEC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *HeidelbergEC) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(hecRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

var _ api.PhaseCurrents = (*HeidelbergEC)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *HeidelbergEC) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(hecRegCurrents, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var curr [3]float64
	for l := 0; l < 3; l++ {
		curr[l] = float64(binary.BigEndian.Uint16(b[2*l:])) / 10
	}

	return curr[0], curr[1], curr[2], nil
}

var _ api.PhaseVoltages = (*HeidelbergEC)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *HeidelbergEC) Voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(hecRegVoltages, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var volt [3]float64
	for l := 0; l < 3; l++ {
		volt[l] = float64(binary.BigEndian.Uint16(b[2*l:]))
	}

	return volt[0], volt[1], volt[2], nil
}

var _ api.Diagnosis = (*HeidelbergEC)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *HeidelbergEC) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(hecRegTemperature, 1); err == nil {
		fmt.Printf("Temperature:\t%.1fC\n", float64(int16(binary.BigEndian.Uint16(b)))/10)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hecRegTimeoutConfig, 1); err == nil {
		fmt.Printf("Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hecRegStandbyConfig, 1); err == nil {
		fmt.Printf("Standby:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hecRegRemoteLock, 1); err == nil {
		fmt.Printf("Remote Lock:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hecRegFailSafeConfig, 1); err == nil {
		fmt.Printf("FailSafe:\t%d\n", binary.BigEndian.Uint16(b))
	}
}

var _ api.Resurrector = (*HeidelbergEC)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *HeidelbergEC) WakeUp() error {
	// force status F by locking
	if err := wb.set(hecRegRemoteLock, 0); err == nil {
		// Takes at least ~10 sec to return to normal operation
		// after locking even if unlocking immediately.
		wb.wakeup = true
	}

	// return to normal operation by unlocking after ~10 sec
	return wb.set(hecRegRemoteLock, 1)
}
