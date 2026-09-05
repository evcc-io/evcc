package charger

/*
This file is distributed under the MIT license
Copyright (c) 2026 marvinfortytwo

This file provides charger support for pre-modbus ABL EVCC controllers
found for example in the "E.ON Ladebox Basis" wallbox. See:
https://web.archive.org/web/20160122105853/http://www.abl-sursum.com/global/downloads/bedienungsanleitungen/EVCC.pdf
Tested with firmware V2.3 and EVSE205/Renault ZOE Q210
*/

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/grid-x/serial"
)

// ABLv1 EVCC command codes
const (
	cmdGetFirmware         = 1
	cmdGetStateMachine     = 2
	cmdGetChargeCurrent    = 11
	cmdSetMaxCurrent       = 12
	cmdStopCharging        = 25
	cmdSetBreak            = 27
	cmdClearBreak          = 28
	cmdGetBreakStatus      = 29
	cmdSwitchToAprime      = 30
	cmdSwitchFromAprimeToA = 31
)

// charger states
const (
	StateA       = 0
	StateB2      = 4
	StateC       = 5
	StateD       = 6
	StateBPrime  = 9
	StateB1Break = 12
	StateB1      = 13
	StateAPrime  = 17
	StateECS     = 33
	StateEEV     = 35
	StateELock   = 37
	StateEVent   = 39
	StateManual  = 255
)

var ablStateNames = map[int]string{
	StateA:       "A Idle",
	StateB2:      "B Waiting for EV",
	StateC:       "C Charging",
	StateD:       "C Charging with Ventilation",
	StateBPrime:  "B' Charging stopped by EV",
	StateB1Break: "B Break Charing stopped by EVCC",
	StateB1:      "B Connected",
	StateAPrime:  "A' Idle CP off",
	StateECS:     "E Error CS",
	StateEEV:     "E Error EV",
	StateELock:   "E Error Lock",
	StateEVent:   "E Error Ventilation",
	StateManual:  "F Manual",
}

// charger blocks when this value is given as charge current
const blockCharge = 999

// ABLv1 implements api.Charger for the ABL SURSUM EVCC ASCII protocol.
type ABLv1 struct {
	tr              sync.Mutex         // lock for transfers
	conn            io.ReadWriteCloser // serial connection
	r               *bufio.Reader      // buffered read for serial connection
	addr            uint8              // device address 0..9, configurable
	rxTimeout       time.Duration      // read timeout for replies (transaction timeout)
	log             *util.Logger       // logger
	allowedCurrents []int64            // supported discrete currents
	curr            int64              // requested current
}

func init() {
	registry.AddCtx("ablv1", NewABLv1FromConfig)
}

// NewABLv1FromConfig creates an ABLv1 from config map (used by registry.AddCtx).
// Expected keys in `other`:
// - device (string) e.g. "/dev/ttyUSB0"
// - id (int) device address 0..9
// - timeout (int) milliseconds (optional, default 2000)
func NewABLv1FromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	var cfg struct {
		ID      uint8
		Device  string
		Timeout int `yaml:"timeout,omitempty" json:",omitempty"`
	}

	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, err
	}

	// use default timeout if not specified
	if cfg.Timeout == 0 {
		cfg.Timeout = 2000
	}

	// ABLv1 has fix communication parameters
	cc := serial.Config{
		BaudRate: 38400,
		Timeout:  time.Duration(cfg.Timeout) * time.Millisecond,
		DataBits: 8,
		Parity:   "N",
		StopBits: 1,
		Address:  cfg.Device,
	}

	return NewABLv1(ctx, cc, cfg.ID)
}

