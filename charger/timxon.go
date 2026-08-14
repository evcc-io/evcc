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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Timxon charger implementation
type Timxon struct {
	log       *util.Logger
	conn      *modbus.Connection
	base      uint16
	connector int
}

const (
	timxonRegVersion        = 0x0000 // string[16] RO
	timxonRegSerial         = 0x0010 // string[16] RO
	timxonRegConnectorState = 0x0022 // Uint16 RO ENUM
	timxonRegCableState     = 0x0024 // Uint16 RO ENUM
	timxonRegErrorCode      = 0x0026 // Uint16 RO BITMASK
	timxonRegCPState        = 0x0028 // Uint16 RO ENUM
	timxonRegCPValue        = 0x002A // Uint16 RO 0.1V
	timxonRegPPState        = 0x002C // Uint16 RO ENUM
	timxonRegCurrentLimit   = 0x0032 // Uint16 WR 1A
	timxonRegChargeCommand  = 0x0034 // Uint16 WR ENUM

	timxonCon1Base = 0x0064
	timxonCon2Base = 0x00C8

	timxonRegConCurrents       = 0  // Uint16 RO 0.1A
	timxonRegConVoltages       = 3  // Uint16 RO 0.1V
	timxonRegConPowers         = 6  // Uint32 RO 1W
	timxonRegConPower          = 12 // Uint32 RO 1W
	timxonRegConEnergy         = 14 // Uint32 RO 0.1kWh
	timxonRegConChargedEnergy  = 16 // Uint16 RO 0.1kWh
	timxonRegConChargeDuration = 17 // Uint32 RO 1s
	timxonRegConMaxCurrent     = 19 // Uint16 RO 0.1A
	timxonRegConMinCurrent     = 20 // Uint16 RO 0.1A
	timxonRegConUID            = 21 // string[16] RO
)

func init() {
	registry.AddCtx("timxon", NewTimxonFromConfig)
}

// NewTimxonFromConfig creates a TIMXON charger from generic config
func NewTimxonFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		Connector       int
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
		Connector: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewTimxon(ctx, cc.Settings, cc.Connector)
}

// NewTimxon creates a TIMXON charger
func NewTimxon(ctx context.Context, settings modbus.Settings, connector int) (api.Charger, error) {
	if connector < 1 || connector > 2 {
		return nil, fmt.Errorf("invalid connector %d, must be 1 or 2", connector)
	}

	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("timxon")
	conn.Logger(log.TRACE)

	base := uint16(timxonCon1Base)
	if connector == 2 {
		base = timxonCon2Base
	}

	return &Timxon{
		log:       log,
		conn:      conn,
		base:      base,
		connector: connector,
	}, nil
}

func (wb *Timxon) connectorReg(reg uint16) uint16 {
	return reg + uint16(wb.connector-1)
}

func (wb *Timxon) readUint16(reg uint16) (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

func (wb *Timxon) readUint32(reg uint16) (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 2)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(b), nil
}

func (wb *Timxon) writeUint16(reg uint16, value uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, value)

	_, err := wb.conn.WriteMultipleRegisters(reg, 1, b)

	return err
}

func (wb *Timxon) readString(reg, length uint16) (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, length)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

// Status implements the api.Charger interface
func (wb *Timxon) Status() (api.ChargeStatus, error) {
	state, err := wb.readUint16(wb.connectorReg(timxonRegConnectorState))
	if err != nil {
		return api.StatusNone, err
	}

	switch state {
	case
		0, // Available
		7, // Unavailable
		8: // Faulted
		return api.StatusA, nil
	case
		1, // Preparing
		3, // SuspendedEVSE
		4, // SuspendedEV
		5, // Finishing
		6: // Reserved
		return api.StatusB, nil
	case 2: // Charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (wb *Timxon) Enabled() (bool, error) {
	command, err := wb.readUint16(wb.connectorReg(timxonRegChargeCommand))
	if err != nil {
		return false, err
	}

	return command == 1, nil
}

// Enable implements the api.Charger interface
func (wb *Timxon) Enable(enable bool) error {
	var command uint16 = 2 // Stop
	if enable {
		command = 1 // Start
	}

	return wb.writeUint16(wb.connectorReg(timxonRegChargeCommand), command)
}

// MaxCurrent implements the api.Charger interface
func (wb *Timxon) MaxCurrent(current int64) error {
	if current < 0 || current > 0xffff {
		return fmt.Errorf("invalid current: %d", current)
	}

	return wb.writeUint16(wb.connectorReg(timxonRegCurrentLimit), uint16(current))
}

var _ api.CurrentGetter = (*Timxon)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *Timxon) GetMaxCurrent() (float64, error) {
	current, err := wb.readUint16(wb.connectorReg(timxonRegCurrentLimit))
	if err != nil {
		return 0, err
	}

	return float64(current), nil
}

