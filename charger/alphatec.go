package charger

// LICENSE

// Copyright (c) 2019-2023 andig, premultiply

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
	"github.com/evcc-io/evcc/util/sponsor"
)

// https://shop.alphatec-systeme.de/media/pdf/4d/0e/64/MontageanleitungwlFxbRgs4NKK3.pdf

// Alphatec charger implementation
type Alphatec struct {
	conn *modbus.Connection
	curr uint16
}

const (
	alphatecRegStatus     = 0
	alphatecRegAmpsConfig = 5
)

func init() {
	registry.Add("alphatec", NewAlphatecFromConfig)
}

// NewAlphatecFromConfig creates a Alphatec charger from generic config
func NewAlphatecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Timeout         time.Duration
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAlphatec(cc.URI, cc.Device, cc.Comset, cc.Baudrate, modbus.ProtocolFromRTU(cc.RTU), cc.ID, cc.Timeout)
}

// NewAlphatec creates Alphatec charger
func NewAlphatec(uri, device, comset string, baudrate int, proto modbus.Protocol, slaveID uint8, timeout time.Duration) (api.Charger, error) {
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

	log := util.NewLogger("alphatec")
	conn.Logger(log.TRACE)

	wb := &Alphatec{
		conn: conn,
		curr: 6,
	}

	// get initial state from charger
	curr, err := wb.getCurrent()
	if err != nil {
		return nil, fmt.Errorf("current limit: %w", err)
	}
	if curr > 0 {
		wb.curr = curr
	}

	return wb, err
}

func (wb *Alphatec) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(alphatecRegAmpsConfig, 1, b)

	return err
}

func (wb *Alphatec) getCurrent() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(alphatecRegAmpsConfig, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

// Status implements the api.Charger interface
func (wb *Alphatec) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(alphatecRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch u := binary.BigEndian.Uint16(b); u {
	case 1:
		return api.StatusA, nil
	case 2, 8:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *Alphatec) Enabled() (bool, error) {
	curr, err := wb.getCurrent()

	return curr >= 6, err
}

// Enable implements the api.Charger interface
func (wb *Alphatec) Enable(enable bool) error {
	var curr uint16
	if enable {
		curr = wb.curr
	}

	return wb.setCurrent(curr)
}

// MaxCurrent implements the api.Charger interface
func (wb *Alphatec) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	err := wb.setCurrent(uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}

	return err
}
