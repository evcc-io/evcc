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
	"errors"
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/stiebel"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// StiebelIsg charger implementation
type StiebelIsg struct {
	*embed
	log  *util.Logger
	conn *modbus.Connection
	lp   loadpoint.API
	conf stiebel.TempConfig
}

func init() {
	registry.Add("stiebel-isg", NewStiebelIsgFromConfig)
}

// NewStiebelIsgFromConfig creates a Stiebel ISG charger from generic config
func NewStiebelIsgFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		stiebel.TempConfig `mapstructure:",squash"`
		embed              `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
		TempConfig: stiebel.TempConfig{
			// temp
			SollAddr: 522, // WW
			IstAddr:  521, // WW
			// enable/disable
			ModeAddr:    1500, // Betriebsart
			EnableMode:  3,    // Komfortbetrieb
			DisableMode: 2,    // Programmbetrieb
		},
		embed: embed{
			Icon_:     "heatpump",
			Features_: []api.Feature{api.IntegratedDevice},
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewStiebelIsg(&cc.embed, cc.URI, cc.ID, cc.TempConfig)
}

// NewStiebelIsg creates Stiebel ISG charger
func NewStiebelIsg(embed *embed, uri string, slaveID uint8, conf stiebel.TempConfig) (api.Charger, error) {
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
		embed: embed,
		log:   log,
		conn:  conn,
		conf:  conf,
	}

	return wb, nil
}

func (wb *StiebelIsg) mode() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(wb.conf.ModeAddr, 1)
	if err != nil {
		return 0, err
	}

	return encoding.Uint16(b), nil
}

// Status implements the api.Charger interface
func (wb *StiebelIsg) Status() (api.ChargeStatus, error) {
	mode, err := wb.mode()
	if err != nil {
		return api.StatusNone, err
	}

	status := map[uint16]api.ChargeStatus{wb.conf.EnableMode: api.StatusC, wb.conf.DisableMode: api.StatusB}[mode]
	return status, nil
}

// Enabled implements the api.Charger interface
func (wb *StiebelIsg) Enabled() (bool, error) {
	mode, err := wb.mode()
	return mode == wb.conf.EnableMode, err
}

// Enable implements the api.Charger interface
func (wb *StiebelIsg) Enable(enable bool) error {
	mode := map[bool]uint16{true: wb.conf.EnableMode, false: wb.conf.DisableMode}[enable]
	_, err := wb.conn.WriteSingleRegister(wb.conf.ModeAddr, mode)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *StiebelIsg) MaxCurrent(current int64) error {
	return nil
}

func (wb *StiebelIsg) sollIst() (float64, float64, error) {
	soll, err := wb.conn.ReadInputRegisters(wb.conf.SollAddr, 1)
	if err != nil {
		return 0, 0, err
	}

	ist, err := wb.conn.ReadInputRegisters(wb.conf.IstAddr, 1)
	if err != nil {
		return 0, 0, err
	}

	if stiebel.Invalid(soll) || stiebel.Invalid(ist) {
		return 0, 0, errors.New("invalid reading")
	}

	return float64(encoding.Int16(soll)) / 10, float64(encoding.Int16(ist)) / 10, nil
}

var _ api.Battery = (*StiebelIsg)(nil)

func (wb *StiebelIsg) Soc() (float64, error) {
	_, ist, err := wb.sollIst()
	return ist, err
}

var _ api.SocLimiter = (*StiebelIsg)(nil)

func (wb *StiebelIsg) GetLimitSoc() (int64, error) {
	soll, _, err := wb.sollIst()
	return int64(soll), err
}

var _ api.Diagnosis = (*StiebelIsg)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *StiebelIsg) Diagnose() {
	// if _, err := wb.conn.WriteSingleRegister(1510, 28*10); err != nil {
	// 	fmt.Println(err)
	// }

	// {
	// 	fmt.Println()
	// 	reg := stiebel.Block3[0]
	// 	if b, err := wb.conn.ReadInputRegisters(reg.Addr(), 1); err == nil {
	// 		wb.print(reg, b)
	// 	}
	// }

	// {
	// 	fmt.Println()
	// 	ist, _ := wb.conn.ReadInputRegisters(521, 1)
	// 	soll, _ := wb.conn.ReadInputRegisters(522, 1)

	// 	fmt.Println((float64(encoding.Int16(soll))-float64(encoding.Int16(ist)))/10*100*4.2/3.6e3, "kWh")
	// }

	for _, reg := range stiebel.Block1 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr(), 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block2 {
		if b, err := wb.conn.ReadHoldingRegisters(reg.Addr(), 1); err == nil {
			wb.print(reg, b)
		}
	}

	// fmt.Println()
	// for _, reg := range stiebel.Block3 {
	// 	if b, err := wb.conn.ReadInputRegisters(reg.Addr(), 1); err == nil {
	// 		wb.print(reg, b)
	// 	}
	// }

	fmt.Println()
	for _, reg := range stiebel.Block4 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr(), 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block5 {
		if b, err := wb.conn.ReadHoldingRegisters(reg.Addr(), 1); err == nil {
			wb.print(reg, b)
		}
	}

	fmt.Println()
	for _, reg := range stiebel.Block6 {
		if b, err := wb.conn.ReadInputRegisters(reg.Addr(), 1); err == nil {
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
		fmt.Printf("\t%d %s:\t%04X\n", reg.Addr(), name, b)

	default:
		f := reg.Float(b)
		if math.IsNaN(f) {
			return
		}

		fmt.Printf("\t%d %s:\t%.1f%s\n", reg.Addr(), name, f, reg.Unit)
	}
}
