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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// SigenergyEVDC is the Sigenergy EVDC DC charger module attached to a
// SigenStor/Sigen Hybrid inverter. All registers live in the hybrid
// inverter's Modbus slave address space; the EVDC has no own slave ID.
type SigenergyEVDC struct {
	conn       *modbus.Connection
	enabled    bool
	ratedPower uint32 // W
	inputG     func() ([]byte, error)
}

const (
	// input registers (FC04), all on the hybrid inverter's slave ID
	evdcBase                    = 31500 // start of the bulk-read block
	evdcRegVehicleVoltage       = 31500 // U16, 0.1 V
	evdcRegChargingCurrent      = 31501 // U16, 0.1 A
	evdcRegOutputPower          = 31502 // S32, W (assumed negative while discharging)
	evdcRegVehicleSoc           = 31504 // U16, 0.1 %
	evdcRegSessionEnergy        = 31505 // U32, 10 Wh, single session
	evdcRegSessionDuration      = 31507 // U32, s, single session
	evdcRegRunningState         = 31513 // U16
	evdcRegDischargeCurrent     = 31514 // U16, 0.1 A
	evdcRegTotalEnergy          = 31519 // U32, 10 Wh
	evdcRegTotalDischargeEnergy = 31521 // U32, 10 Wh
	evdcRegRatedPower           = 31523 // U32, W
	evdcRegRatedDischargePower  = 31525 // U32, W
	// 31515/31517 (session discharge energy/duration, U32) exist but are unused
	// 31509/31511 inside the bulk-read block are inverter PV registers — not EVDC data

	// holding registers
	evdcRegStartStop      = 41000 // U16, WO, single FC06 write only (41001 is reserved): 0 = start, 1 = stop
	evdcRegPowerLimit     = 41002 // U32, W, RW, FC10; clamped to [evdcMinPower, rated] — never write 0!
	evdcRegDischargeLimit = 41004 // U32, W, RW; discharge control is out of scope

	evdcInputLen = 23 // registers 31500-31522 in a single FC04 read

	// minimum setpoint
	evdcMinPower = 500 // 1A@500V per DC CCS Power Classes

	evdcMinCurrent = 1.0 // A

	// evcc current setpoints are converted using the 3-phase AC convention
	evdcPowerPerAmp = 230 * 3 // W/A
)

// running states of evdcRegRunningState
const (
	evdcStateIdle        = 0x00
	evdcStateOccupied    = 0x01 // gun plugged in but not detected
	evdcStatePreparing   = 0x02 // establishing communication
	evdcStateCharging    = 0x03
	evdcStateFault       = 0x04
	evdcStateScheduled   = 0x05
	evdcStateEnded       = 0x06
	evdcStateUnavailable = 0x07 // under maintenance
	evdcStateDischarging = 0x08
	evdcStateAlarm       = 0x09
	evdcStateInsulation  = 0x0A // insulation detection in progress
)

var evdcStateNames = map[uint16]string{
	evdcStateIdle:        "Idle",
	evdcStateOccupied:    "Occupied",
	evdcStatePreparing:   "Preparing",
	evdcStateCharging:    "Charging",
	evdcStateFault:       "Fault",
	evdcStateScheduled:   "Scheduled",
	evdcStateEnded:       "Ended",
	evdcStateUnavailable: "Unavailable",
	evdcStateDischarging: "Discharging",
	evdcStateAlarm:       "Alarm",
	evdcStateInsulation:  "Insulation detection",
}

func init() {
	registry.AddCtx("sigenergy-evdc", NewSigenergyEVDCFromConfig)
}

// NewSigenergyEVDCFromConfig creates a new Sigenergy EVDC ModbusTCP charger
func NewSigenergyEVDCFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSigenergyEVDC(ctx, cc.URI, cc.ID)
}

