package charger

// LICENSE

// Copyright (c) 2025 andig

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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Compleo charger implementation
type Compleo struct {
	lp    loadpoint.API
	conn  *modbus.Connection
	base  uint16
	power uint16
}

const (
	// global
	compleoRegBase       = 0x0100 // input
	compleoRegFallback   = 0x5    // holding
	compleoRegConnectors = 0x0008 // input

	// per connector
	compleoRegMaxPower       = 0x0 // holding
	compleoRegStatus         = 0x1 // input
	compleoRegActualPower    = 0x2 // input
	compleoRegCurrents       = 0x3 // input
	compleoRegChargeDuration = 0x6 // input
	compleoRegEnergy         = 0x8 // input
	compleoRegVoltages       = 0xD // input

	compleoRegIdTag = 0x1000 - compleoRegBase // input
)

func init() {
	registry.AddCtx("compleo", NewCompleoFromConfig)
}

// NewCompleoFromConfig creates a Compleo charger from generic config
func NewCompleoFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Connector          uint16
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		Connector: 1,
		TcpSettings: modbus.TcpSettings{
			ID: 0xFF,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewCompleo(ctx, cc.URI, cc.ID, cc.Connector)
}

// NewCompleo creates Compleo charger
func NewCompleo(ctx context.Context, uri string, slaveID uint8, connector uint16) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("compleo")
	conn.Logger(log.TRACE)

	b, err := conn.ReadInputRegisters(compleoRegConnectors, 1)
	if err != nil {
		return nil, err
	}

	if connector > binary.BigEndian.Uint16(b) {
		return nil, fmt.Errorf("invalid connector: %d", connector)
	}

	wb := &Compleo{
		conn:  conn,
		base:  compleoRegBase + (connector-1)*0x010,
		power: 3 * 230 * 6, // assume min power
	}

	// heartbeat
	b, err = wb.conn.ReadHoldingRegisters(compleoRegFallback, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(ctx, log, time.Duration(u)*time.Second/2)
	}

	return wb, nil
}

func (wb *Compleo) heartbeat(ctx context.Context, log *util.Logger, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if _, err := wb.status(); err != nil {
			log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *Compleo) status() (byte, error) {
	b, err := wb.conn.ReadInputRegisters(compleoRegStatus, 1)
	if err != nil {
		return 0, err
	}

	return b[1], nil
}

// Status implements the api.Charger interface
func (wb *Compleo) Status() (api.ChargeStatus, error) {
	s, err := wb.status()
	if err != nil {
		return api.StatusNone, err
	}

	res := api.StatusA
	if s&1 > 0 {
		res = api.StatusB
	}
	if s&2 > 0 {
		res = api.StatusC
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *Compleo) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(compleoRegMaxPower, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Compleo) Enable(enable bool) error {
	var power uint16
	if enable {
		power = wb.power
	}

	return wb.setPower(power)
}

// setPower writes the power limit in 100W steps
func (wb *Compleo) setPower(power uint16) error {
	_, err := wb.conn.WriteSingleRegister(compleoRegMaxPower, power/100)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Compleo) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Compleo)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *Compleo) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	phases := 3
	if wb.lp != nil {
		if p := wb.lp.ActivePhases(); p != 0 {
			phases = p
		}
	}

	power := uint16(voltage * current * float64(phases))

	err := wb.setPower(power)
	if err == nil {
		wb.power = power
	}

	return err
}

var _ api.Meter = (*Compleo)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Compleo) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(compleoRegActualPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) * 100, err
}

var _ api.ChargeRater = (*Compleo)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Compleo) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(compleoRegEnergy, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, err
}

// getPhaseValues returns 3 sequential register values
func (wb *Compleo) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Compleo)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Compleo) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(compleoRegCurrents, 10)
}

var _ api.PhaseVoltages = (*Compleo)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Compleo) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(compleoRegVoltages, 1)
}

var _ api.ChargeTimer = (*Compleo)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Compleo) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(compleoRegChargeDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.LittleEndian.Uint32(b)) * time.Second, nil
}

var _ api.Identifier = (*Compleo)(nil)

// Identify implements the api.Identifier interface
func (wb *Compleo) Identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(compleoRegIdTag, 0x10)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

var _ loadpoint.Controller = (*Compleo)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Compleo) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
