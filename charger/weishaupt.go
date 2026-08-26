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
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Weishaupt heat pump charger implementation
type Weishaupt struct {
	conn    *modbus.Connection
	lp      loadpoint.API
	power   uint16
	tempReg uint16
}

// Datenpunktliste Modbus TCP (WWP), 83807301
// https://www.weishaupt.de/uploads/tx_weishaupt_documents/documents/83807301.pdf
//
// 3xxxx are input registers, 4xxxx are holding registers.
// SG Ready (35101/35102) is read-only, hence power control uses SollwertPV.
const (
	wsRegOutsideTemp = 30001 // Aussentemperatur, 0.1K
	wsRegDhwSetTemp  = 32101 // Warmwassersolltemperatur, 0.1K
	wsRegDhwTemp     = 32102 // Warmwassertemperatur, 0.1K
	wsRegPowerDemand = 33103 // Leistungsanforderung, %
	wsRegFlowTemp    = 33104 // Vorlauftemperatur, 0.1K
	wsRegBufferTemp  = 33108 // Weichentemperatur, 0.1K
	wsRegPvPower     = 40002 // SollwertPV, W
)

var wsTempSource = map[string]uint16{
	"warmwater": wsRegDhwTemp,
	"buffer":    wsRegBufferTemp,
	"flow":      wsRegFlowTemp,
	"outside":   wsRegOutsideTemp,
}

func init() {
	registry.AddCtx("weishaupt", NewWeishauptFromConfig)
}

// NewWeishauptFromConfig creates a Weishaupt charger from generic config
func NewWeishauptFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
		TempSource      string
	}{
		Settings: modbus.Settings{
			ID: 1,
		},
		TempSource: "warmwater",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWeishaupt(ctx, cc.Settings, cc.TempSource)
}

// NewWeishaupt creates Weishaupt charger
func NewWeishaupt(ctx context.Context, settings modbus.Settings, tempSource string) (api.Charger, error) {
	tempReg, ok := wsTempSource[tempSource]
	if !ok {
		return nil, fmt.Errorf("invalid temp source: %s", tempSource)
	}

	conn, err := settings.Connection(ctx)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("weishaupt")
	conn.Logger(log.TRACE)

	wb := &Weishaupt{
		conn:    conn,
		tempReg: tempReg,
	}

	// validate connection
	_, err = wb.getPower()

	return wb, err
}

var _ api.IconDescriber = (*Weishaupt)(nil)

// Icon implements the api.IconDescriber interface
func (wb *Weishaupt) Icon() string {
	return "heatpump"
}

var _ api.FeatureDescriber = (*Weishaupt)(nil)

// Features implements the api.FeatureDescriber interface
func (wb *Weishaupt) Features() []api.Feature {
	return []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice}
}

// temp reads a temperature sensor register. Values outside of -50..500°C
// indicate a missing, broken or digital sensor.
func (wb *Weishaupt) temp(reg uint16) (float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, 1)
	if err != nil {
		return 0, err
	}

	if v := int16(binary.BigEndian.Uint16(b)); v >= -500 && v <= 5000 {
		return float64(v) / 10, nil
	}

	return 0, api.ErrNotAvailable
}

func (wb *Weishaupt) getPower() (uint16, error) {
	b, err := wb.conn.ReadHoldingRegisters(wsRegPvPower, 1)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint16(b), nil
}

func (wb *Weishaupt) setPower(power uint16) error {
	_, err := wb.conn.WriteSingleRegister(wsRegPvPower, power)
	return err
}

// Status implements the api.Charger interface
func (wb *Weishaupt) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(wsRegPowerDemand, 1)
	if err != nil {
		return api.StatusNone, err
	}

	// 0..100%, anything else is invalid
	if demand := binary.BigEndian.Uint16(b); demand > 0 && demand <= 100 {
		return api.StatusC, nil
	}

	return api.StatusB, nil
}

// Enabled implements the api.Charger interface
func (wb *Weishaupt) Enabled() (bool, error) {
	power, err := wb.getPower()
	return power > 0, err
}

// Enable implements the api.Charger interface
func (wb *Weishaupt) Enable(enable bool) error {
	// writing 0 releases the power setpoint and returns the heat pump to normal operation
	var power uint16
	if enable {
		power = wb.power
	}

	return wb.setPower(power)
}

// MaxCurrent implements the api.Charger interface
func (wb *Weishaupt) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Weishaupt)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Weishaupt) MaxCurrentMillis(current float64) error {
	phases := 1
	if wb.lp != nil {
		if p := wb.lp.GetPhases(); p != 0 {
			phases = p
		}
	}

	power := uint16(min(voltage*current*float64(phases), 65535))

	err := wb.setPower(power)
	if err == nil {
		wb.power = power
	}

	return err
}

var _ api.Battery = (*Weishaupt)(nil)

// Soc implements the api.Battery interface
func (wb *Weishaupt) Soc() (float64, error) {
	return wb.temp(wb.tempReg)
}

var _ api.SocLimiter = (*Weishaupt)(nil)

// GetLimitSoc implements the api.SocLimiter interface
func (wb *Weishaupt) GetLimitSoc() (int64, error) {
	if wb.tempReg != wsRegDhwTemp {
		return 0, api.ErrNotAvailable
	}

	temp, err := wb.temp(wsRegDhwSetTemp)

	return int64(temp), err
}

var _ loadpoint.Controller = (*Weishaupt)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Weishaupt) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
