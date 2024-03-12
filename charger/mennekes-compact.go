package charger

// LICENSE

// Copyright (c) 2023, 2024 premultiply, andig, Marcel Ludwig <malud>

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

	"github.com/volkszaehler/mbmd/encoding"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/mennekes"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	mennekesCompactName = "mennekes-compact"

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
	mennekesRegActiveErrorCode      = 0x0E00 // uint16
	mennekesRegChargedEnergyTotal   = 0x1000 // float32

	mennekesAllowed           = 1
	mennekesHeartbeatInterval = 8 * time.Second
	mennekesHeartbeatToken    = 0x55AA // 21930
)

// MennekesCompact is an api.Charger implementation
type MennekesCompact struct {
	log  *util.Logger
	conn *modbus.Connection
}

type MennekesCompactConfig struct {
	ModbusSettings    modbus.Settings `mapstructure:",squash"`
	ModbusTimeout     time.Duration   `mapstructure:"timeout"`
	HeartbeatInterval time.Duration   `mapstructure:"heartbeat"`
}

var defaultMennekesCompactConfig = MennekesCompactConfig{
	ModbusSettings: modbus.Settings{
		ID:       50,
		Baudrate: 57600,
		Comset:   "8N2",
	},
	HeartbeatInterval: mennekesHeartbeatInterval,
}

func init() {
	registry.Add(mennekesCompactName, NewMennekesCompactFromConfig)
}

// NewMennekesCompactFromConfig creates a new Mennekes Compact Modbus charger
func NewMennekesCompactFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := defaultMennekesCompactConfig
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// maxHeartbeatInterval according to the modbus documentation
	const maxHeartbeatInterval = time.Second * 10
	if cc.HeartbeatInterval >= maxHeartbeatInterval {
		return nil, fmt.Errorf("heartbeat interval must be less than %v", maxHeartbeatInterval)
	}

	return NewMennekesCompact(&cc)
}

// NewMennekesCompact creates Mennekes Compact charger
func NewMennekesCompact(conf *MennekesCompactConfig) (api.Charger, error) {
	conn, err := modbus.NewConnectionFromSettings(&conf.ModbusSettings)
	if err != nil {
		return nil, err
	}

	if conf.ModbusTimeout > 0 {
		conn.Timeout(conf.ModbusTimeout)
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger(mennekesCompactName)
	conn.Logger(log.TRACE)

	wb := &MennekesCompact{
		log:  log,
		conn: conn,
	}

	// send heartbeat on startup and don't wait for the first tick
	wb.doHeartbeat()
	// initiate heartbeat with given interval
	go wb.heartbeat(mennekesHeartbeatInterval)

	return wb, err
}

// heartbeat sends a regular heartbeat to the charger.
func (wb *MennekesCompact) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		wb.doHeartbeat()
	}
}

func (wb *MennekesCompact) doHeartbeat() {
	if _, err := wb.conn.WriteSingleRegister(mennekesRegHeartbeat, mennekesHeartbeatToken); err != nil {
		wb.log.ERROR.Println("heartbeat:", err)
	}
}

// Status implements the api.Charger interface
func (wb *MennekesCompact) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegEvseState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch state := mennekes.EVSEState(encoding.Uint16(b)); state {
	case mennekes.NotInitialized:
		return api.StatusNone, nil
	case mennekes.Idle:
		return api.StatusA, nil
	case mennekes.EVConnected, mennekes.PreconditionsValid, mennekes.ReadyToCharge:
		return api.StatusB, nil
	case mennekes.Charging:
		return api.StatusC, nil
	case mennekes.Error:
		if errCode, e := wb.conn.
			ReadHoldingRegisters(mennekesRegActiveErrorCode, 1); e == nil && encoding.Uint16(errCode) != 0 {
			return api.StatusNone, fmt.Errorf("invalid status: %s: code: %d", state, encoding.Uint16(errCode))
		}
		fallthrough
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", state)
	}
}

// Enabled implements the api.Charger interface
func (wb *MennekesCompact) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegChargingReleaseEM, 1)
	if err != nil {
		return false, err
	}
	u := encoding.Uint16(b)
	return u == mennekesAllowed, nil
}

// Enable implements the api.Charger interface
func (wb *MennekesCompact) Enable(enable bool) error {
	var u uint16
	if enable {
		u = mennekesAllowed
	}
	_, err := wb.conn.WriteSingleRegister(mennekesRegChargingReleaseEM, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *MennekesCompact) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*MennekesCompact)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *MennekesCompact) MaxCurrentMillis(current float64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(float32(current)))

	_, err := wb.conn.WriteMultipleRegisters(mennekesRegChargingCurrentEM, 2, b)
	return err
}

// CurrentPower implements the api.Meter interface
func (wb *MennekesCompact) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32(b)), nil
}

var _ api.MeterEnergy = (*MennekesCompact)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *MennekesCompact) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegChargedEnergyTotal, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32(b)), nil
}

var _ api.PhaseCurrents = (*MennekesCompact)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *MennekesCompact) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(mennekesRegCurrents)
}

var _ api.PhaseVoltages = (*MennekesCompact)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *MennekesCompact) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(mennekesRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *MennekesCompact) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Float32(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

/*
var _ api.ChargeRater = (*MennekesCompact)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *MennekesCompact) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegChargedEnergySession, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Float32(b)), err
}

var _ api.ChargeTimer = (*MennekesCompact)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *MennekesCompact) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(mennekesRegDurationSession, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint32(b)) * time.Second, nil
}
*/

var _ api.PhaseSwitcher = (*MennekesCompact)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *MennekesCompact) Phases1p3p(phases int) error {
	var u uint16
	if phases == 1 {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(mennekesRegRequestedPhases, u)
	return err
}

var _ api.Diagnosis = (*MennekesCompact)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *MennekesCompact) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(mennekesRegModbusVersion, 1); err == nil {
		fmt.Printf("\tModbus: %03X\n", encoding.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(mennekesRegFirmwareVersion, 8); err == nil {
		fmt.Printf("\tFirmware: %s\n", string(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(mennekesRegSerialNumber, 8); err == nil {
		fmt.Printf("\tSerial: %s\n", string(b))
	}
}
