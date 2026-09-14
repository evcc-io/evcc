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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// MyPv charger implementation
type MyPv struct {
	myPvBase
	log     *util.Logger
	power   uint32
	scale   float64
	name    string
	statusC uint16
	enabled bool
}

const (
	elwaRegSetPower           = 1000
	elwaRegStatus             = 1003
	elwaRegLoadState          = 1059
	elwaERegOperationState    = elwaRegStatus // same register for elwa-e operation state
	elwaRegRelayState         = 1058
	elwaRegVoltage            = 1061
	elwaRegOperationMode      = 1065 // https://github.com/evcc-io/evcc/discussions/23708
	elwaRegMaxControlledPower = 1014 // max. power for linear controlled output
	elwaRegMaxCombinedPower   = 1071 // (max. power for linear controlled output + configured relais power) * 1.10
)

var elwaTemp = []uint16{1001, 1030, 1031}
var elwaStandbyPower uint16 = 10

func init() {
	// https://github.com/evcc-io/evcc/discussions/12761
	registry.AddCtx("ac-elwa-2", func(ctx context.Context, other map[string]any) (api.Charger, error) {
		return newMyPvFromConfig(ctx, "ac-elwa-2", other, 2)
	})

	// https: // github.com/evcc-io/evcc/issues/18020
	registry.AddCtx("ac-thor", func(ctx context.Context, other map[string]any) (api.Charger, error) {
		return newMyPvFromConfig(ctx, "ac-thor", other, 9)
	})

	registry.AddCtx("ac-elwa-e", func(ctx context.Context, other map[string]any) (api.Charger, error) {
		return newMyPvFromConfig(ctx, "ac-elwa-e", other, 2)
	})
}

// newMyPvFromConfig creates a MyPv charger from generic config
func newMyPvFromConfig(ctx context.Context, name string, other map[string]any, statusC uint16) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		TempSource         int
		Scale              float64
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1, // default
		},
		TempSource: 1,
		Scale:      1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMyPv(ctx, name, cc.TcpSettings, cc.TempSource, statusC, cc.Scale)
}

// NewMyPv creates myPV AC Elwa 2 or Thor charger
func NewMyPv(ctx context.Context, name string, settings modbus.TcpSettings, tempSource int, statusC uint16, scale float64) (api.Charger, error) {
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

	wb := &MyPv{
		myPvBase: newMyPvBase(conn, elwaTemp[tempSource-1]),
		log:      log,
		name:     name,
		statusC:  statusC,
		scale:    scale,
	}

	go wb.heartbeat(ctx, 30*time.Second)

	return wb, nil
}

func (wb *MyPv) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if power := uint16(atomic.LoadUint32(&wb.power)); power > 0 {
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
func (wb *MyPv) Status() (api.ChargeStatus, error) {
	if wb.name == "ac-thor" {
		load, err := wb.readUint16(elwaRegLoadState)
		if err != nil {
			return api.StatusNone, err
		}

		// all loads detached
		if load == 0 {
			return api.StatusA, nil
		}
	}

	res := api.StatusB

	state, err := wb.readUint16(elwaRegStatus)
	if err != nil {
		return api.StatusNone, err
	}

	power, err := wb.readUint16(myPvRegPower)
	if err != nil {
		return api.StatusNone, err
	}

	// ignore standby power
	if state == wb.statusC && power > elwaStandbyPower {
		res = api.StatusC
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *MyPv) Enabled() (bool, error) {
	// "ac-thor" and "ac-elwa-2"
	reg := myPvRegOperationState
	enabled := []uint16{1, 2} // heating PV excess, boost backup

	if wb.name == "ac-elwa-e" {
		reg = elwaERegOperationState
		enabled = []uint16{2, 4} // heating PV excess, boost backup
	}

	// register read
	state, err := wb.readUint16(uint16(reg))
	if err != nil {
		return false, err
	}

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

func (wb *MyPv) setPower(power uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(wb.scale*float64(power)))

	_, err := wb.conn.WriteMultipleRegisters(elwaRegSetPower, 1, b)
	return err
}

// Enable implements the api.Charger interface
func (wb *MyPv) Enable(enable bool) error {
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
func (wb *MyPv) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*MyPv)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *MyPv) MaxCurrentMillis(current float64) error {
	phases := 1
	if wb.lp != nil {
		if p := wb.lp.GetPhases(); p != 0 {
			phases = p
		}
	}

	power := uint16(voltage * current * float64(phases))

	err := wb.setPower(power)
	if err == nil {
		atomic.StoreUint32(&wb.power, uint32(power))
	}

	return err
}

var _ api.Meter = (*MyPv)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MyPv) CurrentPower() (float64, error) {
	power, err := wb.readUint16(myPvRegPower)
	if err != nil {
		return 0, err
	}

	res := float64(power)
	if wb.name != "ac-thor" {
		return res, nil
	}

	mode, err := wb.readUint16(elwaRegOperationMode)
	if err != nil {
		return 0, err
	}
	wb.log.TRACE.Printf("operation mode %d", mode)

	// AC Thor operation mode != 3
	if mode != 3 {
		return res, nil
	}

	// AC Thor operation mode == 3 "Warm water 9 + 9kW"
	// with extra heater on internal relay
	// see https://github.com/evcc-io/evcc/discussions/23708
	relay, err := wb.readUint16(elwaRegRelayState)
	if err != nil {
		return 0, err
	}

	// relay inactive
	if relay != 1 {
		return res, nil
	}

	// get power of heater on relay as set in web interface
	// (scale factor must be used for correct setting in web interface)
	controlled, err := wb.readUint16(elwaRegMaxControlledPower)
	if err != nil {
		return 0, err
	}

	combined, err := wb.readUint16(elwaRegMaxCombinedPower)
	if err != nil {
		return 0, err
	}
	wb.log.TRACE.Printf("max. power: controlled %.0f W / combined %.0f W", float64(controlled), float64(combined))

	// relay power = combined power - controlled power, finally corrected with 110% factor
	res += float64(int(combined)-int(controlled)) / wb.scale / 1.1

	return res, nil
}
