package charger

// LICENSE

// Copyright (c) 2019-2024 andig

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
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://www.em2go.de/download2/ModBus TCP Registers EM2GO Home Series.pdf

// Em2Go charger implementation
type Em2GoHome struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	em2GoHomeRegStatus          = 0   // Uint16 RO ENUM
	em2GoHomeRegConnectorState  = 2   // Uint16 RO ENUM
	em2GoHomeRegErrorCode       = 4   // Uint16 RO ENUM
	em2GoHomeRegCurrents        = 6   // Uint16 RO 0.1A
	em2GoHomeRegPower           = 12  // Uint32 RO 1W
	em2GoHomeRegEnergy          = 28  // Uint16 RO 0.1KWh
	em2GoHomeRegMaxCurrent      = 32  // Uint16 RO 0.1A
	em2GoHomeRegMinCurrent      = 34  // Uint16 RO 0.1A
	em2GoHomeRegCableMaxCurrent = 36  // Uint16 RO 0.1A
	em2GoHomeRegSerial          = 38  // Chr[16] RO UTF16
	em2GoHomeRegChargedEnergy   = 72  // Uint16 RO 0.1kWh
	em2GoHomeRegChargeDuration  = 78  // Uint32 RO 1s
	em2GoHomeRegSafeCurrent     = 87  // Uint16 WR 0.1A
	em2GoHomeRegCommTimeout     = 89  // Uint16 WR 1s
	em2GoHomeRegCurrentLimit    = 91  // Uint16 WR 0.1A
	em2GoHomeRegChargeMode      = 93  // Uint16 WR ENUM
	em2GoHomeRegChargeCommand   = 95  // Uint16 WR ENUM
	em2GoHomeRegVoltages        = 109 // Uint16 RO 0.1V
	em2GoHomeRegPhases          = 200 // Set charging phase 1 unsigned
)

func init() {
	registry.Add("em2go-home", NewEm2GoHomeFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateEm2GoHome -b *Em2GoHome -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewEm2GoHomeFromConfig creates a Em2Go charger from generic config
func NewEm2GoHomeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEm2GoHome(cc.URI, cc.ID)
}

// NewEm2GoHome creates Em2GoHome charger
func NewEm2GoHome(uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	// Add delay of 60 miliseconds between requests
	conn.Delay(60 * time.Millisecond)

	log := util.NewLogger("em2go-home")
	conn.Logger(log.TRACE)

	wb := &Em2GoHome{
		log:  log,
		conn: conn,
	}

	_, v2, v3, err := wb.Voltages()

	var phases1p3p func(int) error
	if v2 != 0 && v3 != 0 {
		phases1p3p = wb.phases1p3p
	}

	return decorateEm2GoHome(wb, phases1p3p), err
}

// Status implements the api.Charger interface
func (wb *Em2GoHome) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(b) {
	case 1:
		return api.StatusA, nil
	case 2, 3:
		return api.StatusB, nil
	case 4, 6:
		return api.StatusC, nil
	case 5, 7:
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[1])
	}
}

// Enabled implements the api.Charger interface
func (wb *Em2GoHome) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegChargeCommand, 1)
	if err != nil {
		return false, err
	}

	u := binary.BigEndian.Uint16(b)

	return u == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Em2GoHome) Enable(enable bool) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, map[bool]uint16{true: 1, false: 2}[enable])

	_, err := wb.conn.WriteMultipleRegisters(em2GoHomeRegChargeCommand, 1, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Em2GoHome) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Em2GoHome)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Em2GoHome) MaxCurrentMillis(current float64) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(10*current))

	_, err := wb.conn.WriteMultipleRegisters(em2GoHomeRegCurrentLimit, 1, b)
	return err
}

var _ api.CurrentGetter = (*Em2GoHome)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb Em2GoHome) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, err
}

var _ api.Meter = (*Em2GoHome)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Em2GoHome) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b), nil
}

var _ api.MeterEnergy = (*Em2GoHome)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Em2GoHome) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUUint32ToFloat64(b) / 10, nil
}

// getPhaseValues returns 3 register values offset by 2
func (wb *Em2GoHome) getPhaseValues(reg uint16) (float64, float64, float64, error) {
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

var _ api.PhaseCurrents = (*Em2GoHome)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Em2GoHome) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(em2GoHomeRegCurrents)
}

var _ api.PhaseVoltages = (*Em2GoHome)(nil)

// Currents implements the api.PhaseVoltages interface
func (wb *Em2GoHome) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(em2GoHomeRegVoltages)
}

var _ api.ChargeRater = (*Em2GoHome)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Em2GoHome) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ api.ChargeTimer = (*Em2GoHome)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Em2GoHome) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegChargeDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Em2GoHome) phases1p3p(phases int) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(phases))

	_, err := wb.conn.WriteMultipleRegisters(em2GoHomeRegPhases, 1, b)
	return err
}

var _ api.Diagnosis = (*Em2GoHome)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Em2GoHome) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegStatus, 1); err == nil {
		fmt.Printf("\tCharging Station Status:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegErrorCode, 1); err == nil {
		fmt.Printf("\tError Code:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegMaxCurrent, 1); err == nil {
		fmt.Printf("\tEVSE Min. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegCableMaxCurrent, 1); err == nil {
		fmt.Printf("\tCable Max. Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	var serial []byte
	for reg := 0; reg < 8; reg++ {
		b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegSerial+2*uint16(reg), 2)
		if err != nil {
			return
		}
		serial = append(serial, b...)
	}
	fmt.Printf("\tSerial: %s\n", string(serial))
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegCurrentLimit, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%.1fA\n", float64(binary.BigEndian.Uint16(b)/10))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegChargeMode, 1); err == nil {
		fmt.Printf("\tCharge Mode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoHomeRegChargeCommand, 1); err == nil {
		fmt.Printf("\tCharge Command:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
