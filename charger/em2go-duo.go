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
)

// Em2GoDuo charger implementation
type Em2GoDuo struct {
	log       *util.Logger
	conn      *modbus.Connection
	base      uint16
	connector int
}

const (
	em2GoDuoRegSerial         = 0  // Chr[16] RO UTF16
	em2GoDuoRegCommTimeout    = 50 // Uint16 WR 1s
	em2GoDuoRegSafeCurrent    = 52 // Uint16 WR 0.1A
	em2GoDuoRegChargeMode     = 54 // Uint16 WR ENUM
	em2GoDuoRegConnectorState = 56 // Uint16 RO ENUM
	em2GoDuoRegCableState     = 58 // Uint16 RO ENUM
	em2GoDuoRegErrorCode      = 60 // Uint16 RO ENUM

	em2GoDuoRegConCurrents       = 0  // Uint16 RO 0.1A
	em2GoDuoRegConVoltages       = 6  // Uint16 RO 0.1V
	em2GoDuoRegConPower          = 24 // Uint32 RO 1W
	em2GoDuoRegConEnergy         = 28 // Uint32 RO 0.1KWh
	em2GoDuoRegConChargeDuration = 34 // Uint32 RO 1s
	em2GoDuoRegConCurrentLimit   = 44 // Uint16 WR 1A
	em2GoDuoRegConChargeCommand  = 46 // Uint16 WR ENUM
)

func init() {
	registry.AddCtx("em2go-duo", NewEm2GoDuoFromConfig)
}

// NewEm2GoDuoFromConfig creates a Em2GoDuo charger from generic config
func NewEm2GoDuoFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Connector          int
	}{
		TcpSettings: modbus.TcpSettings{ID: 255},
		Connector:   1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEm2GoDuo(ctx, cc.URI, cc.ID, cc.Connector)
}

// NewEm2GoDuo creates Em2GoDuo charger
func NewEm2GoDuo(ctx context.Context, uri string, slaveID uint8, connector int) (api.Charger, error) {
	if connector < 1 || connector > 2 {
		return nil, fmt.Errorf("invalid connector %d, must be 1 or 2", connector)
	}

	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	// Add delay of 60 milliseconds between requests
	conn.Delay(60 * time.Millisecond)

	log := util.NewLogger("em2go-duo")
	conn.Logger(log.TRACE)

	wb := &Em2GoDuo{
		log:       log,
		conn:      conn,
		base:      256 * uint16(connector),
		connector: connector,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Em2GoDuo) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(em2GoDuoRegConnectorState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	// High 8-bit value for connector 2, low 8-bit value for connector 1
	if wb.connector == 2 {
		s >>= 8
	}

	switch s & 0xff {
	case
		1, // Available
		9: // Unavailable
		return api.StatusA, nil
	case
		2, // Preparing
		4, // SuspendedEV
		5, // SuspendedEVSE
		6: // Finishing
		return api.StatusB, nil
	case
		3: // Charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Em2GoDuo) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+em2GoDuoRegConChargeCommand, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Em2GoDuo) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(wb.base+em2GoDuoRegConChargeCommand, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Em2GoDuo) MaxCurrent(current int64) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(wb.base+em2GoDuoRegConCurrentLimit, 1, b)

	return err
}

var _ api.CurrentGetter = (*Em2GoDuo)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *Em2GoDuo) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+em2GoDuoRegConCurrentLimit, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), err
}

var _ api.Meter = (*Em2GoDuo)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Em2GoDuo) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+em2GoDuoRegConPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.MeterEnergy = (*Em2GoDuo)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Em2GoDuo) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+em2GoDuoRegConEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, nil
}

// getPhaseValues returns 3 register values offset by 2
func (wb *Em2GoDuo) getPhaseValues(reg uint16) (float64, float64, float64, error) {
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

var _ api.PhaseCurrents = (*Em2GoDuo)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Em2GoDuo) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(wb.base + em2GoDuoRegConCurrents)
}

var _ api.PhaseVoltages = (*Em2GoDuo)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Em2GoDuo) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(wb.base + em2GoDuoRegConVoltages)
}

var _ api.ChargeTimer = (*Em2GoDuo)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Em2GoDuo) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+em2GoDuoRegConChargeDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.Diagnosis = (*Em2GoDuo)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Em2GoDuo) Diagnose() {
	fmt.Printf("\tConnector:\t%d\n", wb.connector)
	if b, err := wb.conn.ReadHoldingRegisters(em2GoDuoRegConnectorState, 1); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoDuoRegCableState, 1); err == nil {
		fmt.Printf("\tCable State:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoDuoRegErrorCode, 1); err == nil {
		fmt.Printf("\tError Code:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoDuoRegSafeCurrent, 1); err == nil {
		fmt.Printf("\tSafe Current:\t%.1fA\n", float64(binary.BigEndian.Uint16(b))/10)
	}
	if b, err := wb.conn.ReadHoldingRegisters(em2GoDuoRegCommTimeout, 1); err == nil {
		fmt.Printf("\tConnection Timeout:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
