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
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// GoodWe AC EV Charger Gen2 (e.g. GW11K-HCA) — Modbus TCP.
// Protocol reference: "AC EV Charger 2nd Gen Modbus Protocol v1.0.15"
// derived from the Home Assistant integration on the gen2-modbus branch:
//   https://github.com/pedrodivisez/goodwe-wallbox-sems-home-assistant/tree/gen2-modbus
// Default device ID is 247 (0xF7); TCP Modbus must be enabled in SolarGo.

type GoodWe struct {
	implement.Caps
	lp       loadpoint.API
	conn     *modbus.Connection
	phases   int
	maxPower int
}

const (
	goodweRegVoltages       = 10009 // U16 ×0.1 V, 3 regs (L1/L2/L3)
	goodweRegCurrents       = 10012 // U16 ×0.1 A, 3 regs (L1/L2/L3)
	goodweRegActualPower    = 10015 // U16 ×0.1 kW
	goodweRegStatus         = 10017 // U16 enum (see status mapping)
	goodweRegPhaseSwEnabled = 10023 // U16 (0=Off, 1=On)
	goodweRegMaxPower       = 10029 // U16 ×0.1 kW, raw range [14,220]
	goodweRegChargeMode     = 10032 // U16 (0=fast, 1=PV, 2=PV+battery)
	goodweRegRfid           = 10050 // 7 regs ASCII (14 bytes)
	goodweRegPowerSpec      = 10058 // U16 (0=7kW, 1=11kW, 2=22kW)
	goodweRegPhaseSpec      = 10059 // U16 (0=1p, 1=3p)
	goodweRegChargeCommand  = 10060 // U16 (1=stop, 2=start)
	goodweRegTotalEnergy    = 10065 // U32 ×0.1 kWh, 2 regs

	goodweChargeStop  = 1
	goodweChargeStart = 2

	goodweChargeModeFast = 0
)

func init() {
	registry.AddCtx("goodwe-wallbox", NewGoodWeFromConfig)
}

// NewGoodWeFromConfig creates a GoodWe wallbox charger from generic config.
func NewGoodWeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{ID: 247},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewGoodWe(ctx, cc.URI, cc.ID)
}

// NewGoodWe creates a GoodWe wallbox charger.
func NewGoodWe(ctx context.Context, uri string, slaveID uint8) (api.Charger, error) {
	uri = util.DefaultPort(uri, 502)

	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("goodwe")
	conn.Logger(log.TRACE)

	// defaults if hardware capability registers can't be read:
	// assume 3-phase wallbox without phase switching at typical 11 kW class.
	wb := &GoodWe{
		Caps:     implement.New(),
		conn:     conn,
		phases:   3,
		maxPower: 11000,
	}

	// read hardware power spec (0=7kW, 1=11kW, 2=22kW); fall back to default on error/unknown
	if b, err := wb.conn.ReadHoldingRegisters(goodweRegPowerSpec, 1); err == nil {
		switch v := binary.BigEndian.Uint16(b); v {
		case 0:
			wb.phases = 1
			wb.maxPower = 7000
		case 1:
			wb.maxPower = 11000
		case 2:
			wb.maxPower = 22000
		default:
			log.WARN.Printf("unknown power spec %d, defaulting to %d W", v, wb.maxPower)
		}
	} else {
		log.WARN.Printf("read power capability failed, defaulting to %d W: %v", wb.maxPower, err)
	}

	// read hardware phase spec (0=1-phase, 1=3-phase); fall back to 3-phase on error
	if b, err := wb.conn.ReadHoldingRegisters(goodweRegPhaseSpec, 1); err == nil {
		if binary.BigEndian.Uint16(b) == 0 {
			wb.phases = 1
		}
	} else {
		log.WARN.Printf("read hw phase count failed, defaulting to 3-phase: %v", err)
	}

	// only 3-phase hardware can do 1p/3p switching; conditionally register phase switching
	// based on hardware capability. On read error, fall back to fixed-phase operation.
	if wb.phases == 3 {
		if b, err := wb.conn.ReadHoldingRegisters(goodweRegPhaseSwEnabled, 1); err == nil {
			if binary.BigEndian.Uint16(b) == 1 {
				implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
			}
		} else {
			log.WARN.Printf("read phase switch config failed, disabling dynamic phase switching: %v", err)
		}
	}

	// force "fast" charging mode so evcc fully controls power setpoint
	if _, err := wb.conn.WriteSingleRegister(goodweRegChargeMode, goodweChargeModeFast); err != nil {
		return nil, fmt.Errorf("set charge mode: %w", err)
	}

	return wb, nil
}

