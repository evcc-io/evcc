package charger

// LICENSE

// Copyright (c) 2019-2021 andig

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
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	innogyRegID           = 0    // Input
	innogyRegSerial       = 25   // Input
	innogyRegProtocol     = 50   // Input
	innogyRegManufacturer = 100  // Input
	innogyRegFirmware     = 200  // Input
	innogyRegStatus       = 275  // Input
	innogyRegEnable       = 1028 // Holding
)

var (
	innogyRegMaxCurrents = []uint16{1012, 1014, 1016} // max current per phase
	innogyRegCurrents    = []uint16{1006, 1008, 1010} // current readings per phase
)

// Innogy is an api.Charger implementation for Innogy eBox wallboxes.
type Innogy struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("innogy", NewInnogyFromConfig)
}

// NewInnogyFromConfig creates a Innogy charger from generic config
func NewInnogyFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI string
		ID  uint8
	}{
		ID: 1,
	}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewInnogy(cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	return wb, nil
}

// NewInnogy creates a Innogy charger
func NewInnogy(uri string, id uint8) (*Innogy, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.TcpFormat, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("innogy")
	conn.Logger(log.TRACE)

	wb := &Innogy{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Innogy) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(innogyRegStatus, 2)
	if err != nil {
		return api.StatusNone, err
	}

	switch r := rune(b[0]); r {
	case 'A', 'B', 'D', 'E', 'F':
		return api.ChargeStatus(r), nil
	case 'C':
		// C1 is "connected"
		if rune(b[1]) == '1' {
			return api.StatusB, nil
		}
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %0x", b[:1])
	}
}

// Enabled implements the api.Charger interface
func (wb *Innogy) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(innogyRegEnable, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Innogy) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(innogyRegEnable, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Innogy) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Innogy)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Innogy) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(float32(current)))

	for _, reg := range innogyRegMaxCurrents {
		if _, err := wb.conn.WriteMultipleRegisters(reg, 2, b); err != nil {
			return err
		}
	}

	return nil
}

var _ api.MeterCurrent = (*Innogy)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Innogy) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range innogyRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(math.Float32frombits(binary.BigEndian.Uint32(b))))
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Diagnosis = (*Innogy)(nil)

// Diagnose implements the Diagnosis interface
func (wb *Innogy) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(innogyRegID, 25); err == nil {
		fmt.Printf("Id:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(innogyRegSerial, 25); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(innogyRegManufacturer, 25); err == nil {
		fmt.Printf("Manufacturer:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(innogyRegProtocol, 25); err == nil {
		fmt.Printf("Protocol:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(innogyRegFirmware, 25); err == nil {
		fmt.Printf("Firmware:\t%s\n", b)
	}
}