// NewABLv1 creates ABLv1 charger
func NewABLv1(ctx context.Context, settings serial.Config, id uint8) (api.Charger, error) {
	conn, err := serial.Open(&settings)
	if err != nil {
		return nil, fmt.Errorf("error during serial open: %w", err)
	}

	log := util.NewLogger("ABLv1")

	wb := &ABLv1{
		conn:            conn,
		r:               bufio.NewReader(conn),
		log:             log,
		curr:            6,
		addr:            id,
		allowedCurrents: []int64{6, 10, 13, 16, 20, 30, 32, 63, 70, 80},
		rxTimeout:       settings.Timeout,
	}

	log.TRACE.Println("Address: ", settings.Address)
	log.TRACE.Println("Id: ", id)

	return wb, nil
}

// Status impements the api.Charger interface
// Status implements api.Charger by querying cmd 02 (get state machine status).
func (wb *ABLv1) Status() (api.ChargeStatus, error) {
	stateCode, err := wb.transact(context.Background(), cmdGetStateMachine, "")

	wb.log.TRACE.Println("status call returned: ", stateCode)

	if err != nil {
		return api.StatusNone, fmt.Errorf("error during transaction %d: %w", cmdGetStateMachine, err)
	}

	state, ok := ablStateNames[stateCode]
	if !ok {
		return api.StatusNone, fmt.Errorf("unknown status code %04d", stateCode)
	}

	return api.ChargeStatusString(state)
}

// Enabled implements the api.Charger interface
// The device doesn't provide an explicit enabled flag, so check if break or current block
// is active
func (wb *ABLv1) Enabled() (bool, error) {
	var err error
	var hasBreak, current int

	wb.log.TRACE.Println("enabled? called")

	hasBreak, err = wb.transact(context.Background(), cmdGetBreakStatus, "")
	if err != nil {
		return false, fmt.Errorf("error during transaction %d: %w", cmdGetBreakStatus, err)
	}

	current, err = wb.transact(context.Background(), cmdGetChargeCurrent, "")
	if err != nil {
		return false, fmt.Errorf("error during transaction %d: %w", cmdGetChargeCurrent, err)
	}

	if hasBreak == 0 {
		wb.log.TRACE.Println("break is disabled", hasBreak)
	} else {
		wb.log.TRACE.Println("break is enabled !", hasBreak)
	}

	if current == blockCharge {
		wb.log.TRACE.Println("charging is blocked !", current)
	} else {
		wb.log.TRACE.Println("charging is unblocked", current)
	}

	return (hasBreak == 0) && (current != blockCharge), nil
}

// find the current (or the next smaller one) in the allowedCurrents array
// returns the found current in mA and index
func (wb *ABLv1) GetNearestCurrent(current int64) (int64, int) {
	// do a search for the next larger current
	i := sort.Search(len(wb.allowedCurrents), func(i int) bool {
		return wb.allowedCurrents[i] > current
	})

	// use the index before the hit if it wasn't the lowest already
	if i > 0 {
		i--
	}

	wb.log.TRACE.Println("found nearest current", wb.allowedCurrents[i])

	return wb.allowedCurrents[i], i
}

// Get current charge current
// returns current in mA and forward error
func (wb *ABLv1) GetCurrent() (current int64, err error) {
	pwm, err := wb.transact(context.Background(), cmdGetChargeCurrent, "")
	if err != nil {
		return 0, fmt.Errorf("error during transaction %d: %w", cmdGetChargeCurrent, err)
	}

	// check if charging is blocked
	if pwm == blockCharge {
		return blockCharge, nil
	}

	// fixme: only up to 32 A!
	return int64(pwm * 6 / 100), nil
}

// Get break status
func (wb *ABLv1) GetBreakState() (breakState bool, err error) {
	bs, err := wb.transact(context.Background(), cmdGetBreakStatus, "")
	if err != nil {
		return false, fmt.Errorf("error during transaction %d: %w", cmdGetBreakStatus, err)
	}

	return bs != 0, nil
}

