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
	"sync/atomic"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// MyPvHea charger implementation
type MyPvHea struct {
	conn      *modbus.Connection
	lp        loadpoint.API
	relays    uint16
	stepPower uint16
	regTemp   uint16
	mask      atomic.Uint32
	current   atomic.Uint64
}

const (
	heaRegPower          = 1000
	heaRegTempLimit      = 1002
	heaRegOperationState = 1077
	heaRegSetPower       = 1080
)

var heaTemp = []uint16{1001, 1030}

func init() {
	registry.AddCtx("mypv-hea-35", func(ctx context.Context, other map[string]any) (api.Charger, error) {
		return newMyPvHeaFromConfig(ctx, "mypv-hea-35", other, 1)
	})

	registry.AddCtx("mypv-hea-90", func(ctx context.Context, other map[string]any) (api.Charger, error) {
		return newMyPvHeaFromConfig(ctx, "mypv-hea-90", other, 3)
	})
}

func newMyPvHeaFromConfig(ctx context.Context, name string, other map[string]any, relays uint16) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		TempSource         int
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1, // default
		},
		TempSource: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMyPvHea(ctx, name, cc.TcpSettings, cc.TempSource, relays)
}

// NewMyPvHea creates a my-PV HEA charger.
func NewMyPvHea(ctx context.Context, name string, settings modbus.TcpSettings, tempSource int, relays uint16) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	if tempSource < 1 || tempSource > len(heaTemp) {
		return nil, fmt.Errorf("invalid temp source: %d", tempSource)
	}

	var stepPower uint16
	switch relays {
	case 1:
		stepPower = 3500
	case 3:
		stepPower = 3000
	default:
		return nil, fmt.Errorf("invalid relay count: %d", relays)
	}

	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger(name)
	conn.Logger(log.TRACE)

	wb := &MyPvHea{
		conn:      conn,
		relays:    relays,
		stepPower: stepPower,
		regTemp:   heaTemp[tempSource-1],
	}

	return wb, nil
}

var _ api.IconDescriber = (*MyPvHea)(nil)

// Icon implements the api.IconDescriber interface
func (v *MyPvHea) Icon() string {
	return "waterheater"
}

var _ api.FeatureDescriber = (*MyPvHea)(nil)

// Features implements the api.FeatureDescriber interface
func (wb *MyPvHea) Features() []api.Feature {
	return []api.Feature{api.IntegratedDevice, api.Heating}
}

// Status implements the api.Charger interface
func (wb *MyPvHea) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(heaRegOperationState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	if state := binary.BigEndian.Uint16(b); state >= 4 {
		return api.StatusNone, fmt.Errorf("device error: operation state %d", state)
	}

	power, err := wb.CurrentPower()
	if err != nil {
		return api.StatusNone, err
	}

	if power > 10 { // ignore standby power
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *MyPvHea) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(heaRegSetPower, 1)
	if err != nil {
		return false, err
	}
	return binary.BigEndian.Uint16(b)&((1<<wb.relays)-1) != 0, nil
}

func (wb *MyPvHea) setRelays(mask uint16) error {
	_, err := wb.conn.WriteSingleRegister(heaRegSetPower, mask)
	return err
}

// Enable implements the api.Charger interface
func (wb *MyPvHea) Enable(enable bool) error {
	var mask uint16
	if enable {
		mask = uint16(wb.mask.Load())
	}

	return wb.setRelays(mask)
}

// MaxCurrent implements the api.Charger interface
func (wb *MyPvHea) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*MyPvHea)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface.
func (wb *MyPvHea) MaxCurrentMillis(current float64) error {
	if current < 0 || math.IsNaN(current) || math.IsInf(current, 0) {
		return fmt.Errorf("invalid current: %g", current)
	}

	phases := int(wb.relays)
	if wb.lp != nil {
		if p := wb.lp.GetPhases(); p != 0 {
			phases = p
		}
	}

	power := voltage * current * float64(phases)
	// Tolerate floating-point conversion at exact stage boundaries, but never round up a partial stage.
	steps := uint16(min(math.Floor(power/float64(wb.stepPower)+1e-9), float64(wb.relays)))
	mask := uint16(1<<steps) - 1

	err := wb.setRelays(mask)
	if err == nil {
		wb.mask.Store(uint32(mask))
		wb.current.Store(math.Float64bits(current))
	}

	return err
}

var _ api.CurrentGetter = (*MyPvHea)(nil)

// GetMaxCurrent returns the requested current rather than the discrete relay-stage current.
func (wb *MyPvHea) GetMaxCurrent() (float64, error) {
	return math.Float64frombits(wb.current.Load()), nil
}

var _ api.PowerLimiter = (*MyPvHea)(nil)

// GetMinMaxPower implements the api.PowerLimiter interface.
func (wb *MyPvHea) GetMinMaxPower() (float64, float64, error) {
	return float64(wb.stepPower), float64(wb.stepPower * wb.relays), nil
}

var _ api.Meter = (*MyPvHea)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MyPvHea) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(heaRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.Battery = (*MyPvHea)(nil)

// Soc implements the api.Battery interface (returns water temperature).
func (wb *MyPvHea) Soc() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.regTemp, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ api.SocLimiter = (*MyPvHea)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (wb *MyPvHea) GetLimitSoc() (int64, error) {
	b, err := wb.conn.ReadHoldingRegisters(heaRegTempLimit, 1)
	if err != nil {
		return 0, err
	}

	return int64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ loadpoint.Controller = (*MyPvHea)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *MyPvHea) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
