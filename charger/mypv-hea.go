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
	"slices"
	"sync/atomic"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// MyPvHea charger implementation
type MyPvHea struct {
	log   *util.Logger
	conn  *modbus.Connection
	lp    loadpoint.API
	power uint32
	// scale float64
	relais  uint16
	enabled bool
	regTemp uint16
}

func init() {
	registry.AddCtx("mypv-hea-35", func(ctx context.Context, other map[string]any) (api.Charger, error) {
		return newMyPvHeaFromConfig(ctx, util.NewLogger("mypv-hea-35"), other, 1)
	})

}

// newMyPvHeaFromConfig creates a MyPv charger from generic config
func newMyPvHeaFromConfig(ctx context.Context, log *util.Logger, other map[string]any, relays uint16) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		TempSource         int
		// Scale              float64
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1, // default
		},
		TempSource: 1,
		// Scale:      1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMyPvHea(ctx, name, cc.TcpSettings, cc.TempSource, statusC)
}

// NewMyPvHea creates myPV HEA charger
func NewMyPvHea(ctx context.Context, name string, settings modbus.TcpSettings, tempSource int, statusC uint16) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	if tempSource < 1 || tempSource > len(elwaTemp) {
		return nil, fmt.Errorf("invalid temp source: %d", tempSource)
	}

	log := util.NewLogger(name)
	conn.Logger(log.TRACE)

	wb := &MyPvHea{
		log:     log,
		conn:    conn,
		name:    name,
		statusC: statusC,
		// scale:   scale,
		regTemp: elwaTemp[tempSource-1],
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
	var b []byte
	var err error

	if wb.name == "ac-thor" {
		b, err := wb.conn.ReadHoldingRegisters(elwaRegLoadState, 1)
		if err != nil {
			return api.StatusNone, err
		}

		// all loads detached
		if binary.BigEndian.Uint16(b) == 0 {
			return api.StatusA, nil
		}
	}

	res := api.StatusB

	b, err = wb.conn.ReadHoldingRegisters(elwaRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	c, err := wb.conn.ReadHoldingRegisters(elwaRegPower, 1)
	if err != nil {
		return api.StatusNone, err
	}

	// ignore standby power
	if binary.BigEndian.Uint16(b) == wb.statusC && binary.BigEndian.Uint16(c) > elwaStandbyPower {
		res = api.StatusC
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *MyPvHea) Enabled() (bool, error) {
	// "ac-thor" and "ac-elwa-2"
	reg := elwaRegOperationState
	enabled := []uint16{1, 2} // heating PV excess, boost backup

	if wb.name == "ac-elwa-e" {
		reg = elwaERegOperationState
		enabled = []uint16{2, 4} // heating PV excess, boost backup
	}

	// register read
	b, err := wb.conn.ReadHoldingRegisters(uint16(reg), 1)
	if err != nil {
		return false, err
	}
	state := binary.BigEndian.Uint16(b)

	// determine enabled state
	if state == 0 { // standby
		return false, nil
	}
	if slices.Contains(enabled, state) {
		return true, nil
	}

	// fallback to cached value as last resort
	return wb.enabled, nil
}

func (wb *MyPvHea) setPower(power uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(wb.scale*float64(power)))

	_, err := wb.conn.WriteMultipleRegisters(elwaRegSetPower, 1, b)
	return err
}

// Enable implements the api.Charger interface
func (wb *MyPvHea) Enable(enable bool) error {
	var power uint16
	if enable {
		power = uint16(atomic.LoadUint32(&wb.power))
	}

	res := wb.setPower(power)
	if res == nil {
		wb.enabled = enable
	}

	return res
}

// MaxCurrent implements the api.Charger interface
func (wb *MyPvHea) MaxCurrent(current int64) error {
	phases := 1
	if wb.lp != nil {
		if p := wb.lp.GetPhases(); p != 0 {
			phases = p
		}
	}

	power := uint16(voltage * float64(current) * float64(phases))

	err := wb.setPower(power)
	if err == nil {
		atomic.StoreUint32(&wb.power, uint32(power))
	}

	return err
}

var _ api.Meter = (*MyPvHea)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MyPvHea) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(elwaRegPower, 1)
	if err != nil {
		return 0, err
	}

	res := float64(binary.BigEndian.Uint16(b))
	if wb.name != "ac-thor" {
		return res, nil
	}

	c, err := wb.conn.ReadHoldingRegisters(elwaRegOperationMode, 1)
	if err != nil {
		return 0, err
	}
	wb.log.TRACE.Printf("operation mode %d", binary.BigEndian.Uint16(c))

	// AC Thor operation mode != 3
	if binary.BigEndian.Uint16(c) != 3 {
		return res, nil
	}

	// AC Thor operation mode == 3 "Warm water 9 + 9kW"
	// with extra heater on internal relay
	// see https://github.com/evcc-io/evcc/discussions/23708
	f, err := wb.conn.ReadHoldingRegisters(elwaRegRelayState, 1)
	if err != nil {
		return 0, err
	}

	// relay inactive
	if binary.BigEndian.Uint16(f) != 1 {
		return res, nil
	}

	// get power of heater on relay as set in web interface
	// (scale factor must be used for correct setting in web interface)
	d, err := wb.conn.ReadHoldingRegisters(elwaRegMaxControlledPower, 1)
	if err != nil {
		return 0, err
	}

	e, err := wb.conn.ReadHoldingRegisters(elwaRegMaxCombinedPower, 1)
	if err != nil {
		return 0, err
	}
	wb.log.TRACE.Printf("max. power: controlled %.0f W / combined %.0f W", float64(binary.BigEndian.Uint16(d)), float64(binary.BigEndian.Uint16(e)))

	// relay power = combined power - controlled power, finally corrected with 110% factor
	res += float64(int(binary.BigEndian.Uint16(e))-int(binary.BigEndian.Uint16(d))) / wb.scale / 1.1

	return res, nil
}

var _ api.Battery = (*MyPvHea)(nil)

// CurrentPower implements the api.Meter interface
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
	b, err := wb.conn.ReadHoldingRegisters(elwaRegTempLimit, 1)
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
