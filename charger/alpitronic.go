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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// AlpitronicHYC charger implementation
type AlpitronicHYC struct {
	conn      *modbus.Connection
	inputG    func() ([]byte, error)
	curr      float64
	enabled   bool
	connector uint16
}

// input registers of the charging station (connector 0)
//
//nolint:unused // complete register map as documented by the vendor
const (
	hycRegStationTime           = 0  // UINT32, s
	hycRegStationNumConnectors  = 2  // UINT16
	hycRegStationState          = 3  // UINT16, 0-Available, 8-Unavailable, 10-Faulted
	hycRegStationPowerDrained   = 4  // UINT32, W
	hycRegStationSerial         = 6  // 24 byte string
	hycRegStationLoadManagement = 18 // UINT16, bool
	hycRegStationChargepointId  = 30 // 32 byte string
	hycRegStationVersionMajor   = 46 // UINT16
	hycRegStationVersionMinor   = 47 // UINT16
	hycRegStationVersionPatch   = 48 // UINT16
	hycRegStationVarInductive   = 49 // UINT32, var
	hycRegStationVarCapacitive  = 51 // UINT32, var
)

// input registers, relative to the connector block (1xx..4xx)
const (
	hycRegState              = 0  // UINT16
	hycRegChargingVoltage    = 1  // UINT32, cV
	hycRegChargingCurrent    = 3  // UINT16, cA up to 3.0, A from 3.1.0 on
	hycRegPowerAbsorptionAC  = 4  // UINT32, W- DC charging power up to 3.0, AC absorption from 3.1.0 on
	hycRegChargeTime         = 6  // UINT16, s
	hycRegChargedEnergy      = 7  // UINT16, kWh/100 up to 3.0, kWh/10 from 3.1.0 on
	hycRegSoC                = 8  // UINT16, %/100
	hycRegConnectorType      = 9  // UINT16, 0-ChargePoint, 1-CCS2, 2-CCS1, 3-CHAdeMO, 4-CCS_AC, 5-GBT, 6-MCS, 7-NACS
	hycRegMaxPowerAbsorption = 10 // UINT32, W- includes the limit written by us
	hycRegMinPowerAbsorption = 12 // UINT32, W
	hycRegVID                = 18 // 8 bytes
	hycRegIdTag              = 22 // 20 bytes
	hycRegTotalChargedEnergy = 32 // INT64, Wh

	// read up to the last register in use (x32..x35). x14/x16 (VAR) are covered but
	// no longer documented since 3.1.0- to be verified against the device
	hycInputLength = 36
)

// holding registers, relative to the connector block (0xx is the station)
const (
	hycRegMaxPowerAC       = 0 // UINT32, W per connector, VA for the station
	hycRegSetReactivePower = 2 // INT32, var
)

// connector states as per register x00
const (
	hycStateAvailable uint16 = iota
	hycStatePreparingTagIdReady
	hycStatePreparingEVReady
	hycStateCharging
	hycStateSuspendedEV
	hycStateSuspendedEVSE
	hycStateFinishing
	hycStateReserved
	hycStateUnavailable
	hycStateUnavailableFwUpdate
	hycStateFaulted
	hycStateUnavailableConnObj
)

// a zero power limit makes the station report the connector as unavailable and
// abort a running session, so charging is inhibited with a limit that is too low
// for any connector to actually charge- see hycRegMinPowerAbsorption
const (
	hycPowerPerAmp = 230 * 3 // W/A on the AC side
	hycMinPowerAC  = 1547    // W
	hycMinCurrent  = 2.25    // A, lowest current exceeding hycMinPowerAC
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

	return NewAlpitronicHYC(ctx, cc.TcpSettings, cc.Connector)
}

// NewAlpitronicHYC creates Alpitronic charger
func NewAlpitronicHYC(ctx context.Context, settings modbus.TcpSettings, connector uint16) (*AlpitronicHYC, error) {
	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("alpitronic")
	conn.Logger(log.TRACE)

	return newAlpitronicHYC(conn, connector)
}

// newAlpitronicHYC wires the struct without sponsor gate (also used by tests)
func newAlpitronicHYC(conn *modbus.Connection, connector uint16) (*AlpitronicHYC, error) {
	wb := &AlpitronicHYC{
		conn:      conn,
		curr:      hycMinCurrent,
		connector: connector,
	}

	// share a single bulk read of the connector's input block across all decoders.
	// the station applies a fallback when it is not polled within GridFallbackTimeout,
	// so keeping the number of roundtrips low matters here.
	wb.inputG = util.Cached(func() ([]byte, error) {
		return wb.conn.ReadInputRegisters(wb.reg(hycRegState), hycInputLength)
	}, time.Second)

	// seed the current limit from the charger- this also validates the connector
	b, err := wb.conn.ReadHoldingRegisters(wb.reg(hycRegMaxPowerAC), 2)
	if err != nil {
		return nil, err
	}

	if power := encoding.Uint32(b); power > hycMinPowerAC {
		wb.curr = float64(power) / hycPowerPerAmp
		wb.enabled = true
	}

	return wb, nil
}

// hycInput returns the bytes of the given register within the cached input block
func hycInput(b []byte, reg uint16, n int) []byte {
	off := 2 * int(reg)
	return b[off : off+n]
}

// hycPower converts a charging current into the AC side power limit
func hycPower(current float64) uint32 {
	if current <= 0 {
		return 0
	}

	return uint32(current * hycPowerPerAmp)
}

