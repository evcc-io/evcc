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
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// GoodWe AC EV Charger Gen2 (e.g. GW11K-HCA) — Modbus TCP.
// Protocol reference: "AC EV Charger 2nd Gen Modbus Protocol v1.0.15"
// derived from the Home Assistant integration on the gen2-modbus branch:
//   https://github.com/pedrodivisez/goodwe-wallbox-sems-home-assistant/tree/gen2-modbus
// Default device ID is 247 (0xF7); TCP Modbus must be enabled in SolarGo.

type GoodWe struct {
	lp   loadpoint.API
	conn *modbus.Connection
}

const (
	goodweRegVoltages      = 10009 // U16 ×0.1 V, 3 regs (L1/L2/L3)
	goodweRegCurrents      = 10012 // U16 ×0.1 A, 3 regs (L1/L2/L3)
	goodweRegActualPower   = 10015 // U16 ×0.1 kW
	goodweRegSessionEnergy = 10016 // U16 ×0.1 kWh
	goodweRegStatus        = 10017 // U16 enum (see status mapping)
	goodweRegPhaseSwitch   = 10023 // U16 (1=single-phase, 0=three-phase)
	goodweRegMaxPower      = 10029 // U16 ×0.1 kW, raw range [14,220]
	goodweRegChargeMode    = 10032 // U16 (0=fast, 1=PV, 2=PV+battery)
	goodweRegSerial        = 10040 // 8 regs ASCII (16 bytes)
	goodweRegChargeCommand = 10060 // U16 (1=stop, 2=start)
	goodweRegTotalEnergy   = 10065 // U32 ×0.1 kWh, 2 regs

	goodweChargeStop     = 1
	goodweChargeStart    = 2
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

	log := util.NewLogger("goodwe")
	conn.Logger(log.TRACE)

	wb := &GoodWe{conn: conn}

	// force "fast" charging mode so evcc fully controls power setpoint
	if _, err := wb.conn.WriteSingleRegister(goodweRegChargeMode, goodweChargeModeFast); err != nil {
		return nil, fmt.Errorf("set charge mode: %w", err)
	}

	return wb, nil
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
	case 5, 7, 8, 9: // alarm, maintenance, start_failed, upgrading
		return api.StatusNone, fmt.Errorf("wallbox in error state %d", s)
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

	phases := 3
	if wb.lp != nil {
		if p := wb.lp.ActivePhases(); p != 0 {
			phases = p
		}
	}

	// raw register unit is 0.1 kW; clamp to documented [14,220] range
	raw := uint16(voltage * current * float64(phases) / 100)
	switch {
	case raw < 14:
		raw = 14
	case raw > 220:
		raw = 220
	}

	_, err := wb.conn.WriteSingleRegister(goodweRegMaxPower, raw)
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

var _ api.ChargeRater = (*GoodWe)(nil)

// ChargedEnergy implements api.ChargeRater
func (wb *GoodWe) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegSessionEnergy, 1)
	if err != nil {
		return 0, err
	}
	return float64(binary.BigEndian.Uint16(b)) / 10, nil
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

var _ api.PhaseSwitcher = (*GoodWe)(nil)

// Phases1p3p implements api.PhaseSwitcher. Only meaningful on 3-phase models
// (11/22 kW): single-phase mode allows charging down to 1.4 kW instead of 4.2 kW.
func (wb *GoodWe) Phases1p3p(phases int) error {
	var v uint16
	switch phases {
	case 1:
		v = 1
	case 3:
		v = 0
	default:
		return fmt.Errorf("invalid phase count: %d", phases)
	}
	_, err := wb.conn.WriteSingleRegister(goodweRegPhaseSwitch, v)
	return err
}

var _ api.Identifier = (*GoodWe)(nil)

// Identify implements api.Identifier
func (wb *GoodWe) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(goodweRegSerial, 8)
	if err != nil {
		return "", err
	}
	return bytesAsString(b), nil
}

var _ loadpoint.Controller = (*GoodWe)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *GoodWe) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
