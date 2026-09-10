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
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/ablevcc"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// ABLevcc charger implementation for the pre-modbus ABL SURSUM EVCC controller
// https://web.archive.org/web/20160122105853/http://www.abl-sursum.com/global/downloads/bedienungsanleitungen/EVCC.pdf
type ABLevcc struct {
	implement.Caps
	conn    *ablevcc.Connection
	addr    uint8
	curr    int // duty cycle in 0.1% steps
	maxA    float64
	enabled bool
}

const (
	ablEvccCmdFirmware    = 1
	ablEvccCmdStatus      = 2
	ablEvccCmdInputs      = 10
	ablEvccCmdGetPwm      = 11
	ablEvccCmdSetPwm      = 12
	ablEvccCmdGetDefault  = 26
	ablEvccCmdSetBreak    = 27
	ablEvccCmdClearBreak  = 28
	ablEvccCmdGetBreak    = 29
	ablEvccCmdEnterAPrime = 30
	ablEvccCmdLeaveAPrime = 31

	ablEvccPwmMin      = 100 // 10.0% = 6A
	ablEvccPwmMax      = 970 // 97.0% = 82.5A
	ablEvccPwmDisabled = 999 // charging not allowed

	ablEvccMinCurrent = 6
)

// ablEvccStatus maps the device state machine to the charge status. The status
// is mapped directly since api.ChargeStatusString only evaluates the first
// character and would return both a status and an error for the error states.
var ablEvccStatus = map[int]api.ChargeStatus{
	0:  api.StatusA, // A  waiting for EV
	17: api.StatusA, // A' CP off, EV detection disabled
	4:  api.StatusB, // B2 enabled, waiting for charge request
	9:  api.StatusB, // B' charging stopped by EV
	12: api.StatusB, // B1 halted by bBreakCharge (undocumented)
	13: api.StatusB, // B1 EV detected
	5:  api.StatusC, // C  charging
	6:  api.StatusC, // D  charging with ventilation
}

var ablEvccErrors = map[int]string{
	33:  "CS error",
	35:  "EV error",
	37:  "lock error",
	39:  "ventilation error",
	255: "manual mode",
}

func init() {
	registry.AddCtx("abl-evcc", NewABLevccFromConfig)
}

// NewABLevccFromConfig creates an ABL EVCC charger from generic config
func NewABLevccFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		Device  string
		URI     string
		Address uint8
		Timeout time.Duration
	}{
		Address: 1,
		Timeout: ablevcc.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewABLevcc(ctx, cc.Device, cc.URI, cc.Address, cc.Timeout)
}

// NewABLevcc creates an ABL EVCC charger
func NewABLevcc(ctx context.Context, device, uri string, addr uint8, timeout time.Duration) (api.Charger, error) {
	if addr < 1 || addr > 8 {
		return nil, fmt.Errorf("invalid address: %d", addr)
	}

	// keep before connecting- the template test instantiates every template
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("abl-evcc")

	conn, err := ablevcc.Instance(ctx, log, device, uri, timeout)
	if err != nil {
		return nil, err
	}

	wb := &ABLevcc{
		Caps: implement.New(),
		conn: conn,
		addr: addr,
		curr: ablEvccPwmMin,
		maxA: ablEvccCurrent(ablEvccPwmMax),
	}

	// verify device presence
	if _, err := conn.Transact(addr, ablEvccCmdFirmware, ""); err != nil {
		return nil, fmt.Errorf("firmware: %w", err)
	}

	// the available current is limited by the module's default current
	if v, err := wb.get(ablEvccCmdGetDefault); err == nil {
		wb.maxA = ablEvccCurrent(v)
		implement.Has(wb, implement.CurrentLimiter(wb.getMinMaxCurrent))
	}

	if wb.enabled, err = wb.Enabled(); err != nil {
		return nil, err
	}

	return wb, nil
}

// ablEvccCurrent converts a duty cycle in 0.1% steps into the signalled current
func ablEvccCurrent(pwm int) float64 {
	d := float64(pwm) / 10

	if d > 85 {
		return (d - 64) * 2.5
	}

	return d * 0.6
}