// setPower writes the connector's power limit
func (wb *AlpitronicHYC) setPower(power uint32) error {
	b := make([]byte, 4)
	encoding.PutUint32(b, power)

	_, err := wb.conn.WriteMultipleRegisters(wb.reg(hycRegMaxPowerAC), 2, b)
	if err == nil {
		// Status is polled before Enabled, so track our own writes to keep
		// the charging state from lagging a cycle behind
		wb.enabled = power > hycMinPowerAC
	}

	return err
}

// reg returns the register address for the connector
func (wb *AlpitronicHYC) reg(reg uint16) uint16 {
	return (wb.connector * 100) + reg
}

// state returns the connector state
func (wb *AlpitronicHYC) state() (uint16, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return encoding.Uint16(hycInput(b, hycRegState, 2)), nil
}

// Status implements the api.Charger interface
func (wb *AlpitronicHYC) Status() (api.ChargeStatus, error) {
	s, err := wb.state()
	if err != nil {
		return api.StatusNone, err
	}

	switch s {
	case
		hycStateAvailable,
		hycStatePreparingTagIdReady, // authorized, waiting for the cable to be plugged in
		hycStateReserved,
		hycStateUnavailable,
		hycStateUnavailableFwUpdate,
		hycStateUnavailableConnObj:
		return api.StatusA, nil
	case
		hycStatePreparingEVReady, // plugged in, waiting for authorization
		hycStateSuspendedEV,
		hycStateSuspendedEVSE,
		hycStateFinishing:
		return api.StatusB, nil
	case hycStateCharging:
		if !wb.enabled {
			return api.StatusB, nil
		}
		return api.StatusC, nil
	case hycStateFaulted:
		return api.StatusNone, errors.New("connector state: faulted")
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

var _ api.StatusReasoner = (*AlpitronicHYC)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *AlpitronicHYC) StatusReason() (api.Reason, error) {
	s, err := wb.state()
	if err != nil {
		return api.ReasonUnknown, err
	}

	switch s {
	case hycStatePreparingEVReady:
		return api.ReasonWaitingForAuthorization, nil
	case hycStateFinishing:
		return api.ReasonDisconnectRequired, nil
	default:
		return api.ReasonUnknown, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *AlpitronicHYC) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.reg(hycRegMaxPowerAC), 2)
	if err != nil {
		return false, err
	}

	wb.enabled = encoding.Uint32(b) > hycMinPowerAC

	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *AlpitronicHYC) Enable(enable bool) error {
	power := uint32(hycMinPowerAC)
	if enable {
		power = hycPower(wb.curr)
	}

	return wb.setPower(power)
}

// MaxCurrent implements the api.Charger interface
func (wb *AlpitronicHYC) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*AlpitronicHYC)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
//
// api.CurrentLimiter is deliberately not implemented: registers x10/x36 contain
// the connector power limit written by evcc itself, so using them as upper bound
// would feed back into the limit. Use the loadpoint's maxcurrent instead.
func (wb *AlpitronicHYC) MaxCurrentMillis(current float64) error {
	if current < hycMinCurrent {
		return fmt.Errorf("invalid current %.1f", current)
	}

	err := wb.setPower(hycPower(current))
	if err == nil {
		wb.curr = current
	}

	return err
}

var _ api.Meter = (*AlpitronicHYC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *AlpitronicHYC) CurrentPower() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint32(hycInput(b, hycRegPowerAbsorptionAC, 4))), nil
}

var _ api.ChargeTimer = (*AlpitronicHYC)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *AlpitronicHYC) ChargeDuration() (time.Duration, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return time.Duration(encoding.Uint16(hycInput(b, hycRegChargeTime, 2))) * time.Second, nil
}

// api.ChargeRater is not implemented: hycRegChargedEnergy is scaled kWh/100 up to
// firmware 3.0 and kWh/10 from 3.1.0 on, so the session energy is derived from
// TotalEnergy instead

var _ api.MeterEnergy = (*AlpitronicHYC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *AlpitronicHYC) TotalEnergy() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int64(hycInput(b, hycRegTotalChargedEnergy, 8))) / 1e3, nil
}

var _ api.Identifier = (*AlpitronicHYC)(nil)

// Identify implements the api.Identifier interface
func (wb *AlpitronicHYC) Identify() ([]string, error) {
	b, err := wb.inputG()
	if err != nil {
		return nil, err
	}

	var res []string

	// the idTag register holds the rfid uid as null-padded ascii string
	if idTag := hycInput(b, hycRegIdTag, 20); !allZero(idTag) {
		res = append(res, strings.ToLower(strings.TrimRight(string(idTag), "\x00")))
	}

	if vid := hycInput(b, hycRegVID, 8); !allZero(vid) {
		res = append(res, hex.EncodeToString(vid))
	}

	return res, nil
}

var _ api.Battery = (*AlpitronicHYC)(nil)

// Soc implements the api.Battery interface
func (wb *AlpitronicHYC) Soc() (float64, error) {
	b, err := wb.inputG()
	if err != nil {
		return 0, err
	}

	return float64(encoding.Uint16(hycInput(b, hycRegSoC, 2))) / 100, nil
}

func allZero(s []byte) bool {
	for _, v := range s {
		if v != 0 {
			return false
		}
	}
	return true
}