// NewSigenergyEVDC creates a new charger
func NewSigenergyEVDC(ctx context.Context, uri string, slaveID uint8) (*SigenergyEVDC, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("sigenergy-evdc")
	conn.Logger(log.TRACE)

	wb := newSigenergyEVDC(conn)

	// probe rated power: verifies DC charger presence and V2.9 firmware, value clamps setpoints
	b, err := conn.ReadInputRegisters(evdcRegRatedPower, 2)
	if err != nil {
		return nil, fmt.Errorf("no DC charger present or firmware too old (requires Modbus protocol V2.9): %w", err)
	}
	wb.ratedPower = encoding.Uint32(b)

	// seed enabled state so evcc restarts mid-session report the true state
	b, err = conn.ReadInputRegisters(evdcRegRunningState, 1)
	if err != nil {
		return nil, err
	}
	wb.enabled = encoding.Uint16(b) == evdcStateCharging

	return wb, nil
}

// newSigenergyEVDC wires the struct without sponsor gate and device probe (also used by tests)
func newSigenergyEVDC(conn *modbus.Connection) *SigenergyEVDC {
	wb := &SigenergyEVDC{
		conn: conn,
	}

	// all cyclic values come from a single cached bulk read to respect the
	// device's documented 1000ms minimum request period
	wb.inputG = util.Cached(func() ([]byte, error) {
		return wb.conn.ReadInputRegisters(evdcBase, evdcInputLen)
	}, time.Second)

	return wb
}

// evdcInput returns the bytes of the given register within the cached bulk read
func evdcInput(b []byte, reg uint16, n int) []byte {
	off := 2 * int(reg-evdcBase)
	return b[off : off+n]
}

// Status implements the api.Charger interface
func (wb *SigenergyEVDC) Status() (api.ChargeStatus, error) {
	b, err := wb.inputG()
	if err != nil {
		return api.StatusNone, err
	}

	switch state := encoding.Uint16(evdcInput(b, evdcRegRunningState, 2)); state {
	case evdcStateIdle:
		return api.StatusA, nil
	case evdcStateOccupied, evdcStatePreparing, evdcStateScheduled, evdcStateEnded, evdcStateInsulation:
		return api.StatusB, nil
	case evdcStateCharging:
		wb.enabled = true
		return api.StatusC, nil
	case evdcStateDischarging:
		return api.StatusC, nil
	case evdcStateFault, evdcStateUnavailable, evdcStateAlarm:
		return api.StatusNone, fmt.Errorf("device state: %s", evdcStateNames[state])
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (wb *SigenergyEVDC) Enabled() (bool, error) {
	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *SigenergyEVDC) Enable(enable bool) error {
	var v uint16
	if !enable {
		v = 1
	}

	// single FC06 write only — 41001 is a reserved register
	_, err := wb.conn.WriteSingleRegister(evdcRegStartStop, v)
	if err == nil {
		wb.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *SigenergyEVDC) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SigenergyEVDC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *SigenergyEVDC) MaxCurrentMillis(current float64) error {
	if current < evdcMinCurrent {
		return fmt.Errorf("invalid current %.3g", current)
	}

	power := min(max(uint32(current*evdcPowerPerAmp), evdcMinPower), wb.ratedPower)

	b := make([]byte, 4)
	encoding.PutUint32(b, power)

	_, err := wb.conn.WriteMultipleRegisters(evdcRegPowerLimit, 2, b)
	return err
}

var _ api.CurrentLimiter = (*SigenergyEVDC)(nil)

// GetMinMaxCurrent implements the api.CurrentLimiter interface
func (wb *SigenergyEVDC) GetMinMaxCurrent() (float64, float64, error) {
	return evdcMinCurrent, float64(wb.ratedPower) / evdcPowerPerAmp, nil
}

var _ api.Meter = (*SigenergyEVDC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SigenergyEVDC) CurrentPower() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	// S32, negative while discharging (to be confirmed on hardware, see spec)
	return float64(encoding.Int32(evdcInput(b, evdcRegOutputPower, 4))), nil
}

var _ api.MeterEnergy = (*SigenergyEVDC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SigenergyEVDC) TotalEnergy() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(evdcInput(b, evdcRegTotalEnergy, 4))) / 100, nil
}

