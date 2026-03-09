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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// https://api.library.loxone.com/downloader/file/2425/Modbus%20Protocol%20RAEDIAN%20AC%20Wallbox%20v0.3.pdf

// Raedian charger implementation
type Raedian struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	raedianRegStatus        = 32780 // uint32 RO ENUM (0=A, 1=B, 2/3/4=C)
	raedianRegCurrents      = 32784 // uint32 RO mA
	raedianRegVoltages      = 32790 // uint32 RO 0.1V
	raedianRegPower         = 32796 // uint32 RO W
	raedianRegChargedEnergy = 32798 // uint32 RO Wh
	raedianRegMaxCurrent    = 33024 // uint32 WR mA
	raedianRegEnable        = 33029 // uint16 WR (0=enabled, 1=disabled)
)

func init() {
	registry.AddCtx("raedian", NewRaedianFromConfig)
}

// NewRaedianFromConfig creates a Raedian charger from generic config
func NewRaedianFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewRaedian(ctx, cc.URI, cc.ID)
}

// NewRaedian creates Raedian charger
func NewRaedian(ctx context.Context, uri string, slaveID uint8) (*Raedian, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("raedian")
	conn.Logger(log.TRACE)

	wb := &Raedian{
		log:  log,
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Raedian) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegStatus, 2)
	if err != nil {
		return api.StatusNone, err
	}

	u := binary.BigEndian.Uint32(b)
	switch u {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2, 3, 4:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", u)
	}
}

// Enabled implements the api.Charger interface
func (wb *Raedian) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegEnable, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 0, nil
}

// Enable implements the api.Charger interface
func (wb *Raedian) Enable(enable bool) error {
	var val uint16 = 1
	if enable {
		val = 0
	}

	_, err := wb.conn.WriteSingleRegister(raedianRegEnable, val)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Raedian) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Raedian)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Raedian) MaxCurrentMillis(current float64) error {
	curr := uint32(current * 1000) // convert A to mA

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, curr)

	_, err := wb.conn.WriteMultipleRegisters(raedianRegMaxCurrent, 2, b)
	return err
}

var _ api.Meter = (*Raedian)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Raedian) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.ChargeRater = (*Raedian)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Raedian) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(raedianRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1000, nil
}

// getPhaseValues reads 3 sequential uint32 registers in a single Modbus operation
func (wb *Raedian) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Raedian)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Raedian) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(raedianRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*Raedian)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Raedian) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(raedianRegVoltages, 10)
}
