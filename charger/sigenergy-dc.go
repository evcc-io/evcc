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
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// SigenergyDC charger implementation for EVDC series.
// It communicates via Modbus TCP at the inverter address.
//
// Limitations:
//   - no current control (DC charging is managed by the inverter)
//   - no phase switching
//   - only start/stop control
type SigenergyDC struct {
	log     *util.Logger
	conn    *modbus.Connection
	enabled bool
}

const (
	// Sigenergy DC charger Modbus registers (protocol V2.5)
	regSigDcAlarm           = 30609 // U16, alarm code
	regSigDcVehicleVoltage  = 31500 // U16, Gain 10, Unit V
	regSigDcChargingCurrent = 31501 // U16, Gain 10, Unit A
	regSigDcOutputPower     = 31502 // S32, Gain 1000, Unit kW
	regSigDcVehicleSoc      = 31504 // U16, Gain 10, Unit %
	regSigDcCurrentCapacity = 31505 // U32, Gain 100, Unit kWh
	regSigDcCurrentDuration = 31507 // U32, Gain 1, Unit s
	regSigDcRunningState    = 31513 // U16, Gain 1, running state (Modbus v2.8+)
	regSigDcStartStop       = 41000 // U16, WO, 0: Start, 1: Stop

	// DC charger running states
	sigDcStateIdle        = 0x00
	sigDcStateOccupied    = 0x01
	sigDcStatePrepComm    = 0x02
	sigDcStateCharging    = 0x03
	sigDcStateFault       = 0x04
	sigDcStateScheduled   = 0x05
	sigDcStateEnded       = 0x06
	sigDcStateUnavail     = 0x07
	sigDcStateDischarging = 0x08
	sigDcStateAlarm       = 0x09
	sigDcStatePrepInsul   = 0x0A
)

func init() {
	registry.AddCtx("sigenergy-dc", NewSigenergyDcFromConfig)
}

// NewSigenergyDcFromConfig creates a new Sigenergy DC ModbusTCP charger
func NewSigenergyDcFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSigenergyDC(ctx, cc.URI, cc.ID)
}

// NewSigenergyDC creates a new Sigenergy DC charger
func NewSigenergyDC(ctx context.Context, uri string, slaveID uint8) (*SigenergyDC, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("sigenergy-dc")
	conn.Logger(log.TRACE)

	wb := &SigenergyDC{
		log:  log,
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *SigenergyDC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigDcRunningState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch state := binary.BigEndian.Uint16(b); state {
	case sigDcStateIdle, sigDcStateUnavail:
		// no vehicle connected
		wb.enabled = false
		return api.StatusA, nil

	case sigDcStateCharging, sigDcStateDischarging:
		// actively charging/discharging
		wb.enabled = true
		return api.StatusC, nil

	case sigDcStateOccupied, sigDcStatePrepComm, sigDcStatePrepInsul,
		sigDcStateScheduled, sigDcStateEnded:
		// vehicle connected but not actively charging
		return api.StatusB, nil

	case sigDcStateFault, sigDcStateAlarm:
		wb.log.WARN.Printf("charger fault/alarm (state: %d)", state)
		wb.enabled = false
		return api.StatusB, nil

	default:
		return api.StatusNone, fmt.Errorf("unknown running state: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (wb *SigenergyDC) Enabled() (bool, error) {
	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *SigenergyDC) Enable(enable bool) error {
	var s uint16
	if !enable {
		s = 1
	}

	_, err := wb.conn.WriteSingleRegister(regSigDcStartStop, s)
	if err == nil {
		wb.enabled = enable
	}
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *SigenergyDC) MaxCurrent(current int64) error {
	// DC charging is managed by the inverter and cannot be current-throttled
	return api.ErrNotAvailable
}

var _ api.Meter = (*SigenergyDC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SigenergyDC) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigDcOutputPower, 2)
	if err != nil {
		return 0, err
	}

	// S32 register with Gain 1000 and Unit kW.
	// A value of 1000 represents 1 kW = 1000 W.
	// The raw value therefore equals Watts directly.
	return float64(int32(binary.BigEndian.Uint32(b))), nil
}

var _ api.MeterEnergy = (*SigenergyDC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SigenergyDC) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigDcCurrentCapacity, 2)
	if err != nil {
		return 0, err
	}

	// U32 register with Gain 100, divide by 100 to get kWh
	return float64(binary.BigEndian.Uint32(b)) / 100, nil
}

var _ api.Battery = (*SigenergyDC)(nil)

// Soc implements the api.Battery interface
func (wb *SigenergyDC) Soc() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigDcVehicleSoc, 1)
	if err != nil {
		return 0, err
	}

	// U16 register with Gain 10, divide by 10 to get %
	return float64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ api.Diagnosis = (*SigenergyDC)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SigenergyDC) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(regSigDcRunningState, 1); err == nil {
		stateNames := map[uint16]string{
			sigDcStateIdle: "Idle", sigDcStateOccupied: "Occupied",
			sigDcStatePrepComm: "Preparing Comm", sigDcStateCharging: "Charging",
			sigDcStateFault: "Fault", sigDcStateScheduled: "Scheduled",
			sigDcStateEnded: "Ended", sigDcStateUnavail: "Unavailable",
			sigDcStateDischarging: "Discharging", sigDcStateAlarm: "Alarm",
			sigDcStatePrepInsul: "Preparing Insulation",
		}
		state := binary.BigEndian.Uint16(b)
		name := stateNames[state]
		if name == "" {
			name = "Unknown"
		}
		fmt.Printf("\tRunning State:\t%d (%s)\n", state, name)
	}
	if b, err := wb.conn.ReadHoldingRegisters(regSigDcAlarm, 1); err == nil {
		fmt.Printf("\tDC Alarm:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(regSigDcVehicleVoltage, 1); err == nil {
		fmt.Printf("\tVehicle Voltage:\t%.1f V\n", float64(binary.BigEndian.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadHoldingRegisters(regSigDcChargingCurrent, 1); err == nil {
		fmt.Printf("\tCharging Current:\t%.1f A\n", float64(binary.BigEndian.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadHoldingRegisters(regSigDcVehicleSoc, 1); err == nil {
		fmt.Printf("\tVehicle SOC:\t%.1f %%\n", float64(binary.BigEndian.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadHoldingRegisters(regSigDcCurrentDuration, 2); err == nil {
		fmt.Printf("\tCurrent Duration:\t%d s\n", binary.BigEndian.Uint32(b))
	}
}
