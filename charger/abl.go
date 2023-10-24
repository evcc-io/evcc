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

// ABLeMH charger implementation
type ABLeMH struct {
	conn *modbus.Connection
	curr uint16
}

const (
	ablRegFirmware    = 0x01
	ablRegStatus      = 0x04
	ablRegModifyState = 0x05
	ablRegEnabled     = 0x0F
	ablRegAmpsConfig  = 0x14
	ablRegStatusLong  = 0x2E

	ablAmpsDisabled uint16 = 0x03E8

	ablSensorPresent = 1 << 5
)

var ablStatus = map[byte]string{
	0xA1: "Waiting for EV",
	0xB1: "EV is asking for charging",
	0xB2: "EV has the permission to charge",
	0xC2: "EV is charged",
	0xC3: "C2, reduced current (error F16, F17)",
	0xC4: "C2, reduced current (imbalance F15)",
	0xE0: "Outlet disabled",
	0xE1: "Production test",
	0xE2: "EVCC setup mode",
	0xE3: "Bus idle",
	0xF1: "Unintended closed contact (Welding)",
	0xF2: "Internal error",
	0xF3: "DC residual current detected",
	0xF4: "Upstream communication timeout",
	0xF5: "Lock of socket failed",
	0xF6: "CS out of range",
	0xF7: "State D requested by EV",
	0xF8: "CP out of range",
	0xF9: "Overcurrent detected",
	0xFA: "Temperature outside limits",
	0xFB: "Unintended opened contact",
}

func init() {
	registry.Add("abl", NewABLeMHFromConfig)
}

// https://www.goingelectric.de/forum/viewtopic.php?p=1550459#p1550459

// NewABLeMHFromConfig creates a ABLeMH charger from generic config
func NewABLeMHFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Timeout         time.Duration
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewABLeMH(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID, cc.Timeout)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateABLeMH -b *ABLeMH -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)"

// NewABLeMH creates ABLeMH charger
func NewABLeMH(uri, device, comset string, baudrate int, slaveID uint8, timeout time.Duration) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.Ascii, slaveID)
	if err != nil {
		return nil, err
	}

	if timeout > 0 {
		conn.Timeout(timeout)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("abl")
	conn.Logger(log.TRACE)

	wb := &ABLeMH{
		conn: conn,
		curr: uint16(6 / 0.06),
	}

	b, err := wb.get(ablRegFirmware, 2)

	// check presence of current sensor
	if err == nil && (b[3]&ablSensorPresent != 0) {
		return decorateABLeMH(wb, wb.currentPower, wb.currents), nil
	}

	return wb, err
}

func (wb *ABLeMH) set(reg, val uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, val)

	// write two times
	_, _ = wb.conn.WriteMultipleRegisters(reg, 1, b)
	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

func (wb *ABLeMH) get(reg, count uint16) ([]byte, error) {
	// read two times
	_, _ = wb.conn.ReadHoldingRegisters(reg, count)
	b, err := wb.conn.ReadHoldingRegisters(reg, count)

	return b, err
}

// Status implements the api.Charger interface
func (wb *ABLeMH) Status() (api.ChargeStatus, error) {
	b, err := wb.get(ablRegStatus, 1)
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
func (wb *ABLeMH) Enabled() (bool, error) {
	b, err := wb.get(ablRegEnabled, 5)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b[6:]) & 0x0FFF
	return u != ablAmpsDisabled, nil
}

// Enable implements the api.Charger interface
func (wb *ABLeMH) Enable(enable bool) error {
	u := ablAmpsDisabled
	if enable {
		u = wb.curr
	}

	err := wb.set(ablRegAmpsConfig, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *ABLeMH) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*ABLeMH)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *ABLeMH) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	// calculate duty cycle according to https://www.goingelectric.de/forum/viewtopic.php?p=1575287#p1575287
	cur := uint16(current / 0.06)

	err := wb.set(ablRegAmpsConfig, cur)
	if err == nil {
		wb.curr = cur
	}

	return err
}

// currentPower implements the api.Meter interface
func (wb *ABLeMH) currentPower() (float64, error) {
	l1, l2, l3, err := wb.currents()
	return 230 * (l1 + l2 + l3), err
}

// Currents implements the api.PhaseCurrents interface
func (wb *ABLeMH) currents() (float64, float64, float64, error) {
	b, err := wb.get(ablRegStatusLong, 5)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		u := binary.BigEndian.Uint16(b[2*(2+i):])
		if u == ablAmpsDisabled || u == 1 {
			u = 0
		}

		res[i] = float64(u) / 10
	}

	return res[2], res[1], res[0], nil
}

var _ api.Diagnosis = (*ABLeMH)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ABLeMH) Diagnose() {
	b, err := wb.get(ablRegFirmware, 2)
	if err == nil {
		fmt.Printf("Firmware: %0 x\n", b)
	}
}

var _ api.Resurrector = (*ABLeMH)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *ABLeMH) WakeUp() error {
	// temporary jump to status E0 (Outlet disabled)
	err := wb.set(ablRegModifyState, 0xE0E0)
	if err == nil {
		time.Sleep(3 * time.Second)
		// jump back to state A1 (Waiting for EV)
		err = wb.set(ablRegModifyState, 0xA1A1)
	}

	return err
}
