package charger

// LICENSE

// Copyright (c) 2024 andig

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
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// MyPvElwa2 charger implementation
type MyPvElwa2 struct {
	log   *util.Logger
	conn  *modbus.Connection
	power uint32
}

const (
	elwaRegSetPower  = 1000
	elwaRegTemp      = 1001
	elwaRegTempLimit = 1002
	elwaRegStatus    = 1003
	elwaRegPower     = 1074
)

func init() {
	registry.AddCtx("ac-elwa-2", NewMyPvElwa2FromConfig)
}

// https://github.com/evcc-io/evcc/discussions/12761

// NewMyPvElwa2FromConfig creates a MyPvElwa2 charger from generic config
func NewMyPvElwa2FromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMyPvElwa2(ctx, cc.URI, cc.ID)
}

// NewMyPvElwa2 creates myPV AC Elwa 2 charger
func NewMyPvElwa2(ctx context.Context, uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("ac-elwa-2")
	conn.Logger(log.TRACE)

	wb := &MyPvElwa2{
		log:  log,
		conn: conn,
	}

	go wb.heartbeat(ctx, 30*time.Second)

	return wb, nil
}

var _ api.IconDescriber = (*MyPvElwa2)(nil)

// Icon implements the api.IconDescriber interface
func (v *MyPvElwa2) Icon() string {
	return "waterheater"
}

var _ api.FeatureDescriber = (*MyPvElwa2)(nil)

// Features implements the api.FeatureDescriber interface
func (wb *MyPvElwa2) Features() []api.Feature {
	return []api.Feature{api.IntegratedDevice, api.Heating}
}

func (wb *MyPvElwa2) heartbeat(ctx context.Context, timeout time.Duration) {
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
func (wb *MyPvElwa2) Status() (api.ChargeStatus, error) {
	res := api.StatusA
	b, err := wb.conn.ReadHoldingRegisters(elwaRegStatus, 1)
	if err != nil {
		return res, err
	}

	res = api.StatusB
	if binary.BigEndian.Uint16(b) == 2 {
		res = api.StatusC
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *MyPvElwa2) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(elwaRegSetPower, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

func (wb *MyPvElwa2) setPower(power uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, power)

	_, err := wb.conn.WriteMultipleRegisters(elwaRegSetPower, 1, b)
	return err
}

// Enable implements the api.Charger interface
func (wb *MyPvElwa2) Enable(enable bool) error {
	var power uint16
	if enable {
		power = uint16(atomic.LoadUint32(&wb.power))
	}

	return wb.setPower(power)
}

// MaxCurrent implements the api.Charger interface
func (wb *MyPvElwa2) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*MyPvElwa2)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *MyPvElwa2) MaxCurrentMillis(current float64) error {
	power := uint16(230 * current)

	err := wb.setPower(power)
	if err == nil {
		atomic.StoreUint32(&wb.power, uint32(power))
	}

	return err
}

var _ api.Meter = (*MyPvElwa2)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MyPvElwa2) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(elwaRegPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.Battery = (*MyPvElwa2)(nil)

// CurrentPower implements the api.Meter interface
func (wb *MyPvElwa2) Soc() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(elwaRegTemp, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 10, nil
}

var _ api.SocLimiter = (*MyPvElwa2)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (wb *MyPvElwa2) GetLimitSoc() (int64, error) {
	b, err := wb.conn.ReadHoldingRegisters(elwaRegTempLimit, 1)
	if err != nil {
		return 0, err
	}

	return int64(binary.BigEndian.Uint16(b)) / 10, nil
}
