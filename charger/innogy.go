package charger

// LICENSE

// Copyright (c) 2019-2022 andig

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
	"github.com/evcc-io/evcc/util/sponsor"
)

const (
	igyRegID           = 0   // Input
	igyRegSerial       = 25  // Input
	igyRegProtocol     = 50  // Input
	igyRegManufacturer = 100 // Input
	igyRegFirmware     = 200 // Input
	igyRegStatus       = 275 // Input
)

var (
	igyRegMaxCurrents = []uint16{1012, 1014, 1016} // max current per phase
	igyRegCurrents    = []uint16{1006, 1008, 1010} // current readings per phase
)

// https://www.innogy-emobility.com/content/dam/revu-global/emobility-solutions/neue-website-feb-2021/downloadcenter/digital-services/eld_instman_modbustcpde.pdf

// Innogy is an api.Charger implementation for Innogy eBox wallboxes.
type Innogy struct {
	conn    *modbus.Connection
	current float64
}

func init() {
	registry.Add("innogy", NewInnogyFromConfig)
}

// NewInnogyFromConfig creates a Innogy charger from generic config
func NewInnogyFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewInnogy(cc.URI, cc.ID)
}

// NewInnogy creates a Innogy charger
func NewInnogy(uri string, id uint8) (*Innogy, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("innogy")
	conn.Logger(log.TRACE)

	wb := &Innogy{
		conn:    conn,
		current: 6,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Innogy) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(igyRegStatus, 2)
	if err != nil {
		return api.StatusNone, err
	}

	switch r := rune(b[0]); r {
	case 'A', 'B', 'D', 'E', 'F':
		return api.ChargeStatusString(string(r))
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
	b, err := wb.conn.ReadHoldingRegisters(igyRegMaxCurrents[0], 2)
	if err != nil {
		return false, err
	}

	return math.Float32frombits(binary.BigEndian.Uint32(b)) >= 6, nil
}

// Enable implements the api.Charger interface
func (wb *Innogy) Enable(enable bool) error {
	var current float64
	if enable {
		current = wb.current
	}

	return wb.setCurrent(current)
}

func (wb *Innogy) setCurrent(current float64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, math.Float32bits(float32(current)))

	for _, reg := range igyRegMaxCurrents {
		if _, err := wb.conn.WriteMultipleRegisters(reg, 2, b); err != nil {
			return err
		}
	}

	return nil
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

	err := wb.setCurrent(current)
	if err == nil {
		wb.current = current
	}

	return err
}

var _ api.Meter = (*Innogy)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Innogy) CurrentPower() (float64, error) {
	// https://github.com/evcc-io/evcc/issues/6848
	if status, err := wb.Status(); status != api.StatusC || err != nil {
		return 0, err
	}
	l1, l2, l3, err := wb.Currents()
	return 230 * (l1 + l2 + l3), err
}

var _ api.PhaseCurrents = (*Innogy)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Innogy) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range igyRegCurrents {
		b, err := wb.conn.ReadInputRegisters(regCurrent, 2)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(math.Float32frombits(binary.BigEndian.Uint32(b))))
	}

	return currents[0], currents[1], currents[2], nil
}

var _ api.Diagnosis = (*Innogy)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Innogy) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(igyRegManufacturer, 25); err == nil {
		fmt.Printf("Manufacturer:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(igyRegID, 25); err == nil {
		fmt.Printf("Id:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(igyRegSerial, 25); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(igyRegProtocol, 25); err == nil {
		fmt.Printf("Protocol:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(igyRegFirmware, 25); err == nil {
		fmt.Printf("Firmware:\t%s\n", b)
	}
}
