package charger

// LICENSE

// Copyright (c) 2023 andig

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
	"math"
	"math/rand"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Heizstab charger implementation
type Heizstab struct {
	conn    *modbus.Connection
	icon    string
	enabled bool
	curr    float64
	scale   float64
}

func init() {
	registry.Add("heizstab", NewHeizstabFromConfig)
}

const (
	hzCurrent = 0x1
	hzStatus  = 0x2
	hzTemp    = 0x3
)

// NewHeizstabFromConfig creates a Heizstab charger from generic config
func NewHeizstabFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Icon            string
		Scale           float64
	}{
		Icon: "heater",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHeizstab(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID, cc.Scale, cc.Icon)
}

// NewHeizstab creates Heizstab charger
func NewHeizstab(uri, device, comset string, baudrate int, slaveID uint8, scale float64, icon string) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.Ascii, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("heizstab")
	conn.Logger(log.TRACE)

	wb := &Heizstab{
		conn:  conn,
		icon:  icon,
		scale: scale,
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Heizstab) Status() (api.ChargeStatus, error) {
	return api.StatusB, nil

	b, err := wb.conn.ReadHoldingRegisters(hzStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	status := api.StatusB
	if binary.BigEndian.Uint16(b) > 0 {
		status = api.StatusC
	}

	return status, nil
}

// Enabled implements the api.Charger interface
func (wb *Heizstab) Enabled() (bool, error) {
	return wb.enabled, nil
}

func (wb *Heizstab) power(current float64) uint16 {
	return uint16(math.Round(current * wb.scale))
}

// Enable implements the api.Charger interface
func (wb *Heizstab) Enable(enable bool) error {
	wb.enabled = enable
	return nil

	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, wb.power(wb.curr))
	}

	_, err := wb.conn.WriteMultipleRegisters(hzCurrent, 1, b)
	if err == nil {
		wb.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Heizstab) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Heizstab)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *Heizstab) MaxCurrentMillis(current float64) error {
	return nil

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, wb.power(current))

	_, err := wb.conn.WriteMultipleRegisters(hzCurrent, 1, b)
	if err == nil {
		wb.curr = current
	}

	return err
}

var _ api.Meter = (*Heizstab)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Heizstab) CurrentPower() (float64, error) {
	if wb.enabled {
		return wb.curr * wb.scale, nil
	}
	return 0, nil
}

var _ api.Battery = (*Heizstab)(nil)

// CurrentPower implements the api.Battery interface
func (wb *Heizstab) Soc() (float64, error) {
	return float64(rand.Int31n(40) + 10), nil

	b, err := wb.conn.ReadHoldingRegisters(hzTemp, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ api.IconDescriber = (*Heizstab)(nil)

// Icon implements api.IconDescriber
func (wb *Heizstab) Icon() string {
	return wb.icon
}

var _ api.FeatureDescriber = (*Heizstab)(nil)

// Features implements api.FeatureDescriber
func (wb *Heizstab) Features() []api.Feature {
	return []api.Feature{api.IntegratedDevice, api.Heating}
}
