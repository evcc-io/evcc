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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Pulsares charger implementation
type Pulsares struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
}

const (
	pulsaresRegConnectionStatus = 0x1b
	pulsaresRegChargeStatus     = 0x1f
	pulsaresRegCurrent          = 0x5d
	pulsaresRegBackup           = 0x61
)

func init() {
	registry.Add("pulsares", NewPulsaresFromConfig)
}

// NewPulsaresFromConfig creates a Pulsares charger from generic config
func NewPulsaresFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPulsares(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID)
}

// NewPulsares creates Pulsares charger
func NewPulsares(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, proto, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("pulsares")
	conn.Logger(log.TRACE)

	wb := &Pulsares{
		conn: conn,
		curr: 6000,
	}

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr >= 6000 {
		wb.curr = curr
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(pulsaresRegBackup, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}

	var t time.Duration
	switch u := binary.BigEndian.Uint16(b); u {
	case 2:
		t = 2 * time.Second
	case 3:
		t = 5 * time.Second
	case 4:
		t = 10 * time.Second
	case 5:
		t = 30 * time.Second
	case 6:
		t = 60 * time.Second
	}

	if t > 0 {
		go wb.heartbeat(t / 2)
	}

	return wb, err
}

func (wb *Pulsares) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.conn.ReadHoldingRegisters(pulsaresRegBackup, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *Pulsares) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(pulsaresRegCurrent, 1, b)

	return err
}

func (wb *Pulsares) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(pulsaresRegCurrent, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// Status implements the api.Charger interface
func (wb *Pulsares) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(pulsaresRegConnectionStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	if binary.BigEndian.Uint16(b) != 1 {
		return api.StatusA, nil
	}

	b, err = wb.conn.ReadHoldingRegisters(pulsaresRegChargeStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	if u := binary.BigEndian.Uint16(b); u == 3 || u == 4 {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *Pulsares) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr >= 6000, err
}

// Enable implements the api.Charger interface
func (wb *Pulsares) Enable(enable bool) error {
	var curr uint16
	if enable {
		curr = wb.curr
	}

	return wb.setCurrent(curr)
}

// MaxCurrent implements the api.Charger interface
func (wb *Pulsares) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Pulsares)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *Pulsares) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	curr := uint16(current * 1e3)

	err := wb.setCurrent(curr)
	if err == nil {
		wb.curr = curr
	}

	return err
}
