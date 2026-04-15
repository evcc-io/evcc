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
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Amperfied charger implementation
type Amperfied struct {
	log            *util.Logger
	conn           *modbus.Connection
	mu             sync.Mutex
	current        uint16
	phases         int
	wakeup         bool
	phaseSwitchEnd time.Time
}

const (
	ampRegChargingState          = 5    // Input   (uint16)
	ampRegCurrents               = 6    // Input   (uint16*3)
	ampRegVoltages               = 10   // Input   (uint16*3)
	ampRegPower                  = 14   // Input   (uint16)
	ampRegEnergy                 = 17   // Input   (uint32)
	ampRegTimeoutConfig          = 257  // Holding (uint16)
	ampRegRemoteLock             = 259  // Holding (uint16)
	ampRegAmpsConfig             = 261  // Holding (uint16)
	ampRegFailSafeConfig         = 262  // Holding (uint16)
	ampRegPhaseSwitchControl     = 501  // Holding (uint16)
	ampRegPhaseSwitchDuration    = 503  // Holding (uint16)
	ampRegPhaseSwitchWaitingTime = 504  // Holding (uint16)
	ampRegPhaseSwitchState       = 5001 // Input   (uint16)
	ampRegRfidUID                = 2002 // Input   (uint16*6)
)

func init() {
	registry.AddCtx("amperfied", NewAmperfiedFromConfig)
}

//go:generate go tool decorate -f decorateAmperfied -b *Amperfied -r api.Charger -t api.PhaseSwitcher,api.PhaseGetter

// NewAmperfiedFromConfig creates a Amperfied charger from generic config
func NewAmperfiedFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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

	return NewAmperfied(ctx, cc.URI, cc.ID, cc.Phases1p3p)
}

// NewAmperfied creates Amperfied charger
func NewAmperfied(ctx context.Context, uri string, slaveID uint8, phases bool) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
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
		go wb.heartbeat(ctx, time.Duration(u)*time.Millisecond/2)
	}

	// disable phase-switch waiting time
	if err := wb.set(ampRegPhaseSwitchWaitingTime, 0); err != nil {
		return nil, fmt.Errorf("phase-switch waiting time: %w", err)
	}

	var phases1p3p func(int) error
	var phasesG func() (int, error)
	if phases {
		phases1p3p = wb.phases1p3p
		phasesG = wb.getPhases
	}

	return decorateAmperfied(wb, phases1p3p, phasesG), nil
}

func (wb *Amperfied) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

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

		fallthrough
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
		wb.mu.Lock()
		wb.current = cur
		wb.mu.Unlock()
	}

	return enabled, nil
}

// Enable implements the api.Charger interface
func (wb *Amperfied) Enable(enable bool) error {
	var cur uint16
	if enable {
		wb.mu.Lock()
		cur = wb.current
		wb.mu.Unlock()
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

	curr := uint16(10 * current)

	// skip write while phase-switch is in progress; re-apply happens via timer
	wb.mu.Lock()
	inSwitch := time.Now().Before(wb.phaseSwitchEnd)
	if inSwitch {
		wb.current = curr
	}
	wb.mu.Unlock()

	if inSwitch {
		return nil
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, curr)

	if _, err := wb.conn.WriteMultipleRegisters(ampRegAmpsConfig, 1, b); err != nil {
		return err
	}

	wb.mu.Lock()
	wb.current = curr
	wb.mu.Unlock()

	return nil
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

// Identify implements the api.Identifier interface
func (wb *Amperfied) Identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(ampRegRfidUID, 6)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
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

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Amperfied) phases1p3p(phases int) error {
	// reduce current to minimum before phase-switching
	if err := wb.set(ampRegAmpsConfig, 60); err != nil {
		return err
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(phases))

	// Set new phases
	_, err := wb.conn.WriteMultipleRegisters(ampRegPhaseSwitchControl, 1, b)
	if err != nil {
		return err
	}
	wb.phases = phases

	// read phase-switch duration during which no current changes are accepted
	d, err := wb.conn.ReadHoldingRegisters(ampRegPhaseSwitchDuration, 1)
	if err != nil {
		return err
	}

	// block current writes until phase-switch completes
	duration := time.Duration(binary.BigEndian.Uint16(d)) * time.Second
	wb.mu.Lock()
	wb.phaseSwitchEnd = time.Now().Add(duration)
	wb.mu.Unlock()

	// re-apply current after phase-switch completes to honor interim changes
	time.AfterFunc(duration, func() {
		wb.mu.Lock()
		cur := wb.current
		wb.mu.Unlock()

		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, cur)
		if _, err := wb.conn.WriteMultipleRegisters(ampRegAmpsConfig, 1, b); err != nil {
			wb.log.ERROR.Printf("re-apply current after phase switch: %v", err)
		}
	})

	return nil
}

// getPhases implements the api.PhaseGetter interface
func (wb *Amperfied) getPhases() (int, error) {
	b, err := wb.conn.ReadInputRegisters(ampRegPhaseSwitchState, 1)
	if err != nil {
		wb.log.ERROR.Println("Error reading ampRegPhaseSwitchState")
		return 0, err
	}

	phases := int(binary.BigEndian.Uint16(b))
	if phases == 0 {
		wb.log.DEBUG.Println("phase-switching in progress")
		return wb.phases, nil
	}

	return phases, nil
}

var _ api.Diagnosis = (*Amperfied)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Amperfied) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(ampRegTimeoutConfig, 1); err == nil {
		fmt.Printf("Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegRemoteLock, 1); err == nil {
		fmt.Printf("Remote Lock:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegFailSafeConfig, 1); err == nil {
		fmt.Printf("FailSafe:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegAmpsConfig, 1); err == nil {
		fmt.Printf("Amps Config:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegPhaseSwitchDuration, 1); err == nil {
		fmt.Printf("Phase Switch Duration:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(ampRegPhaseSwitchWaitingTime, 1); err == nil {
		fmt.Printf("Phase Switch Waiting Time:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(ampRegPhaseSwitchState, 1); err == nil {
		fmt.Printf("Phase Switch State:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
