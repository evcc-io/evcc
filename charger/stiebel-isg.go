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
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/volkszaehler/mbmd/encoding"
)

// StiebelIsg charger implementation
type StiebelIsg struct {
	log        *util.Logger
	conn       *modbus.Connection
	enabled    bool
	lp         loadpoint.API
	tempConfig TempConfig
}

type TempConfig struct {
	SollAddr, IstAddr uint16
	TempDelta         float64
	ModeAddr          uint16
	EnableMode        uint16
	DisableMode       uint16
	EnabledAddr       uint16
	EnabledBits       uint16
	Speicher          float64
	Wärmekoeffizient  float64
}

func init() {
	registry.Add("stiebel-isg", NewStiebelIsgFromConfig)
}

// NewStiebelIsgFromConfig creates a Stiebel ISG charger from generic config
func NewStiebelIsgFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		TempConfig         `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
		TempConfig: TempConfig{
			// temp
			SollAddr:  522, // WW
			IstAddr:   521, // WW
			TempDelta: 5,   // °C
			// enable/disable
			ModeAddr:    1500, // Betriebsart
			EnableMode:  3,    // Komfortbetrieb
			DisableMode: 2,    // Programmbetrieb
			// enabled
			EnabledAddr: 2500,   // Betriebsstatus
			EnabledBits: 1 << 5, // WW Betrieb
			// medium
			Wärmekoeffizient: 4.18, // kJ/kgK
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewStiebelIsg(cc.URI, cc.ID, cc.TempConfig)
}

// NewStiebelIsg creates Stiebel ISG charger
func NewStiebelIsg(uri string, slaveID uint8, tempConfig TempConfig) (api.Charger, error) {
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
		log:        log,
		conn:       conn,
		tempConfig: tempConfig,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *StiebelIsg) Status() (api.ChargeStatus, error) {
	res := api.StatusNone

	ist, err := wb.conn.ReadInputRegisters(wb.tempConfig.IstAddr, 1)
	if err != nil {
		return res, err
	}

	soll, err := wb.conn.ReadInputRegisters(wb.tempConfig.SollAddr, 1)
	if err != nil {
		return res, err
	}

	istF := float64(encoding.Int16(ist)) / 10
	sollF := float64(encoding.Int16(soll)) / 10
	energyRequired := (sollF - istF) * wb.tempConfig.Speicher * wb.tempConfig.Wärmekoeffizient / 3.6e3

	wb.log.DEBUG.Printf("ist: %.1f°C, soll: %.1f°C, energy required: %.3fkWh", istF, sollF, energyRequired)

	res = api.StatusA

	// become "connected" if temp is outside of temp delta
	if sollF-istF > wb.tempConfig.TempDelta {
		res = api.StatusB
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (wb *StiebelIsg) Enabled() (bool, error) {
	b, err := wb.conn.ReadInputRegisters(wb.tempConfig.EnabledAddr, 1)
	if err != nil {
		return false, err
	}

	return encoding.Uint16(b)&wb.tempConfig.EnabledBits != 0, nil
}

// Enable implements the api.Charger interface
func (wb *StiebelIsg) Enable(enable bool) error {
	enabled, err := wb.Enabled()
	if err != nil {
		return err
	}

	if enabled == enable {
		return nil
	}

	b, err := wb.conn.ReadHoldingRegisters(wb.tempConfig.ModeAddr, 1)
	if err != nil {
		return err
	}

	status := encoding.Uint16(b)
	value := map[bool]uint16{true: wb.tempConfig.EnableMode, false: wb.tempConfig.DisableMode}[enable]

	if status != value {
		// don't disable unless pump is silent
		if enabled && !enable {
			return api.ErrMustRetry
		}

		// set new mode
		_, err := wb.conn.WriteSingleRegister(wb.tempConfig.ModeAddr, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *StiebelIsg) MaxCurrent(current int64) error {
	return nil
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

// LoadpointControl implements loadpoint.Controller
func (c *StiebelIsg) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
