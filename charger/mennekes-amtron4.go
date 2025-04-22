package charger

// LICENSE

// Copyright (c) 2022-2025 premultiply, opitzb86, mh81

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
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// Amtron4you/4business charger implementation
type Amtron4 struct {
	*BenderCC
	current float64
	phases  int
	lp      loadpoint.API
}

const (
	// Amtron 4You (SW >=1.1)
	amtronRegHemsCurrentLimit = 1001 // Current limit of the HEMS module (0.1 A)
	amtronRegHemsPowerLimit   = 1002 // Power limit of the HEMS module (W)
)

func init() {
	registry.AddCtx("mennekes-amtron4", NewAmtron4FromConfig)
}

// NewAmtron4FromConfig creates a Amtron4 charger from generic config
func NewAmtron4FromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewAmtron4(ctx, cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	var (
		currentPower func() (float64, error)
		currents     func() (float64, float64, float64, error)
		voltages     func() (float64, float64, float64, error)
		totalEnergy  func() (float64, error)

		// DON'T EXIST ??
		// soc          func() (float64, error)
		// identify     func() (string, error)
	)

	// check presence of metering
	reg := uint16(bendRegActivePower)

	// TODO is this really necessary since voltage is always used?
	if b, err := wb.conn.ReadHoldingRegisters(reg, 2); err == nil && binary.BigEndian.Uint32(b) != math.MaxUint32 {
		currentPower = wb.currentPower
		currents = wb.currents
		totalEnergy = wb.totalEnergy

		// check presence of "ocpp meter"
		if b, err := wb.conn.ReadHoldingRegisters(bendRegVoltages, 2); err == nil && binary.BigEndian.Uint32(b) > 0 {
			voltages = wb.voltages
		}
	}

	return decorateAmtron4(wb, currentPower, currents, voltages, totalEnergy), nil
}

//go:generate go tool decorate -f decorateAmtron4 -b *Amtron4 -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)"

// NewAmtron4 creates Amtron4 charger
func NewAmtron4(ctx context.Context, uri string, id uint8) (*Amtron4, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	bcc, err := NewBenderCC(ctx, uri, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("amtron4")
	bcc.conn.Logger(log.TRACE)

	wb := &Amtron4{
		BenderCC: bcc,
		current:  6, // assume min current
	}

	return wb, nil
}

// Enabled implements the api.Charger interface
func (wb *Amtron4) Enabled() (bool, error) {
	// Check if the charger is enabled by reading the HEMS Power and Current limits.
	// If both limit are non-zero, the charger is enabled.
	// If either limit is zero, the charger is disabled.
	bp, err := wb.conn.ReadHoldingRegisters(amtronRegHemsPowerLimit, 1)
	if err != nil {
		return false, err
	}

	bc, err := wb.conn.ReadHoldingRegisters(bendRegHemsCurrentLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(bp) != 0 && binary.BigEndian.Uint16(bc) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Amtron4) Enable(enable bool) error {

	phases := wb.phases
	if phases == 0 && wb.lp != nil {
		// in case loadpoint has fixed phase configuration
		phases = wb.lp.GetPhases()
	}

	bp := make([]byte, 2)
	bc := make([]byte, 2)

	if enable {
		binary.BigEndian.PutUint16(bp, uint16(230*16*phases))
		binary.BigEndian.PutUint16(bc, uint16(wb.current*10))
	}

	if _, err := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, bp); err != nil {
		return fmt.Errorf("power limit: %v", err)
	}

	if _, err := wb.conn.WriteMultipleRegisters(amtronRegHemsCurrentLimit, 1, bc); err != nil {
		return fmt.Errorf("current limit: %v", err)
	}

	return nil
}

var _ api.ChargerEx = (*Amtron4)(nil)

// MaxCurrent implements the api.Charger interface
func (wb *Amtron4) MaxCurrentMillis(current float64) error {

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current*10))

	_, err := wb.conn.WriteMultipleRegisters(amtronRegHemsCurrentLimit, 1, b)

	if err == nil {
		wb.current = current
	}

	return err
}

func (wb *Amtron4) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.PhaseSwitcher = (*Amtron4)(nil)

func (wb *Amtron4) Phases1p3p(phases int) error {

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(230*16*phases))
	_, err := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, b)

	if err == nil {
		wb.phases = phases
	}

	return err
}

var _ loadpoint.Controller = (*Delta)(nil)

// LoadpointControl implements loadpoint.Controller
func (wb *Amtron4) LoadpointControl(lp loadpoint.API) {
	wb.lp = lp
}