// ablEvccPwm converts a current into a duty cycle in 0.1% steps
func ablEvccPwm(current float64) int {
	var pwm int

	switch {
	case current <= 51:
		pwm = int(math.Round(current / 0.06))
	case current < 52.75:
		pwm = 850 // gap between both duty cycle ranges, limit to 51A
	default:
		pwm = int(math.Round((current/2.5 + 64) * 10))
	}

	return min(max(pwm, ablEvccPwmMin), ablEvccPwmMax)
}

func (wb *ABLevcc) get(cmd uint8) (int, error) {
	return wb.conn.Int(wb.addr, cmd, "")
}

func (wb *ABLevcc) setPwm(pwm int) error {
	_, err := wb.conn.Transact(wb.addr, ablEvccCmdSetPwm, fmt.Sprintf("%04d", pwm))
	return err
}

// Status implements the api.Charger interface
func (wb *ABLevcc) Status() (api.ChargeStatus, error) {
	v, err := wb.get(ablEvccCmdStatus)
	if err != nil {
		return api.StatusNone, err
	}

	if status, ok := ablEvccStatus[v]; ok {
		return status, nil
	}

	status, ok := ablEvccErrors[v]
	if !ok {
		status = fmt.Sprintf("%04d", v)
	}

	return api.StatusNone, fmt.Errorf("invalid status: %s", status)
}

// Enabled implements the api.Charger interface
func (wb *ABLevcc) Enabled() (bool, error) {
	v, err := wb.get(ablEvccCmdGetBreak)
	if err != nil {
		return false, err
	}

	wb.enabled = v == 0

	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *ABLevcc) Enable(enable bool) error {
	pwm, cmd := ablEvccPwmDisabled, uint8(ablEvccCmdSetBreak)
	if enable {
		pwm, cmd = wb.curr, ablEvccCmdClearBreak
	}

	// stop an ongoing charge before halting the state machine, apply the
	// current before releasing it
	if err := wb.setPwm(pwm); err != nil {
		return err
	}

	if _, err := wb.conn.Transact(wb.addr, cmd, ""); err != nil {
		return err
	}

	wb.enabled = enable

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *ABLevcc) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*ABLevcc)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *ABLevcc) MaxCurrentMillis(current float64) error {
	if current < ablEvccMinCurrent {
		return fmt.Errorf("invalid current %.1f", current)
	}

	pwm := ablEvccPwm(min(current, wb.maxA))

	// don't lift the charging block while disabled
	if wb.enabled {
		if err := wb.setPwm(pwm); err != nil {
			return err
		}
	}

	wb.curr = pwm

	return nil
}

var _ api.CurrentGetter = (*ABLevcc)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *ABLevcc) GetMaxCurrent() (float64, error) {
	v, err := wb.get(ablEvccCmdGetPwm)
	if err != nil {
		return 0, err
	}

	// outside of the signalling range the module does not offer any current
	if v < ablEvccPwmMin || v > ablEvccPwmMax {
		return 0, nil
	}

	return ablEvccCurrent(v), nil
}

// getMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *ABLevcc) getMinMaxCurrent() (float64, float64, error) {
	return ablEvccMinCurrent, wb.maxA, nil
}

var _ api.Diagnosis = (*ABLevcc)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ABLevcc) Diagnose() {
	if s, err := wb.conn.Transact(wb.addr, ablEvccCmdFirmware, ""); err == nil {
		fmt.Printf("\tFirmware: %s\n", s)
	}
	if v, err := wb.get(ablEvccCmdGetDefault); err == nil {
		fmt.Printf("\tMax. current: %.1fA\n", ablEvccCurrent(v))
	}
	if v, err := wb.get(ablEvccCmdInputs); err == nil {
		fmt.Printf("\tEnable input: %t\n", v&1 != 0)
		fmt.Printf("\tLock input: %t\n", v&2 != 0)
	}
}

var _ api.Resurrector = (*ABLevcc)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *ABLevcc) WakeUp() error {
	// CP off
	if _, err := wb.conn.Transact(wb.addr, ablEvccCmdEnterAPrime, ""); err != nil {
		return err
	}

	time.Sleep(3 * time.Second)

	// CP on
	_, err := wb.conn.Transact(wb.addr, ablEvccCmdLeaveAPrime, "")

	return err
}