// calcPower converts a per-phase current target into the wallbox's 0.1 kW power setpoint.
// The protocol exposes only a total power register; how the wallbox converts that back into
// a per-phase current limit is unspecified — it may assume a fixed nominal voltage (e.g. 230 V)
// or use its own live voltage measurement. 230 V is used here as a best-effort approximation;
// actual delivered current may deviate accordingly until verified on hardware.
func (wb *GoodWe) calcPower(current float64, phases int) uint16 {
	return uint16(min(max(230*float64(phases)*current, 1400), float64(wb.maxPower)) / 100) // 0.1 kW
}

// Status implements api.ChargeState
func (wb *GoodWe) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 0: // idle, no plug
		return api.StatusA, nil
	case 1, 2, 4, 6, 10: // plugged, handshaking, completed, scheduled, interrupted
		return api.StatusB, nil
	case 3: // charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements api.Charger
func (wb *GoodWe) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegChargeCommand, 1)
	if err != nil {
		return false, err
	}
	return binary.BigEndian.Uint16(b) == goodweChargeStart, nil
}

// Enable implements api.Charger
func (wb *GoodWe) Enable(enable bool) error {
	v := uint16(goodweChargeStop)
	if enable {
		v = goodweChargeStart
	}
	_, err := wb.conn.WriteSingleRegister(goodweRegChargeCommand, v)
	return err
}

// MaxCurrent implements api.Charger
func (wb *GoodWe) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*GoodWe)(nil)

// MaxCurrentMillis implements api.ChargerEx
func (wb *GoodWe) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	_, err := wb.conn.WriteSingleRegister(goodweRegMaxPower, wb.calcPower(current, wb.phases))
	return err
}

var _ api.Meter = (*GoodWe)(nil)

// CurrentPower implements api.Meter
func (wb *GoodWe) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegActualPower, 1)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint16(b)) * 100, nil
}

var _ api.MeterEnergy = (*GoodWe)(nil)

// TotalEnergy implements api.MeterEnergy
func (wb *GoodWe) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint32(b)) / 10, nil
}

func (wb *GoodWe) phaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var v [3]float64
	for i := range v {
		v[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / divider
	}
	return v[0], v[1], v[2], nil
}

var _ api.PhaseCurrents = (*GoodWe)(nil)

// Currents implements api.PhaseCurrents
func (wb *GoodWe) Currents() (float64, float64, float64, error) {
	return wb.phaseValues(goodweRegCurrents, 10)
}

var _ api.PhaseVoltages = (*GoodWe)(nil)

// Voltages implements api.PhaseVoltages
func (wb *GoodWe) Voltages() (float64, float64, float64, error) {
	return wb.phaseValues(goodweRegVoltages, 10)
}

func (wb *GoodWe) phases1p3p(phases int) error {
	wb.phases = phases
	return nil
}

var _ api.Identifier = (*GoodWe)(nil)

// Identify implements api.Identifier (RFID UID, 14-byte NUL-padded ASCII).
func (wb *GoodWe) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegRfid, 7)
	if err != nil {
		return "", err
	}
	return bytesAsString(bytes.Trim(b, "\x00")), nil
}

var _ loadpoint.Controller = (*GoodWe)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *GoodWe) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
