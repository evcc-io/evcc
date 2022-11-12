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
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/meters/rs485"
)

// https://github.com/evcc-io/evcc/discussions/1965

// Alfen charger implementation
type Alfen struct {
	log     *util.Logger
	conn    *modbus.Connection
	mu      sync.Mutex
	curr    float64
	enabled bool
}

const (
	alfenRegCurrents   = 320 // 3 registers
	alfenRegPower      = 344
	alfenRegEnergy     = 374  // 390
	alfenRegStatus     = 1201 // 5 registers
	alfenRegAmpsConfig = 1210
	alfenRegPhases     = 1215
)

func init() {
	registry.Add("alfen", NewAlfenFromConfig)
}

// NewAlfenFromConfig creates a Alfen charger from generic config
func NewAlfenFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAlfen(cc.URI, cc.ID)
}

// NewAlfen creates Alfen charger
func NewAlfen(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("alfen")
	conn.Logger(log.TRACE)

	wb := &Alfen{
		log:  log,
		conn: conn,
	}

	go wb.heartbeat()

	return wb, err
}

func (wb *Alfen) heartbeat() {
	for range time.NewTicker(25 * time.Second).C {
		wb.mu.Lock()
		var curr float64
		if wb.enabled {
			curr = wb.curr
		}
		wb.mu.Unlock()

		if err := wb.setCurrent(curr); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Alfen) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegStatus, 5)
	if err != nil {
		return api.StatusNone, err
	}

	switch r := rune(b[0]); r {
	case 'A', 'B', 'D', 'E', 'F':
		return api.ChargeStatus(r), nil
	case 'C':
		// C1 is "connected"
		if rune(b[1]) == '1' {
			return api.StatusB, nil
		}
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[:1])
	}
}

// Enabled implements the api.Charger interface
func (wb *Alfen) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegAmpsConfig, 2)
	if err != nil {
		return false, err
	}

	return math.Float32frombits(binary.BigEndian.Uint32(b)) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Alfen) Enable(enable bool) error {
	var curr float64
	if enable {
		wb.mu.Lock()
		curr = wb.curr
		wb.mu.Unlock()
	}

	err := wb.setCurrent(curr)
	if err == nil {
		wb.mu.Lock()
		wb.enabled = enable
		wb.mu.Unlock()
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Alfen) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Alfen)(nil)

// setCurrent sets the current in milliamps without modifying the stored current value
func (wb *Alfen) setCurrent(current float64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(float32(current)))

	_, err := wb.conn.WriteMultipleRegisters(alfenRegAmpsConfig, 2, b)

	return err
}

// MaxCurrent implements the api.ChargerEx interface
func (wb *Alfen) MaxCurrentMillis(current float64) error {
	err := wb.setCurrent(current)
	if err == nil {
		wb.mu.Lock()
		wb.curr = current
		wb.mu.Unlock()
	}

	return err
}

var _ api.Meter = (*Alfen)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Alfen) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegPower, 2)
	if err != nil {
		return 0, err
	}

	return rs485.RTUIeee754ToFloat64(b), err
}

var _ api.MeterEnergy = (*Alfen)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Alfen) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return math.Float64frombits(binary.BigEndian.Uint64(b)) / 1e3, err
}

var _ api.MeterCurrent = (*Alfen)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Alfen) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(alfenRegCurrents, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		f := rs485.RTUIeee754ToFloat64(b[4*i:])
		if math.IsNaN(f) {
			f = 0
		}

		res[i] = f
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseSwitcher = (*Alfen)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *Alfen) Phases1p3p(phases int) error {
	_, err := c.conn.WriteSingleRegister(alfenRegPhases, uint16(phases))
	return err
}

// var _ api.Diagnosis = (*Alfen)(nil)

// // Diagnose implements the api.Diagnosis interface
// func (wb *Alfen) Diagnose() {
// 	b, err := wb.conn.ReadHoldingRegisters(ablRegFirmware, 2)
// 	if err == nil {
// 		fmt.Printf("Firmware: %0 x\n", b)
// 	}
// }
