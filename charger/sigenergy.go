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

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Sigenergy charger implementation
type Sigenergy struct {
	log     *util.Logger
	conn    *modbus.Connection
	enabled bool
}

const (
	regSigSystemState         = 32000 // System states according to IEC61851-1 definition
	regSigTotalEnergyConsumed = 32001 // kWh*100, total energy consumed during charging
	regSigChargingPower       = 32003 // W, instantaneous charging power
	regSigStartStop           = 42000 // Start/Stop charger (0: Start 1: Stop), WO
	regSigOutputCurrent       = 42001 // Amperes, R/W, charger output current ([6, X] X is the smaller value between the rated current and the AC-Charger input breaker rated current.)
)

func init() {
	registry.AddCtx("sigenergy", NewSigenergyFromConfig)
}

// NewSigenergyFromConfig creates a new Sigenergy ModbusTCP charger
func NewSigenergyFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSigenergy(ctx, cc.URI, cc.ID)
}

// NewSigenergy creates a new charger
func NewSigenergy(ctx context.Context, uri string, slaveID uint8) (*Sigenergy, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("sigenergy")
	conn.Logger(log.TRACE)

	wb := &Sigenergy{
		log:  log,
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Sigenergy) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigSystemState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch state := binary.BigEndian.Uint16(b); state {
	case 0x01: // A1/A2
		return api.StatusA, nil
	case 0x02: // B1
		wb.enabled = false // B1 indicates the charger is not enabled
		return api.StatusB, nil
	case 0x03: // B2
		wb.enabled = true // B2 indicates the charger is enabled
		return api.StatusB, nil
	case 0x04: // C1
		wb.enabled = false // C1 indicates the charger is not enabled
		return api.StatusC, nil
	case 0x05: // C2
		wb.enabled = true // C2 indicates the charger is enabled
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (wb *Sigenergy) Enabled() (bool, error) {
	return wb.enabled, nil
}

// Enable implements the api.Charger interface
func (wb *Sigenergy) Enable(enable bool) error {
	var s uint16
	if !enable {
		s = 1
	}

	_, err := wb.conn.WriteSingleRegister(regSigStartStop, s)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Sigenergy) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Sigenergy)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Sigenergy) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	b := make([]byte, 4)

	binary.BigEndian.PutUint32(b, uint32(current*100)) // Convert to mA (100x for 0.01 A resolution)

	_, err := wb.conn.WriteMultipleRegisters(regSigOutputCurrent, 2, b)

	return err
}

var _ api.Meter = (*Sigenergy)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Sigenergy) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigChargingPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.MeterEnergy = (*Sigenergy)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Sigenergy) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(regSigTotalEnergyConsumed, 2)
	if err != nil {
		return 0, err
	}

	// U32 register with gain 100, divide by 100 to get kWh
	return float64(binary.BigEndian.Uint32(b)) / 100, nil
}

var _ api.Diagnosis = (*Sigenergy)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Sigenergy) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(regSigSystemState, 1); err == nil {
		state := binary.BigEndian.Uint16(b)
		stateNames := []string{"System init", "A1/A2", "B1", "B2", "C1", "C2", "F", "E"}
		stateName := "Unknown"
		if int(state) < len(stateNames) {
			stateName = stateNames[state]
		}
		fmt.Printf("\tSystem State:\t%d (%s)\n", state, stateName)
	}

	if b, err := wb.conn.ReadHoldingRegisters(regSigOutputCurrent, 2); err == nil {
		current := float64(binary.BigEndian.Uint32(b)) / 100
		fmt.Printf("\tOutput Current:\t%.2f A\n", current)
	}
}
