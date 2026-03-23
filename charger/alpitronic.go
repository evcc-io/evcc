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
	"encoding/hex"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// AlpitronicHYC charger implementation
type AlpitronicHYC struct {
	log       *util.Logger
	conn      *modbus.Connection
	curr      float64
	connector uint16
}

const (
	// Input
	hycRegState              = 0
	hycRegChargingPower      = 4
	hycRegChargeTime         = 6
	hycRegChargedEnergy      = 7
	hycRegSoC                = 8
	hycRegVID                = 18
	hycRegIdTag              = 22
	hycRegTotalChargedEnergy = 32

	// Holding
	hycRegMaxPowerAC = 0
)

func init() {
	registry.AddCtx("alpitronic", NewAlpitronicHYCFromConfig)
}

// NewAlpitronicHYCFromConfig creates a Alpitronic charger from generic config
func NewAlpitronicHYCFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		Connector          uint16
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		Connector: 1,
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewAlpitronicHYC(ctx, cc.URI, cc.ID, cc.Connector)
}

// NewAlpitronicHYC creates Alpitronic charger
func NewAlpitronicHYC(ctx context.Context, uri string, id uint8, connector uint16) (*AlpitronicHYC, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("alpitronic")
	conn.Logger(log.TRACE)

	wb := &AlpitronicHYC{
		log:       log,
		conn:      conn,
		curr:      6,
		connector: connector,
	}

	return wb, nil
}

// setCurrent writes the current limit as power
func (wb *AlpitronicHYC) setCurrent(current float64) error {
	power := uint32(current * 230 * 3)
	b := make([]byte, 4)
	encoding.PutUint32(b, power)

	_, err := wb.conn.WriteMultipleRegisters(wb.reg(hycRegMaxPowerAC), 2, b)
	return err
}

// reg returns the register address for the connector
func (wb *AlpitronicHYC) reg(reg uint16) uint16 {
	return (wb.connector * 100) + reg
}

// Status implements the api.Charger interface
func (wb *AlpitronicHYC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegState), 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := encoding.Uint16(b); s {
	case 0, 7, 8, 9, 10:
		return api.StatusA, nil
	case 1, 2, 4, 5, 6:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *AlpitronicHYC) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.reg(hycRegMaxPowerAC), 2)
	if err != nil {
		return false, err
	}

	return encoding.Uint32(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *AlpitronicHYC) Enable(enable bool) error {
	var c float64
	if enable {
		c = wb.curr
	}

	return wb.setCurrent(c)
}

// MaxCurrent implements the api.Charger interface
func (wb *AlpitronicHYC) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*AlpitronicHYC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *AlpitronicHYC) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	err := wb.setCurrent(current)
	if err == nil {
		wb.curr = current
	}

	return err
}

var _ api.Meter = (*AlpitronicHYC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *AlpitronicHYC) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegChargingPower), 2)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(b)), err
}

var _ api.ChargeTimer = (*AlpitronicHYC)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *AlpitronicHYC) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegChargeTime), 1)
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint16(b)) * time.Second, nil
}

var _ api.ChargeRater = (*AlpitronicHYC)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *AlpitronicHYC) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegChargedEnergy), 1)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint16(b)) / 100, err
}

var _ api.MeterEnergy = (*AlpitronicHYC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *AlpitronicHYC) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegTotalChargedEnergy), 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int64(b)) / 1e3, err
}

var _ api.StatusReasoner = (*AlpitronicHYC)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *AlpitronicHYC) StatusReason() (api.Reason, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegState), 1)
	if err != nil {
		return api.ReasonUnknown, err
	}

	switch s := encoding.Uint16(b); s {
	case 1:
		return api.ReasonWaitingForAuthorization, nil
	case 6:
		return api.ReasonDisconnectRequired, nil
	default:
		return api.ReasonUnknown, nil
	}
}

var _ api.Identifier = (*AlpitronicHYC)(nil)

// Identify implements the api.Identifier interface
func (wb *AlpitronicHYC) Identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegVID), 4)
	if err != nil {
		return "", err
	}

	if !allZero(b) {
		return hex.EncodeToString(b), nil
	}

	b, err = wb.conn.ReadInputRegisters(wb.reg(hycRegIdTag), 10)
	if err != nil {
		return "", err
	}

	if !allZero(b) {
		return hex.EncodeToString(b), nil
	}

	return "", nil
}

var _ api.Battery = (*AlpitronicHYC)(nil)

// Soc implements the api.Battery interface
func (wb *AlpitronicHYC) Soc() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(wb.reg(hycRegSoC), 1)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint16(b)) / 100, nil
}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}
