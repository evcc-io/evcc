package charger

// LICENSE

// Copyright (c) 2023 premultiply

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
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// Mennekes is an api.Charger implementation
type Mennekes struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	mennekesRegModbusVersion        = 0x0000 // uint16
	mennekesRegFirmwareVersion      = 0x0001 // ascii[16]
	mennekesRegSerialNumber         = 0x0013 // ascii[16]
	mennekesRegEvseState            = 0x0100 // uint16
	mennekesRegAuthorizationStatus  = 0x0101 // uint16
	mennekesRegCpState              = 0x0108 // uint16
	mennekesRegChargingCurrentEM    = 0x0302 // float32
	mennekesRegPhaseOptionsHW       = 0x030C // uint16
	mennekesRegGridPhasesConnected  = 0x0311 // uint16
	mennekesRegAuthorization        = 0x0312 // uint16
	mennekesRegCurrents             = 0x0500 // float32[3]
	mennekesRegVoltages             = 0x0506 // float32[3]
	mennekesRegPower                = 0x0512 // float32
	mennekesRegChargedEnergySession = 0x0B02 // float32
	mennekesRegDurationSession      = 0x0B04 // uint32
	mennekesRegHeartbeat            = 0x0D00 // uint16
	mennekesRegRequestedPhases      = 0x0D04 // uint16
	mennekesRegChargingReleaseEM    = 0x0D05 // uint16
	mennekesRegChargedEnergyTotal   = 0x1000 // float32

	mennekesAllowed           = 1
	mennekesHeartbeatInterval = 10
	mennekesHeartbeatToken    = 0x55AA // 21930
)

func init() {
	registry.Add("mennekes", NewMennekesFromConfig)
}

// NewMennekesFromConfig creates a new Mennekes ModbusTCP charger
func NewMennekesFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Timeout         time.Duration
	}{
		Settings: modbus.Settings{
			Baudrate: 57600,
			Comset:   "8N2",
			ID:       50,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMennekes(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID, cc.Timeout)
}

// NewMennekes creates Mennekes charger
func NewMennekes(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, timeout time.Duration) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	if timeout > 0 {
		conn.Timeout(timeout)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("mennekes")
	conn.Logger(log.TRACE)

	wb := &Mennekes{
		log:  log,
		conn: conn,
	}

	// failsafe
	go wb.heartbeat(time.Duration(mennekesHeartbeatInterval) * time.Second)

	return wb, err
}

func (wb *Mennekes) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.WriteSingleRegister(mennekesRegHeartbeat, mennekesHeartbeatToken); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Mennekes) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegEvseState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch status := encoding.Uint16(b); status {
	case 1:
		return api.StatusA, nil

	case 2, 3, 4:
		return api.StatusB, nil

	case 5:
		return api.StatusC, nil

	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *Mennekes) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegChargingReleaseEM, 1)
	if err != nil {
		return false, err
	}
	u := encoding.Uint16(b)
	return u == mennekesAllowed, nil
}

// Enable implements the api.Charger interface
func (wb *Mennekes) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}
	_, err := wb.conn.WriteSingleRegister(mennekesRegChargingReleaseEM, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Mennekes) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Mennekes)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Mennekes) MaxCurrentMillis(current float64) error {
	b := make([]byte, 4)

	// LSW first
	u := math.Float32bits(float32(current))
	binary.BigEndian.PutUint16(b, uint16(u))
	binary.BigEndian.PutUint16(b[2:], uint16(u>>16))

	_, err := wb.conn.WriteMultipleRegisters(mennekesRegChargingCurrentEM, 2, b)
	return err
}

// CurrentPower implements the api.Meter interface
func (wb *Mennekes) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32LswFirst(b)) * 1e3, nil
}

var _ api.MeterEnergy = (*Mennekes)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Mennekes) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegChargedEnergyTotal, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(b)), nil
}

var _ api.PhaseCurrents = (*Mennekes)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Mennekes) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(mennekesRegCurrents)
}

var _ api.PhaseVoltages = (*Mennekes)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Mennekes) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(mennekesRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *Mennekes) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Float32LswFirst(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

/*
var _ api.ChargeRater = (*Mennekes)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Mennekes) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegChargedEnergySession, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32LswFirst(b)), err
}

var _ api.ChargeTimer = (*Mennekes)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Mennekes) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegDurationSession, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint32(b)) * time.Second, nil
}
*/

var _ api.PhaseSwitcher = (*Mennekes)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Mennekes) Phases1p3p(phases int) error {
	var u uint16
	if phases == 3 {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(mennekesRegRequestedPhases, u)
	return err
}

var _ api.Diagnosis = (*Mennekes)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Mennekes) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(mennekesRegModbusVersion, 1); err == nil {
		fmt.Printf("\tModbus:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(mennekesRegFirmwareVersion, 8); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", encoding.StringLsbFirst(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(mennekesRegSerialNumber, 8); err == nil {
		fmt.Printf("\tSerial:\t%s\n", encoding.StringLsbFirst(b))
	}
}
