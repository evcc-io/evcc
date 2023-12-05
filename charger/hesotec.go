package charger

// LICENSE

// Copyright (c) 2023 premultiply

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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Hesotec charger implementation
type Hesotec struct {
	conn *modbus.Connection
	curr uint16
}

const (
	hesotecRegFirmware1 = 0x0010 // R	Firmware Version eSat Control	3	-	ASCII
	hesotecRegFirmware2 = 0x0013 // R	Firmware Version eSat Power	3	-	ASCII
	hesotecRegFirmware3 = 0x0016 // R	Firmware Version eSat Oak	3	-	ASCII
	hesotecRegFirmware4 = 0x0019 // R	Firmware Version eSat Display	3	-	ASCII
	hesotecRegFirmware5 = 0x001C // R	Firmware Version ESP	3	-	ASCII
	hesotecRegSessAuth  = 0x1000 // RW	B_Autorisierung	1	-	u16
	hesotecRegSessStop  = 0x1001 // RW	B_Stop_RFID	1	-	u16
	hesotecRegSessPause = 0x1002 //	RW	B_Pause			1	-	u16
	hesotecRegCurrent   = 0x1003 //	RW	I_Strom_Max_Last	1	A	u16
	hesotecRegTemp      = 0x4000 //	R	N_Temperatur	1	°C	i16
	hesotecRegVoltages  = 0x4001 //	R	N_Spannung_1		1	V	u16
	hesotecRegCurrents  = 0x4004 //	R	N_Strom_1		2	mA	u32
	hesotecRegPower     = 0x400A //	R	N_Wirkleistung		2	mW	u32
	hesotecRegCurrCP    = 0x4016 //	R	I_Strom_CP		2	mA	u32
	hesotecRegStatus    = 0x4018 //	R	E_Status_CP		1	-	ASCII
	hesotecRegDuration  = 0x401A //	R	N_Dauer_Ladesitzung	2	s	u32
	hesotecRegEnergy    = 0x401C //	R	N_Energie_Ladesitzung	2	Wh	u32
)

func init() {
	registry.Add("hesotec", NewHesotecFromConfig)
}

// NewHesotecFromConfig creates a Hesotec charger from generic config
func NewHesotecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHesotec(cc.URI, cc.ID)
}

// NewHesotec creates Hesotec charger
func NewHesotec(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("hesotec")
	conn.Logger(log.TRACE)

	wb := &Hesotec{
		conn: conn,
		curr: 6, // assume min current
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Hesotec) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}
	return api.ChargeStatusString(string(b[0]))
}

// Enabled implements the api.Charger interface
func (wb *Hesotec) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegSessPause, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 0, nil
}

// Enable implements the api.Charger interface
func (wb *Hesotec) Enable(enable bool) error {
	b := make([]byte, 2)
	if !enable {
		binary.BigEndian.PutUint16(b, 1)
	}

	_, err := wb.conn.WriteMultipleRegisters(hesotecRegSessPause, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Hesotec) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	wb.curr = uint16(current)

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, wb.curr)

	_, err := wb.conn.WriteMultipleRegisters(hesotecRegCurrent, 1, b)

	return err
}

var _ api.Meter = (*Hesotec)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Hesotec) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.ChargeTimer = (*Hesotec)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *Hesotec) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.MeterEnergy = (*Hesotec)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Hesotec) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

var _ api.PhaseCurrents = (*Hesotec)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Hesotec) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegCurrents, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / 1e3
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseVoltages = (*Hesotec)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Hesotec) Voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegVoltages, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		res[i] = float64(binary.BigEndian.Uint16(b[2*i:]))
	}

	return res[0], res[1], res[2], nil
}

var _ api.Diagnosis = (*Hesotec)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Hesotec) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegFirmware1, 3); err == nil {
		fmt.Printf("\tFirmware Version eSat Control:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegFirmware2, 3); err == nil {
		fmt.Printf("\tFirmware Version eSat Power:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegFirmware3, 3); err == nil {
		fmt.Printf("\tFirmware Version eSat Oak:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegFirmware4, 3); err == nil {
		fmt.Printf("\tFirmware Version eSat Display:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegFirmware5, 3); err == nil {
		fmt.Printf("\tFirmware Version ESP:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegSessAuth, 1); err == nil {
		fmt.Printf("\tB_Autorisierung:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegSessStop, 1); err == nil {
		fmt.Printf("\tB_Stop_RFID:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegSessPause, 1); err == nil {
		fmt.Printf("\tB_Pause:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegCurrent, 1); err == nil {
		fmt.Printf("\tI_Strom_Max_Last:\t%dA\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegTemp, 1); err == nil {
		fmt.Printf("\tN_Temperatur:\t%d°C\n", int16(binary.BigEndian.Uint16(b)))
	}
	if b, err := wb.conn.ReadHoldingRegisters(hesotecRegCurrCP, 2); err == nil {
		fmt.Printf("\tI_Strom_CP:\t%dmA\n", binary.BigEndian.Uint32(b))
	}
}
