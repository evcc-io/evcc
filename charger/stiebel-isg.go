package charger

// LICENSE

// Copyright (c) 2022 premultiply, andig

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
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/stiebel"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// StiebelIsg charger implementation
type StiebelIsg struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("stiebel-isg", NewStiebelIsgFromConfig)
}

// NewStiebelIsgFromConfig creates a Stiebel ISG charger from generic config
func NewStiebelIsgFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewStiebelIsg(cc.URI, cc.ID)
}

// NewStiebelIsg creates Stiebel ISG charger
func NewStiebelIsg(uri string, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("stiebel")
	conn.Logger(log.TRACE)

	wb := &StiebelIsg{
		conn: conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *StiebelIsg) Status() (api.ChargeStatus, error) {
	return api.StatusNone, api.ErrNotAvailable
}

// Enabled implements the api.Charger interface
func (wb *StiebelIsg) Enabled() (bool, error) {
	return false, api.ErrNotAvailable
}

// Enable implements the api.Charger interface
func (wb *StiebelIsg) Enable(enable bool) error {
	return api.ErrNotAvailable
}

// MaxCurrent implements the api.Charger interface
func (wb *StiebelIsg) MaxCurrent(current int64) error {
	return api.ErrNotAvailable
}

var _ api.Diagnosis = (*StiebelIsg)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *StiebelIsg) Diagnose() {
	for _, reg := range stiebel.Block1 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr, 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block2 {
		if b, err := wb.conn.ReadHoldingRegisters(reg.Addr, 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block3 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr, 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block4 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr, 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block5 {
		if b, err := wb.conn.ReadHoldingRegisters(reg.Addr, 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block6 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr, 1); err == nil {
			wb.print(reg, b)
		}
	}
}

func (wb *StiebelIsg) print(reg stiebel.Register, b []byte) {
	name := reg.Name
	if reg.Comment != "" {
		name = fmt.Sprintf("%s (%s)", name, reg.Comment)
	}

	switch reg.Typ {
	case stiebel.Bits:
		if stiebel.Invalid(b) {
			return
		}
		fmt.Printf("\t%d %s:\t%04X\n", reg.Addr, name, b)

	default:
		f := reg.Float(b)
		if math.IsNaN(f) {
			return
		}

		fmt.Printf("\t%d %s:\t%.1f%s\n", reg.Addr, name, f, reg.Unit)
	}
}
