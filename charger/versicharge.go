package charger

// LICENSE

// Copyright (c) 2023 premultiply
// Initial implementation and testing by achgut, Flo56958

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
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	versiRegBrand          = 0    //  5 RO ASCII
	versiRegProductionDate = 5    //  2 RO UNIT16[]
	versiRegSerial         = 7    //  5 RO ASCII
	versiRegModel          = 12   // 10 RO ASCII
	versiRegFirmware       = 31   // 10 RO ASCII
	versiRegModbusTable    = 41   //  1 RO UINT16
	versiRegMeterType      = 30   //  1 RO UINT16
	versiRegErrorCode      = 1600 //  1 RO INT16
	versiRegTemp           = 1602 //  1 RO INT16
	versiRegChargeStatus   = 1601 //  1 RO INT16
	versiRegPause          = 1629 //  1 RW UNIT16
	versiRegMaxCurrent     = 1633 //  1 RW UNIT16
	versiRegCurrents       = 1647 //  3 RO UINT16
	versiRegVoltages       = 1651 //  3 RO UINT16
	versiRegPowers         = 1662 //  3 RO UINT16
	versiRegTotalEnergy    = 1692 //  2 RO UINT32
)

// Versicharge is an api.Charger implementation for Versicharge wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at id 2 (default).

type Versicharge struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("versicharge", NewVersichargeFromConfig)
}

// NewVersichargeFromConfig creates a Versicharge charger from generic config
func NewVersichargeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 2,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVersicharge(cc.URI, cc.ID)
}

// NewVersicharge creates a Versicharge charger
func NewVersicharge(uri string, id uint8) (*Versicharge, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("versicharge")
	conn.Logger(log.TRACE)

	wb := &Versicharge{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Versicharge) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(versiRegChargeStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	switch s {
	case 1: // Available
		return api.StatusA, nil
	case 2, 5: // Preparing, Suspended EV, Suspended EVSE
		return api.StatusB, nil
	case 3, 4: // Charging
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Versicharge) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(versiRegPause, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 2, nil
}

// Enable implements the api.Charger interface
func (wb *Versicharge) Enable(enable bool) error {
	var u uint16 = 1
	if enable {
		u = 2
	}

	_, err := wb.conn.WriteSingleRegister(versiRegPause, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Versicharge) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(versiRegMaxCurrent, uint16(current))

	return err
}

var _ api.Meter = (*Versicharge)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Versicharge) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(versiRegPowers, 3)
	if err != nil {
		return 0, err
	}

	var sum float64
	for i := 0; i < 3; i++ {
		sum += float64(binary.BigEndian.Uint16(b[2*i:]))
	}

	return sum, nil
}

var _ api.MeterEnergy = (*Versicharge)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Versicharge) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(versiRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e4, err
}

// getPhaseValues returns 3 sequential register values
func (wb *Versicharge) getPhaseValues(reg uint16) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < 3; i++ {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:]))
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Versicharge)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Versicharge) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(versiRegCurrents)
}

var _ api.PhaseVoltages = (*Versicharge)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Versicharge) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(versiRegVoltages)
}

var _ api.Diagnosis = (*Versicharge)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Versicharge) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(versiRegBrand, 5); err == nil {
		fmt.Printf("Brand:\t\t\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegModel, 10); err == nil {
		fmt.Printf("Model:\t\t\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegSerial, 5); err == nil {
		fmt.Printf("Serial:\t\t\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegProductionDate, 2); err == nil {
		fmt.Printf("Production Date:\t%d.%d.%d\n", b[3], b[2], binary.BigEndian.Uint16(b[0:2]))
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegFirmware, 10); err == nil {
		fmt.Printf("Firmware:\t\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegModbusTable, 1); err == nil {
		fmt.Printf("Modbus Table:\t\t%d\n", b[1])
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegMeterType, 1); err == nil {
		fmt.Printf("Meter Type:\t\t%d\n", b[1])
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegTemp, 1); err == nil {
		fmt.Printf("Temperature PCB:\t%d°C\n\n", b[1])
	}
	if b, err := wb.conn.ReadHoldingRegisters(versiRegErrorCode, 1); err == nil {
		fmt.Printf("EVSE Error Code:\t%d\n\n", b[1])
	}
}
