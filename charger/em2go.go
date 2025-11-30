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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://www.em2go.de/download2/ModBus TCP Registers EM2GO  Series.pdf

// Em2Go charger implementation
type Em2Go struct {
	log        *util.Logger
	conn       *modbus.Connection
	current    uint16
	workaround bool
	phases     int
}

const (
	em2GoRegStatus          = 0   // Uint16 RO ENUM
	em2GoRegConnectorState  = 2   // Uint16 RO ENUM
	em2GoRegErrorCode       = 4   // Uint16 RO ENUM
	em2GoRegCurrents        = 6   // Uint16 RO 0.1A
	em2GoRegPower           = 12  // Uint32 RO 1W
	em2GoRegEnergy          = 28  // Uint16 RO 0.1KWh
	em2GoRegMaxCurrent      = 32  // Uint16 RO 0.1A
	em2GoRegMinCurrent      = 34  // Uint16 RO 0.1A
	em2GoRegCableMaxCurrent = 36  // Uint16 RO 0.1A
	em2GoRegSerial          = 38  // Chr[16] RO UTF16
	em2GoRegChargeDuration  = 78  // Uint32 RO 1s
	em2GoRegSafeCurrent     = 87  // Uint16 WR 0.1A
	em2GoRegCommTimeout     = 89  // Uint16 WR 1s
	em2GoRegCurrentLimit    = 91  // Uint16 WR 0.1A
	em2GoRegChargeMode      = 93  // Uint16 WR ENUM
	em2GoRegChargeCommand   = 95  // Uint16 WR ENUM
	em2GoRegVoltages        = 109 // Uint16 RO 0.1V
	em2GoRegPhases          = 200 // Set charging phase 1 unsigned

	//removed due to unreliable session energy when pausing or switching phases
	//em2GoRegChargedEnergy   = 72  // Uint16 RO 0.1kWh
)

func init() {
	registry.AddCtx("em2go", NewEm2GoFromConfig)
	registry.AddCtx("em2go-home", NewEm2GoFromConfig)
}

//go:generate go tool decorate -f decorateEm2Go -b *Em2Go -r api.Charger -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewEm2GoFromConfig creates a Em2Go charger from generic config
func NewEm2GoFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Connector          int // TODO remove deprecated
	}{
		TcpSettings: modbus.TcpSettings{ID: 255},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEm2Go(ctx, cc.URI, cc.ID)
}

// NewEm2Go creates Em2Go charger
func NewEm2Go(ctx context.Context, uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	// Add delay of 60 milliseconds between requests
	conn.Delay(60 * time.Millisecond)

	log := util.NewLogger("em2go")
	conn.Logger(log.TRACE)

	wb := &Em2Go{
		log:     log,
		conn:    conn,
		current: 60,
	}

	return wb.initialize()
}

// initialize performs common initialization for both Em2Go models
func (wb *Em2Go) initialize() (api.Charger, error) {
	var (
		maxCurrent func(float64) error
		phases1p3p func(int) error
		phasesG    func() (int, error)
	)

	// test if workaround is needed (Home fw <1.3)
	if err := wb.maxCurrentMillis(6.1); err != nil {
		return nil, err
	}

	chargerCurrent, err := wb.GetMaxCurrent()
	if err != nil {
		return nil, err
	}

	// did rounding occur?
	if chargerCurrent == 6 {
		wb.workaround = true
	} else {
		maxCurrent = wb.maxCurrentMillis
	}

	if _, err := wb.conn.ReadHoldingRegisters(em2GoRegPhases, 1); err == nil {
		phases1p3p = wb.phases1p3p
		phasesG = wb.getPhases
	}

	return decorateEm2Go(wb, maxCurrent, phases1p3p, phasesG), nil
}

// Status implements the api.Charger interface
func (wb *Em2Go) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 1: //Standby
		return api.StatusA, nil
	case
		2, //Connected
		3, //Starting
		6: //Charging end
		return api.StatusB, nil
	case
		4: //Charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", b[1])
	}
}

// Enabled implements the api.Charger interface
func (wb *Em2Go) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargeCommand, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Em2Go) Enable(enable bool) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, map[bool]uint16{true: 1, false: 2}[enable])

	if _, err := wb.conn.WriteMultipleRegisters(em2GoRegChargeCommand, 1, b); err != nil {
		return err
	}

	// re-set 1p if required
	if wb.workaround && enable {
		if wb.phases == 1 {
			binary.BigEndian.PutUint16(b, uint16(wb.phases))
			if _, err := wb.conn.WriteMultipleRegisters(em2GoRegPhases, 1, b); err != nil {
				return err
			}
		}

		// send default current
		return wb.setCurrent(wb.current)
	}

	return nil
}

func (wb *Em2Go) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(em2GoRegCurrentLimit, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Em2Go) MaxCurrent(current int64) error {
	return wb.maxCurrentMillis(float64(current))
}

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *Em2Go) maxCurrentMillis(current float64) error {
	curr := uint16(current * 10)
	err := wb.setCurrent(curr)
	if err == nil {
		wb.current = curr
	}
	return err
}

var _ api.CurrentGetter = (*Em2Go)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb Em2Go) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, err
}

var _ api.Meter = (*Em2Go)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Em2Go) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b), nil
}

var _ api.MeterEnergy = (*Em2Go)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Em2Go) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b) / 10, nil
}

// getPhaseValues returns 3 register values offset by 2
func (wb *Em2Go) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	var res [3]float64

	for i := range 3 {
		b, err := wb.conn.ReadHoldingRegisters(reg+2*uint16(i), 1)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = float64(binary.BigEndian.Uint16(b)) / 10
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Em2Go)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Em2Go) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(em2GoRegCurrents)
}

var _ api.PhaseVoltages = (*Em2Go)(nil)

// Currents implements the api.PhaseVoltages interface
func (wb *Em2Go) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(em2GoRegVoltages)
}

var _ api.ChargeTimer = (*Em2Go)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Em2Go) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargeDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Em2Go) phases1p3p(phases int) error {
	if wb.workaround {
		// when enabled, disable, wait 10 seconds, enable and set phases
		enabled, err := wb.Enabled()
		if err != nil {
			return err
		}

		if enabled {
			if err := wb.Enable(false); err != nil {
				return err
			}

			time.Sleep(10 * time.Second)

			if err := wb.Enable(true); err != nil {
				return err
			}
		}
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(phases))

	_, err := wb.conn.WriteMultipleRegisters(em2GoRegPhases, 1, b)
	if err == nil {
		wb.phases = phases
	}

	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *Em2Go) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoRegPhases, 1)
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint16(b)), nil
}

var _ api.Diagnosis = (*Em2Go)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Em2Go) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegStatus, 1); err == nil {
		fmt.Printf("\tCharging Station Status:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegErrorCode, 1); err == nil {
		fmt.Printf("\tError Code:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegMinCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Min. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegCableMaxCurrent, 1); err == nil {
		fmt.Printf("\tCable Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	var serial []byte
	for reg := range 8 {
		b, err := wb.conn.ReadHoldingRegisters(em2GoRegSerial+2*uint16(reg), 2)
		if err != nil {
			return
		}
		serial = append(serial, b...)
	}
	fmt.Printf("\tSerial: %s\n", string(serial))
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegCurrentLimit, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargeMode, 1); err == nil {
		fmt.Printf("\tCharge Mode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoRegChargeCommand, 1); err == nil {
		fmt.Printf("\tCharge Command:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
