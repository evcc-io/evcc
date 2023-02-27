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
	hesotecRegPause    = 0x1002 //	RW	B_Pause			1	-	u16
	hesotecRegCurrent  = 0x1003 //	RW	I_Strom_Max_Last	1	A	u16
	hesotecRegVoltages = 0x4001 //	R	N_Spannung_1		1	V	u16
	hesotecRegCurrents = 0x4004 //	R	N_Strom_1		2	mA	u32
	hesotecRegPower    = 0x400A //	R	N_Wirkleistung		2	mW	u32
	hesotecRegCurrCP   = 0x4016 //	R	I_Strom_CP		2	mA	u32
	hesotecRegStatus   = 0x4018 //	R	E_Status_CP		1	-	ASCII
	hesotecRegDuration = 0x401A //	R	N_Dauer_Ladesitzung	2	s	u32
	hesotecRegEnergy   = 0x401C //	R	N_Energie_Ladesitzung	2	Wh	u32
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

func (wb *Hesotec) setCurrent(current uint16) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, current)

	_, err := wb.conn.WriteMultipleRegisters(hesotecRegCurrent, 1, b)

	return err
}

// Status implements the api.Charger interface
func (wb *Hesotec) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := string(b[0]); s {
	case "A", "B", "C":
		return api.ChargeStatus(s), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %s", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Hesotec) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(hesotecRegCurrent, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) > 0, nil
}

// Enable implements the api.Charger interface
func (wb *Hesotec) Enable(enable bool) error {
	var current uint16
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// MaxCurrent implements the api.Charger interface
func (wb *Hesotec) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	wb.curr = uint16(current)

	return wb.setCurrent(wb.curr)
}
