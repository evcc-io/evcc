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

// Supports all chargers based on Phoenix Contact "EV-ETH" controller series
// EV-CC-AC1-M3-CBC-RCM-ETH, EV-CC-AC1-M3-CBC-RCM-ETH-3G, EV-CC-AC1-M3-RCM-ETH-XP, EV-CC-AC1-M3-RCM-ETH-3G-XP
// with OEM firmware from Phoenix Contact and modified firmware versions (Wallbe).
// All features should be autodetected.
// * Set DIP switch 10 to ON

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

type PhoenixEVEth struct {
	conn     *modbus.Connection
	isWallbe bool
}

const (
	phxRegStatus          = 100  // Input
	phxRegChargeTime      = 102  // Input
	phxRegFirmware        = 105  // Input
	phxRegVoltages        = 108  // Input
	phxRegCurrents        = 114  // Input
	phxRegPower           = 120  // Input
	phxRegEnergy          = 128  // Input
	phxRegChargedEnergy   = 132  // Input
	phxRegFirmwareWallbe  = 149  // Input
	phxRegEnable          = 400  // Coil
	phxRegCardEnabled     = 419  // Coil
	phxRegMaxCurrent      = 528  // Holding
	phxRegCardUID         = 606  // Holding
	phxRegEnergyWh        = 904  // Holding, 32bit, Wh (2), Wallbe: 16bit (1)
	phxRegEnergyWallbe    = 2980 // Holding, 64bit, Wh (4)
	phxRegChargedEnergyEx = 3376 // Holding, 64bit, Wh (4)
)

func init() {
	registry.Add("phoenix-ev-eth", NewPhoenixEVEthFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePhoenixEVEth -b *PhoenixEVEth -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.Identifier,Identify,func() (string, error)"

// NewPhoenixEVEthFromConfig creates a PhoenixEVEth charger from generic config
func NewPhoenixEVEthFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixEVEth(cc.URI, cc.ID)
}

// NewPhoenixEVEth creates a PhoenixEVEth charger
func NewPhoenixEVEth(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("ev-eth")
	conn.Logger(log.TRACE)

	wb := &PhoenixEVEth{
		conn: conn,
	}

	var (
		currentPower     func() (float64, error)
		totalEnergy      func() (float64, error)
		currents         func() (float64, float64, float64, error)
		voltages         func() (float64, float64, float64, error)
		maxCurrentMillis func(float64) error
		identify         func() (string, error)
	)

	// check presence of meter by voltage on l1
	if b, err := wb.conn.ReadInputRegisters(phxRegVoltages, 2); err == nil && encoding.Uint32LswFirst(b) > 0 {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
		voltages = wb.voltages
	}

	// check card reader enabled
	if b, err := wb.conn.ReadCoils(phxRegCardEnabled, 1); err == nil && b[0] == 1 {
		identify = wb.identify
	}

	// check presence of extended Wallbe firmware
	if b, err := wb.conn.ReadHoldingRegisters(phxRegMaxCurrent, 1); err == nil && encoding.Uint16(b) >= 60 {
		wb.isWallbe = true
		maxCurrentMillis = wb.maxCurrentMillis
	}

	return decoratePhoenixEVEth(wb, currentPower, totalEnergy, currents, voltages, maxCurrentMillis, identify), err
}

// Status implements the api.Charger interface
func (wb *PhoenixEVEth) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phxRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatusString(string(b[1]))
}

// Enabled implements the api.Charger interface
func (wb *PhoenixEVEth) Enabled() (bool, error) {
	b, err := wb.conn.ReadCoils(phxRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the api.Charger interface
func (wb *PhoenixEVEth) Enable(enable bool) error {
	var u uint16
	if enable {
		u = modbus.CoilOn
	}

	_, err := wb.conn.WriteSingleCoil(phxRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *PhoenixEVEth) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current)
	_, err := wb.conn.WriteSingleRegister(phxRegMaxCurrent, u)

	return err
}

// maxCurrentMillis implements the api.ChargerEx interface (Wallbe Firmware only)
func (wb *PhoenixEVEth) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	u := uint16(current * 10) // 0.1A Steps
	_, err := wb.conn.WriteSingleRegister(phxRegMaxCurrent, u)

	return err
}

// currentPower implements the api.Meter interface
func (wb *PhoenixEVEth) currentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(phxRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int32LswFirst(b)), nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *PhoenixEVEth) totalEnergy() (float64, error) {
	if wb.isWallbe {
		b, err := wb.conn.ReadHoldingRegisters(phxRegEnergyWallbe, 4)
		if err != nil {
			return 0, err
		}

		return float64(encoding.Uint64LswFirst(b)) / 1e3, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(phxRegEnergyWh, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32LswFirst(b)) / 1e3, nil
}

// currents implements the api.PhaseCurrents interface
func (wb *PhoenixEVEth) currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(phxRegCurrents)
}

// voltages implements the api.PhaseVoltages interface
func (wb *PhoenixEVEth) voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(phxRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *PhoenixEVEth) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(encoding.Int32LswFirst(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

// identify implements the api.Identifier interface
func (wb *PhoenixEVEth) identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(phxRegCardUID, 16)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

var _ api.Diagnosis = (*PhoenixEVEth)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *PhoenixEVEth) Diagnose() {
	if wb.isWallbe {
		if b, err := wb.conn.ReadInputRegisters(phxRegFirmwareWallbe, 6); err == nil {
			fmt.Printf("\tFirmware (Wallbe):\t%s\n", encoding.StringLsbFirst(b))
		}
	} else {
		if b, err := wb.conn.ReadInputRegisters(phxRegFirmware, 2); err == nil {
			fmt.Printf("\tFirmware (Phoenix):\t%s\n", encoding.StringLsbFirst(b))
		}
	}
}