var _ api.MeterReturnEnergy = (*SigenergyEVDC)(nil)

// ReturnEnergy implements the api.MeterReturnEnergy interface
func (wb *SigenergyEVDC) ReturnEnergy() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(evdcInput(b, evdcRegTotalDischargeEnergy, 4))) / 100, nil
}

var _ api.ChargeRater = (*SigenergyEVDC)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *SigenergyEVDC) ChargedEnergy() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(evdcInput(b, evdcRegSessionEnergy, 4))) / 100, nil
}

var _ api.ChargeTimer = (*SigenergyEVDC)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *SigenergyEVDC) ChargeDuration() (time.Duration, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint32(evdcInput(b, evdcRegSessionDuration, 4))) * time.Second, nil
}

var _ api.Battery = (*SigenergyEVDC)(nil)

// Soc implements the api.Battery interface
func (wb *SigenergyEVDC) Soc() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	if soc := float64(encoding.Uint16(evdcInput(b, evdcRegVehicleSoc, 2))) / 10; soc > 0 {
		return soc, nil
	}

	// no vehicle connected or no BMS data
	return 0, api.ErrNotAvailable
}

var _ api.Diagnosis = (*SigenergyEVDC)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SigenergyEVDC) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(evdcRegRunningState, 1); err == nil {
		state := encoding.Uint16(b)
		name := evdcStateNames[state]
		if name == "" {
			name = "Unknown"
		}
		fmt.Printf("\tRunning state:\t%d (%s)\n", state, name)
	}
	if b, err := wb.conn.ReadInputRegisters(evdcRegVehicleVoltage, 1); err == nil {
		fmt.Printf("\tVehicle voltage:\t%.1f V\n", float64(encoding.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadInputRegisters(evdcRegChargingCurrent, 1); err == nil {
		fmt.Printf("\tCharging current:\t%.1f A\n", float64(encoding.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadInputRegisters(evdcRegDischargeCurrent, 1); err == nil {
		fmt.Printf("\tDischarging current:\t%.1f A\n", float64(encoding.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadInputRegisters(evdcRegTotalDischargeEnergy, 2); err == nil {
		fmt.Printf("\tTotal discharged:\t%.2f kWh\n", float64(encoding.Uint32(b))/100)
	}
	fmt.Printf("\tRated power:\t%d W\n", wb.ratedPower)
	if b, err := wb.conn.ReadInputRegisters(evdcRegRatedDischargePower, 2); err == nil {
		fmt.Printf("\tRated discharge power:\t%d W\n", encoding.Uint32(b))
	}
	// power limit readback: FC04 per spec text, FC03 field-proven fallback
	if b, err := wb.conn.ReadInputRegisters(evdcRegPowerLimit, 2); err == nil {
		fmt.Printf("\tPower limit:\t%d W\n", encoding.Uint32(b))
	} else if b, err := wb.conn.ReadHoldingRegisters(evdcRegPowerLimit, 2); err == nil {
		fmt.Printf("\tPower limit (FC03):\t%d W\n", encoding.Uint32(b))
	}
	if b, err := wb.conn.ReadInputRegisters(evdcRegDischargeLimit, 2); err == nil {
		fmt.Printf("\tDischarge limit:\t%d W\n", encoding.Uint32(b))
	}
}

var _ api.Resurrector = (*SigenergyEVDC)(nil)

// WakeUp implements the api.Resurrector interface
func (wb *SigenergyEVDC) WakeUp() error {
	// re-issue Start: recovers sessions stopped on the vehicle/vendor-app side,
	// which latch until a new start command (device-verified)
	_, err := wb.conn.WriteSingleRegister(evdcRegStartStop, 0)
	return err
}
