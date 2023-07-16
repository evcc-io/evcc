package charger

// LICENSE

// Copyright (c) 2019-2022 andig

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
	conn   *modbus.Connection
	status api.ChargeStatus
}

const (
	alphatecRegStatus     = 0
	alphatecRegEnable     = 4
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
		conn:   conn,
		status: api.StatusB,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Alphatec) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(alphatecRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	var res api.ChargeStatus
	switch u := binary.BigEndian.Uint16(b); u {
	case 1:
		res = api.StatusA
	case 2:
		res = api.StatusB
	case 3:
		res = api.StatusC
	case 8:
		res = wb.status
		if wb.status == api.StatusC {
			res = api.StatusB
		}
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}

	wb.status = res

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *Alphatec) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(alphatecRegEnable, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 0, nil
}

// Enable implements the api.Charger interface
func (wb *Alphatec) Enable(enable bool) error {
	b := make([]byte, 2)
	if !enable {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(alphatecRegEnable, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Alphatec) MaxCurrent(current int64) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(alphatecRegAmpsConfig, 1, b)

	return err
}
