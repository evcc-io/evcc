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
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Ego charger implementation for E.G.O. Smart Heater
type Ego struct {
	*embed
	log   *util.Logger
	conn  *modbus.Connection
	lp    loadpoint.API
	power uint32
}

const (
	egoRegRelais1Power   = 4096 // 0x1000
	egoRegRelais2Power   = 4128 // 0x1020
	egoRegRelais3Power   = 4160 // 0x1040
	egoRegRelaisStatus   = 5128 // 0x1408
	egoRegTempBoiler     = 5124 // 0x1404
	egoRegTempMax        = 4618 // 0x120A
	egoRegTempNominal    = 4619 // 0x120B
	egoRegPowerNominal   = 4864 // 0x1300
	egoRegHomeTotalPower = 4865 // 0x1301
	egoRegManufacturerID = 8192 // 0x2000
	egoModbusID          = 247  // Modbus slave address
)

func init() {
	registry.AddCtx("ego-smartheater", newEgoFromConfig)
}

// newEgoFromConfig creates an Ego charger from generic config
func newEgoFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed              `mapstructure:",squash"`
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		embed: embed{
			Icon_:     "heater",
			Features_: []api.Feature{api.IntegratedDevice, api.Heating},
		},
		TcpSettings: modbus.TcpSettings{
			ID: egoModbusID,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEgo(ctx, &cc.embed, cc.URI, cc.ID)
}

// NewEgo creates E.G.O. Smart Heater charger
func NewEgo(ctx context.Context, embed *embed, uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("ego")
	conn.Logger(log.TRACE)

	wb := &Ego{
		embed: embed,
		log:   log,
		conn:  conn,
	}

	// Verify connection by reading manufacturer ID
	b, err := wb.conn.ReadHoldingRegisters(egoRegManufacturerID, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to verify connection: %w", err)
	}

	manufacturerID := binary.BigEndian.Uint16(b)
	if manufacturerID != 0x14ef {
		return nil, fmt.Errorf("unexpected manufacturer ID: 0x%04x (expected 0x14ef)", manufacturerID)
	}

	go wb.heartbeat(ctx, 30*time.Second)

	return wb, nil
}

func (wb *Ego) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if power := int16(atomic.LoadUint32(&wb.power)); power > 0 {
			enabled, err := wb.Enabled()
			if err == nil && enabled {
				err = wb.setPower(power)
			}
			if err != nil {
				wb.log.ERROR.Println("heartbeat:", err)
			}
		}
	}
}

// Status implements the api.Charger interface
func (wb *Ego) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(egoRegRelaisStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	relaisStatus := binary.BigEndian.Uint16(b)

	// If any relay is on, status is C (heating)
	if relaisStatus != 0 {
		return api.StatusC, nil
	}

	// Otherwise status is B (standby)
	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *Ego) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(egoRegPowerNominal, 1)
	if err != nil {
		return false, err
	}

	power := int16(binary.BigEndian.Uint16(b))

	// PowerNominalValue = -1 means automatic mode (enabled)
	// PowerNominalValue > 0 means manual mode with specific power (enabled)
	// PowerNominalValue = 0 means disabled
	return power != 0, nil
}

func (wb *Ego) setPower(power int16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(power))

	_, err := wb.conn.WriteMultipleRegisters(egoRegPowerNominal, 1, b)
	return err
}

// Enable implements the api.Charger interface
func (wb *Ego) Enable(enable bool) error {
	var power int16
	if enable {
		power = int16(atomic.LoadUint32(&wb.power))
		if power == 0 {
			power = -1 // automatic mode
		}
	}

	return wb.setPower(power)
}

// MaxCurrent implements the api.Charger interface
func (wb *Ego) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Ego)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Ego) MaxCurrentMillis(current float64) error {
	phases := 1
	if wb.lp != nil {
		if p := wb.lp.GetPhases(); p != 0 {
			phases = p
		}
	}

	// Calculate power from current
	// E.G.O. Smart Heater works in manual mode with specific power values
	power := int16(voltage * current * float64(phases))

	err := wb.setPower(power)
	if err == nil {
		atomic.StoreUint32(&wb.power, uint32(power))
	}

	return err
}

var _ api.Meter = (*Ego)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Ego) CurrentPower() (float64, error) {
	// Read relay status to determine which relays are active
	b, err := wb.conn.ReadHoldingRegisters(egoRegRelaisStatus, 1)
	if err != nil {
		return 0, err
	}

	relay := binary.BigEndian.Uint16(b)
	var res uint16

	// Read individual relay powers and sum based on status
	for bit, addr := range map[uint16]uint16{
		0x01: egoRegRelais1Power,
		0x02: egoRegRelais2Power,
		0x04: egoRegRelais3Power,
	} {
		if relay&bit != 0 { // Relay is on
			b, err := wb.conn.ReadHoldingRegisters(addr, 1)
			if err != nil {
				return 0, err
			}
			res += binary.BigEndian.Uint16(b)
		}
	}

	return float64(res), nil
}

var _ api.Battery = (*Ego)(nil)

// Soc implements the api.Battery interface (returns boiler temperature)
func (wb *Ego) Soc() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(egoRegTempBoiler, 1)
	if err != nil {
		return 0, err
	}

	return float64(int16(binary.BigEndian.Uint16(b))), nil
}

var _ api.SocLimiter = (*Ego)(nil)

// GetLimitSoc implements the api.SocLimiter interface (returns max temperature)
func (wb *Ego) GetLimitSoc() (int64, error) {
	b, err := wb.conn.ReadHoldingRegisters(egoRegTempMax, 1)
	if err != nil {
		return 0, err
	}

	return int64(binary.BigEndian.Uint16(b)), nil
}

var _ loadpoint.Controller = (*Ego)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Ego) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