// SetCurrent finds the nearest valid current from the current table and set it
func (wb *ABLv1) SetCurrent(current int64) error {
	new_current, idx := wb.GetNearestCurrent(current)

	payload := fmt.Sprintf("%04d", idx)
	wb.log.TRACE.Printf("setcurrent called with value %d, using %d with payload %s",
		current, new_current, payload)

	_, err := wb.transact(context.Background(), cmdSetMaxCurrent, payload)
	if err != nil {
		return fmt.Errorf("error during transaction %d with payload %s: %w", cmdSetMaxCurrent, payload, err)
	}

	return nil
}

// Enable implements api.Charger interface
// Charger can be blocked by either "break" in state A or by current (=999) in state C
func (wb *ABLv1) Enable(enable bool) error {
	var breakState bool
	var cur int64

	// get charging state
	state, err := wb.Status()
	if err != nil {
		return fmt.Errorf("error during get_status: %w", err)
	}

	// get charge current
	cur, err = wb.GetCurrent()
	if err != nil {
		return fmt.Errorf("error during GetCurrent: %w", err)
	}

	// get break status
	breakState, err = wb.GetBreakState()
	if err != nil {
		return fmt.Errorf("error during GetBreakState: %w", err)
	}

	wb.log.TRACE.Printf("enable called with value %t and in state %s (current: %d, break %t)", enable, state, cur, breakState)

	if enable {
		if breakState {
			// blocked via break
			_, err = wb.transact(context.Background(), cmdClearBreak, "")
			if err != nil {
				return fmt.Errorf("error during transaction %d: %w", cmdClearBreak, err)
			}
		}

		if state != api.StatusA {
			// set current only works in B or C, otherwise defer setcurrent to wake
			err = wb.SetCurrent(wb.curr)
			if err != nil {
				return fmt.Errorf("error during SetCurrent with %d: %w", wb.curr, err)
			}
		}
	} else {
		// in state C, break doesn't work, so disable charging via current while charging
		if state == api.StatusC {
			payload := fmt.Sprintf("%04d", blockCharge)
			_, err = wb.transact(context.Background(), cmdSetMaxCurrent, payload)
		} else {
			// disable charging via break in A or B state
			_, err = wb.transact(context.Background(), cmdSetBreak, "")
		}
	}

	if err != nil {
		return fmt.Errorf("error during enable: %w", err)
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
// It maps the requested current to the highest supported step <= requested value using
// the internal SetCurrent method
func (wb *ABLv1) MaxCurrent(current int64) error {
	var err error

	wb.log.TRACE.Println("maxcurrent called with value", current)

	state, err := wb.Status()
	if err != nil {
		return err
	}

	wb.curr = current

	if state == api.StatusC {
		err = wb.SetCurrent(current)
	}

	return err
}

// GetMinMaxCurrent implements api.CurrentLimiter interface
// return the min and max currents from current table
func (wb *ABLv1) GetMinMaxCurrent() (float64, float64, error) {
	min := float64(wb.allowedCurrents[0])
	max := float64(wb.allowedCurrents[len(wb.allowedCurrents)-1])

	wb.log.TRACE.Println("GetMinMaxcurrent called, returned ", min, max)

	return min, max, nil
}

var _ api.Resurrector = (*ABLv1)(nil)

// WakeUp implements the api.Resurrector interface
// by toggle CP, vehicle will be waked
func (wb *ABLv1) WakeUp() error {
	wb.log.TRACE.Println("Wakeup called")

	// CP off (A')
	_, err := wb.transact(context.Background(), cmdSwitchToAprime, "")

	if err != nil {
		return fmt.Errorf("wake: error switching to A': %w", err)
	}

	// sleep for 3 secs
	time.Sleep(3 * time.Second)

	// CP on (A)
	_, err = wb.transact(context.Background(), cmdSwitchFromAprimeToA, "")

	if err != nil {
		return fmt.Errorf("wake: error switching to from A' to A: %w", err)
	}

	time.Sleep(5 * time.Second)

	// set current in state B or C
	err = wb.SetCurrent(wb.curr)
	if err != nil {
		return fmt.Errorf("wake: error during SetCurrent with %d: %w", wb.curr, err)
	}

	return err
}
