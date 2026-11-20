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
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

type SchneiderG4S struct {
	log   *util.Logger
	conn  *modbus.Connection
	curr  uint16
	read  func(address, quantity uint16) ([]byte, error)
	write func(address, value uint16) error
}

const (
	schneiderG4SRegCpwState            = 6
	schneiderG4SRegEvState             = 8
	schneiderG4SRegLastStopCause       = 9
	schneiderG4SRegCommandStatus       = 20
	schneiderG4SRegChargingTime        = 30
	schneiderG4SRegSetCommand          = 150
	schneiderG4SRegSetPoint            = 301
	schneiderG4SRegCurrents            = 350
	schneiderG4SRegEnergy              = 356
	schneiderG4SRegPower               = 358
	schneiderG4SRegVoltages            = 366
	schneiderG4SRegLifebit             = 932
	schneiderG4SRegSessionChargingTime = 2004

	schneiderG4SDisabled = uint16(0)
)

func init() {
	registry.AddCtx("schneider-g4s", NewSchneiderG4SFromConfig)
}

func NewSchneiderG4SFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSchneiderG4S(ctx, cc)
}

func NewSchneiderG4S(ctx context.Context, settings modbus.TcpSettings) (api.Charger, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("schneider-g4s")
	conn.Logger(log.TRACE)

	wb := &SchneiderG4S{
		log:  log,
		conn: conn,
		curr: 6,
	}

	// Detect the supported Modbus register protocol. Only fall back to input
	// registers when holding-register access is rejected with a Modbus exception.
	b, err := wb.conn.ReadHoldingRegisters(schneiderG4SRegSetPoint, 1)
	switch {
	case err == nil:
		wb.log.DEBUG.Println("using FC03/FC06 protocol")

		wb.read = wb.conn.ReadHoldingRegisters
		wb.write = func(address, value uint16) error {
			_, err := wb.conn.WriteSingleRegister(address, value)
			return err
		}

	case modbus.IsException(err):
		b2, err2 := wb.conn.ReadInputRegisters(schneiderG4SRegSetPoint, 1)
		if err2 != nil {
			return nil, fmt.Errorf("current limit: %w", errors.Join(err, err2))
		}
		b = b2

		wb.log.DEBUG.Println("using FC04/FC16 protocol")

		wb.read = wb.conn.ReadInputRegisters
		wb.write = func(address, value uint16) error {
			data := make([]byte, 2)
			binary.BigEndian.PutUint16(data, value)
			_, err := wb.conn.WriteMultipleRegisters(address, 1, data)
			return err
		}

	default:
		return nil, fmt.Errorf("current limit: %w", err)
	}

	if u := encoding.Uint16(b); u > wb.curr {
		wb.curr = u
	}

	b, err = wb.read(schneiderG4SRegLifebit, 1)
	if err != nil {
		return nil, fmt.Errorf("heartbeat timeout: %w", err)
	}
	if u := encoding.Uint16(b); u != 2 {
		go wb.heartbeat(ctx, 2*time.Second)
	}

	return wb, nil
}

func (wb *SchneiderG4S) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if err := wb.write(schneiderG4SRegLifebit, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *SchneiderG4S) Status() (api.ChargeStatus, error) {
	b, err := wb.read(schneiderG4SRegEvState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := encoding.Uint16(b)

	switch s {
	case 0:
		return api.StatusA, nil
	case 1, 2:
		return api.StatusB, nil
	case 3, 4:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *SchneiderG4S) Enabled() (bool, error) {
	b, err := wb.read(schneiderG4SRegSetPoint, 1)
	if err != nil {
		return false, err
	}

	return encoding.Uint16(b) != schneiderG4SDisabled, nil
}

// Enable implements the api.Charger interface
func (wb *SchneiderG4S) Enable(enable bool) error {
	u := schneiderG4SDisabled
	if enable {
		u = wb.curr
	}

	return wb.write(schneiderG4SRegSetPoint, u)
}

// MaxCurrent implements the api.Charger interface
func (wb *SchneiderG4S) MaxCurrent(current int64) error {
	err := wb.write(schneiderG4SRegSetPoint, uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *SchneiderG4S) CurrentPower() (float64, error) {
	b, err := wb.read(schneiderG4SRegPower, 2)
	if err != nil {
		return 0, err
	}
	return float64(encoding.Float32(b)) * 1e3, nil
}

var _ api.MeterEnergy = (*SchneiderG4S)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SchneiderG4S) TotalEnergy() (float64, error) {
	b, err := wb.read(schneiderG4SRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(b)) / 1e3, nil
}

var _ api.PhaseCurrents = (*SchneiderG4S)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SchneiderG4S) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(schneiderG4SRegCurrents)
}

var _ api.PhaseVoltages = (*SchneiderG4S)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *SchneiderG4S) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(schneiderG4SRegVoltages)
}

// getPhaseValues returns 3 sequential phase values
func (wb *SchneiderG4S) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.read(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(encoding.Float32(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

var _ api.ChargeTimer = (*SchneiderG4S)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *SchneiderG4S) ChargeDuration() (time.Duration, error) {
	b, err := wb.read(schneiderG4SRegSessionChargingTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint32(b)) * time.Second, nil
}

var _ api.Diagnosis = (*SchneiderG4S)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *SchneiderG4S) Diagnose() {
	if b, err := wb.read(schneiderG4SRegCpwState, 1); err == nil {
		fmt.Printf("\tcpwState:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.read(schneiderG4SRegEvState, 1); err == nil {
		fmt.Printf("\tevState:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.read(schneiderG4SRegLastStopCause, 1); err == nil {
		fmt.Printf("\tLast stop cause:\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.read(schneiderG4SRegCommandStatus, 1); err == nil {
		fmt.Printf("\tCommand status:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.read(schneiderG4SRegSetCommand, 1); err == nil {
		fmt.Printf("\tSet command:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.read(schneiderG4SRegSetPoint, 1); err == nil {
		fmt.Printf("\tSet Point:\t\t%d\n", encoding.Uint16(b))
	}
	if b, err := wb.read(schneiderG4SRegLifebit, 1); err == nil {
		fmt.Printf("\tLifebit:\t\t%d\n", encoding.Uint16(b))
	}
}