var _ api.Meter = (*Timxon)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Timxon) CurrentPower() (float64, error) {
	power, err := wb.readUint32(wb.base + timxonRegConPower)
	if err != nil {
		return 0, err
	}

	return float64(power), nil
}

var _ api.MeterEnergy = (*Timxon)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Timxon) TotalEnergy() (float64, error) {
	energy, err := wb.readUint32(wb.base + timxonRegConEnergy)
	if err != nil {
		return 0, err
	}

	return float64(energy) / 10, nil
}

var _ api.ChargeRater = (*Timxon)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Timxon) ChargedEnergy() (float64, error) {
	energy, err := wb.readUint16(wb.base + timxonRegConChargedEnergy)
	if err != nil {
		return 0, err
	}

	return float64(energy) / 10, nil
}

func (wb *Timxon) readPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:])) / 10
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Timxon)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Timxon) Currents() (float64, float64, float64, error) {
	return wb.readPhaseValues(wb.base + timxonRegConCurrents)
}

var _ api.PhaseVoltages = (*Timxon)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Timxon) Voltages() (float64, float64, float64, error) {
	return wb.readPhaseValues(wb.base + timxonRegConVoltages)
}

var _ api.PhasePowers = (*Timxon)(nil)

// Powers implements the api.PhasePowers interface
func (wb *Timxon) Powers() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.base+timxonRegConPowers, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:]))
	}

	return res[0], res[1], res[2], nil
}

var _ api.ChargeTimer = (*Timxon)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Timxon) ChargeDuration() (time.Duration, error) {
	duration, err := wb.readUint32(wb.base + timxonRegConChargeDuration)
	if err != nil {
		return 0, err
	}

	return time.Duration(duration) * time.Second, nil
}

var _ api.Identifier = (*Timxon)(nil)

// Identify implements the api.Identifier interface
func (wb *Timxon) Identify() (string, error) {
	return wb.readString(wb.base+timxonRegConUID, 16)
}

var _ api.Diagnosis = (*Timxon)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Timxon) Diagnose() {
	fmt.Printf("\tConnector:\t%d\n", wb.connector)
	if s, err := wb.readString(timxonRegVersion, 16); err == nil {
		fmt.Printf("\tVersion:\t%s\n", s)
	}
	if s, err := wb.readString(timxonRegSerial, 16); err == nil {
		fmt.Printf("\tSerial:\t%s\n", s)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegConnectorState)); err == nil {
		fmt.Printf("\tConnector State:\t%d\n", u)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegCableState)); err == nil {
		fmt.Printf("\tCable State:\t%d\n", u)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegErrorCode)); err == nil {
		fmt.Printf("\tError Code:\t%d\n", u)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegCPState)); err == nil {
		fmt.Printf("\tCP State:\t%d\n", u)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegCPValue)); err == nil {
		fmt.Printf("\tCP Value:\t%.1fV\n", float64(u)/10)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegPPState)); err == nil {
		fmt.Printf("\tPP State:\t%d\n", u)
	}
	if u, err := wb.readUint16(wb.connectorReg(timxonRegCurrentLimit)); err == nil {
		fmt.Printf("\tCurrent Limit:\t%dA\n", u)
	}
	if u, err := wb.readUint16(wb.base + timxonRegConMaxCurrent); err == nil {
		fmt.Printf("\tDefault Max Current:\t%.1fA\n", float64(u)/10)
	}
	if u, err := wb.readUint16(wb.base + timxonRegConMinCurrent); err == nil {
		fmt.Printf("\tDefault Min Current:\t%.1fA\n", float64(u)/10)
	}
	if s, err := wb.readString(wb.base+timxonRegConUID, 16); err == nil {
		fmt.Printf("\tUID:\t%s\n", s)
	}
}
