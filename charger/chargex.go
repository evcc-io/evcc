package charger

// LICENSE

// Copyright (c) 2019-2023 andig

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
	"sync"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// https://github.com/evcc-io/evcc/discussions/1965

// ChargeX charger implementation
type ChargeX struct {
	log     *util.Logger
	conn    *modbus.Connection
	mu      sync.Mutex
	curr    float64
	enabled bool
}

const (
	chargexRegDeviceType     = 0x0c // 8 registers
	chargexRegFirmware       = 0x16 // 8 registers
	chargexRegSerial         = 0x1c // 8 registers
	chargexRegPowerSum       = 0x24 // 2 registers
	chargexRegCurrents       = 0x28 // 3x2 registers
	chargexRegConnectedCars  = 0x2e // 1 register
	chargexRegActiveSessions = 0x30 // 1 register
	chargexRegModules        = 0x32 // 1 register
	chargexRegSystemStatus   = 0x34 // 1 register
	chargexRegMaxCurrent     = 0x36 // 2 registers
)

func init() {
	registry.Add("chargex", NewChargeXFromConfig)
}

// NewChargeXFromConfig creates a ChargeX charger from generic config
func NewChargeXFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewChargeX(cc.URI, cc.ID)
}

// NewChargeX creates ChargeX charger
func NewChargeX(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("chargex")
	conn.Logger(log.TRACE)

	wb := &ChargeX{
		log:  log,
		conn: conn,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *ChargeX) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargexRegConnectedCars, 1)
	if err != nil {
		return api.StatusNone, err
	}

	if binary.BigEndian.Uint16(b) > 0 {
		b, err = wb.conn.ReadHoldingRegisters(chargexRegActiveSessions, 1)
		if err != nil {
			return api.StatusNone, err
		}
		if binary.BigEndian.Uint16(b) > 0 {
			return api.StatusC, nil
		}

		return api.StatusB, nil
	}

	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (wb *ChargeX) Enabled() (bool, error) {
	// b, err := wb.conn.ReadHoldingRegisters(alfenRegAmpsConfig, 2)
	// if err != nil {
	// 	return false, err
	// }

	// return math.Float32frombits(binary.BigEndian.Uint32(b)) > 0, nil
	return false, nil
}

// Enable implements the api.Charger interface
func (wb *ChargeX) Enable(enable bool) error {
	var curr float64
	if enable {
		wb.mu.Lock()
		curr = wb.curr
		wb.mu.Unlock()
	}
	_ = curr

	// err := wb.setCurrent(curr)
	// if err == nil {
	// 	wb.mu.Lock()
	// 	wb.enabled = enable
	// 	wb.mu.Unlock()
	// }

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *ChargeX) MaxCurrent(current int64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(current))
	_, err := wb.conn.WriteMultipleRegisters(chargexRegMaxCurrent, 2, b)
	return err
}

var _ api.Meter = (*ChargeX)(nil)

// CurrentPower implements the api.Meter interface
func (wb *ChargeX) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargexRegPowerSum, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.PhaseCurrents = (*ChargeX)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *ChargeX) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(chargexRegCurrents, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), float64(binary.BigEndian.Uint32(b[4:])), float64(binary.BigEndian.Uint32(b[8:])), nil
}

var _ api.Diagnosis = (*ChargeX)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *ChargeX) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegSerial, 8); err == nil {
		fmt.Printf("\tSerial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegDeviceType, 8); err == nil {
		fmt.Printf("\tDevice type:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegFirmware, 8); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegConnectedCars, 1); err == nil {
		fmt.Printf("\tConnected cars:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegActiveSessions, 1); err == nil {
		fmt.Printf("\tActive sessions:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegModules, 1); err == nil {
		fmt.Printf("\tModules:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(chargexRegSystemStatus, 1); err == nil {
		fmt.Printf("\tSystem status:\t%02x\n", b)
	}
}
