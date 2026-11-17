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
	"math"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	elliRegStatus   = 5
	elliRegCurrents = 6
	elliRegVoltages = 10
	elliRegPower    = 14
	elliRegEnergy   = 17
	elliRegMax      = 100
	elliRegTimeout  = 257
	elliRegCurrent  = 261
)

type elliModbusConn interface {
	ReadInputRegisters(uint16, uint16) ([]byte, error)
	ReadHoldingRegisters(uint16, uint16) ([]byte, error)
	WriteSingleRegister(uint16, uint16) ([]byte, error)
}

// Elli2Modbus is an Elli Charger 2 controlled over Modbus TCP.
type Elli2Modbus struct {
	conn elliModbusConn
	log  *util.Logger

	mu       sync.Mutex
	current  uint16
	command  uint16
	written  bool
	min, max float64
}

func init() {
	registry.AddCtx("elli-2-modbus", NewElli2ModbusFromConfig)
}

// NewElli2ModbusFromConfig creates an Elli Charger 2 from generic config.
func NewElli2ModbusFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
	}{TcpSettings: modbus.TcpSettings{ID: 1}}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewElli2Modbus(ctx, cc.TcpSettings)
}

// NewElli2Modbus creates an Elli Charger 2.
func NewElli2Modbus(ctx context.Context, settings modbus.TcpSettings) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("elli-2-modbus")
	conn.Logger(log.TRACE)
	wb := &Elli2Modbus{conn: conn, log: log}

	b, err := conn.ReadInputRegisters(elliRegMax, 2)
	if err != nil {
		return nil, fmt.Errorf("current limits: %w", err)
	}
	wb.min = max(6, float64(binary.BigEndian.Uint16(b[2:])))
	wb.max = min(16, float64(binary.BigEndian.Uint16(b)))
	if wb.max < wb.min {
		return nil, fmt.Errorf("invalid current limits: %.0f-%.0f A", wb.min, wb.max)
	}
	wb.current = uint16(wb.min * 10)

	b, err = conn.ReadHoldingRegisters(elliRegTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("watchdog timeout: %w", err)
	}
	if timeout := binary.BigEndian.Uint16(b); timeout > 0 {
		go wb.heartbeat(ctx, time.Duration(timeout)*time.Millisecond/2)
	}

	return wb, nil
}

func (wb *Elli2Modbus) heartbeat(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wb.mu.Lock()
			if wb.written && ctx.Err() == nil {
				if _, err := wb.conn.WriteSingleRegister(elliRegCurrent, wb.command); err != nil {
					wb.log.ERROR.Println("heartbeat:", err)
				}
			}
			wb.mu.Unlock()
		}
	}
}

// Status implements api.Charger.
func (wb *Elli2Modbus) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(elliRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}
	state := binary.BigEndian.Uint16(b)
	switch state {
	case 2, 3:
		return api.StatusA, nil
	case 4, 5, 6:
		return api.StatusB, nil
	case 7, 8:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", state)
	}
}

// Enabled implements api.Charger.
func (wb *Elli2Modbus) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(elliRegCurrent, 1)
	if err != nil {
		return false, err
	}
	return binary.BigEndian.Uint16(b) != 0, nil
}

func (wb *Elli2Modbus) setCurrent(current uint16) error {
	if _, err := wb.conn.WriteSingleRegister(elliRegCurrent, current); err != nil {
		wb.written = false
		return err
	}
	wb.command = current
	wb.written = true
	return nil
}

// Enable implements api.Charger.
func (wb *Elli2Modbus) Enable(enable bool) error {
	wb.mu.Lock()
	defer wb.mu.Unlock()
	if enable {
		return wb.setCurrent(wb.current)
	}
	return wb.setCurrent(0)
}

// MaxCurrent implements api.Charger.
func (wb *Elli2Modbus) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Elli2Modbus)(nil)

// MaxCurrentMillis implements api.ChargerEx.
func (wb *Elli2Modbus) MaxCurrentMillis(current float64) error {
	if math.IsNaN(current) || math.IsInf(current, 0) || current < wb.min || current > wb.max {
		return fmt.Errorf("invalid current %.1f A", current)
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()
	value := uint16(current * 10)
	if err := wb.setCurrent(value); err != nil {
		return err
	}
	wb.current = value
	return nil
}

var _ api.CurrentLimiter = (*Elli2Modbus)(nil)

// GetMinMaxCurrent implements api.CurrentLimiter.
func (wb *Elli2Modbus) GetMinMaxCurrent() (float64, float64, error) {
	return wb.min, wb.max, nil
}

var _ api.Meter = (*Elli2Modbus)(nil)

// CurrentPower implements api.Meter.
func (wb *Elli2Modbus) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(elliRegPower, 1)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.MeterEnergy = (*Elli2Modbus)(nil)

// TotalEnergy implements api.MeterEnergy.
func (wb *Elli2Modbus) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(elliRegEnergy, 2)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

func (wb *Elli2Modbus) phaseValues(reg uint16, scale float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}
	return float64(binary.BigEndian.Uint16(b)) / scale,
		float64(binary.BigEndian.Uint16(b[2:])) / scale,
		float64(binary.BigEndian.Uint16(b[4:])) / scale, nil
}

var _ api.PhaseCurrents = (*Elli2Modbus)(nil)

// Currents implements api.PhaseCurrents.
func (wb *Elli2Modbus) Currents() (float64, float64, float64, error) {
	return wb.phaseValues(elliRegCurrents, 10)
}

var _ api.PhaseVoltages = (*Elli2Modbus)(nil)

// Voltages implements api.PhaseVoltages.
func (wb *Elli2Modbus) Voltages() (float64, float64, float64, error) {
	return wb.phaseValues(elliRegVoltages, 1)
}
