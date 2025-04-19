package charger

// LICENSE

// Copyright (c) 2025 opitb86

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
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// MennekesAmtron charger implementation
type MennekesAmtron struct {
	conn    *modbus.Connection
	current float64
	legacy  bool
	phases  int
}

const (
	// all holding type registers
	amtronRegChargePointState = 122  // Vehicle (Control Pilot) state
	amtronRegPhaseEnergy      = 200  // Phase energy from primary meter (Wh)
	amtronRegCurrents         = 212  // Currents from primary meter (mA)
	amtronRegTotalEnergy      = 218  // Total Energy from primary meter (Wh)
	amtronRegActivePower      = 220  // Active Power from primary meter (W)
	amtronRegVoltages         = 222  // Voltages of the ocpp meter (V)
	amtronRegUserID           = 720  // User ID (OCPP IdTag) from the current session. Bytes 0 to 19.
	amtronRegEVBatteryState   = 730  // EV Battery State (% 0-100)
	amtronRegEVCCID           = 741  // ASCII representation of the Hex. Values corresponding to the EVCCID. Bytes 0 to 11.
	amtronRegHemsCurrentLimit = 1000 // Current limit of the HEMS module (A)
	amtronRegHemsPowerLimit   = 1002 // Power limit of the HEMS module (W)

	amtronRegFirmware             = 100 // Application version number
	amtronRegOcppCpStatus         = 104 // Charge Point status according to the OCPP spec. enumaration
	amtronRegProtocolVersion      = 120 // Ebee Modbus TCP Server Protocol Version number
	amtronRegChargePointModel     = 142 // ChargePoint Model. Bytes 0 to 19.
	amtronRegSmartVehicleDetected = 740 // Returns 1 if an EV currently connected is a smart vehicle, or 0 if no EV connected or it is not a smart vehicle
)

func init() {
	registry.AddCtx("mennekes-amtron", NewMennekesAmtronFromConfig)
}

// NewMennekesAmtronFromConfig creates a MennekesAmtron charger from generic config
func NewMennekesAmtronFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewMennekesAmtron(ctx, cc.URI, cc.ID)
}

//go:generate go tool decorate -f decorateMennekesAmtron -b *MennekesAmtron -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.Identifier,Identify,func() (string, error)"

// NewMennekesAmtron creates MennekesAmtron charger
func NewMennekesAmtron(ctx context.Context, uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("mennekes")
	conn.Logger(log.TRACE)

	wb := &MennekesAmtron{
		conn:    conn,
		current: 6, // assume min current
		phases:  3, // assume default phases
	}

	// check legacy register set
	if _, err := wb.conn.ReadHoldingRegisters(amtronRegChargePointModel, 10); err != nil {
		wb.legacy = true
	}

	var (
		currentPower func() (float64, error)
		currents     func() (float64, float64, float64, error)
		voltages     func() (float64, float64, float64, error)
		totalEnergy  func() (float64, error)
		soc          func() (float64, error)
		identify     func() (string, error)
	)

	// check presence of metering
	reg := uint16(amtronRegActivePower)
	if wb.legacy {
		reg = amtronRegPhaseEnergy
	}

	if b, err := wb.conn.ReadHoldingRegisters(reg, 2); err == nil && binary.BigEndian.Uint32(b) != math.MaxUint32 {
		currentPower = wb.currentPower
		currents = wb.currents
		totalEnergy = wb.totalEnergy

		// check presence of "ocpp meter"
		if b, err := wb.conn.ReadHoldingRegisters(amtronRegVoltages, 2); err == nil && binary.BigEndian.Uint32(b) > 0 {
			voltages = wb.voltages
		}

		if !wb.legacy {
			if _, err := wb.conn.ReadHoldingRegisters(amtronRegEVBatteryState, 1); err == nil {
				soc = wb.soc
			}
		}
	}

	// check rfid
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	return decorateMennekesAmtron(wb, currentPower, currents, voltages, totalEnergy, soc, identify), nil
}

// Status implements the api.Charger interface
func (wb *MennekesAmtron) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(amtronRegChargePointState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := binary.BigEndian.Uint16(b); s {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3, 4:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *MennekesAmtron) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(amtronRegHemsPowerLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *MennekesAmtron) Enable(enable bool) error {

	v1, v2, v3, err1 := wb.voltages()
	if err1 != nil {
		return fmt.Errorf("error reading voltages: %v", err1)
	}

	maxVoltage := math.Max(v1, math.Max(v2, v3))
	maxVoltage_tol := maxVoltage * 1.02

	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, uint16(wb.current*maxVoltage_tol*float64(wb.phases)))
	}

	_, err := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *MennekesAmtron) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*MennekesAmtron)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *MennekesAmtron) MaxCurrentMillis(current float64) error {

	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	v1, v2, v3, err1 := wb.voltages()
	if err1 != nil {
		return fmt.Errorf("error reading voltages: %v", err1)
	}

	maxVoltage := math.Max(v1, math.Max(v2, v3))
	maxVoltage_tol := maxVoltage * 1.02

	power := uint16(current * maxVoltage_tol * float64(wb.phases))

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(power))

	_, err := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, b)
	if err == nil {
		wb.current = float64(current)
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *MennekesAmtron) currentPower() (float64, error) {
	if wb.legacy {
		l1, l2, l3, err := wb.currents()
		return 230 * (l1 + l2 + l3), err
	}

	b, err := wb.conn.ReadHoldingRegisters(amtronRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *MennekesAmtron) totalEnergy() (float64, error) {
	if wb.legacy {
		b, err := wb.conn.ReadHoldingRegisters(amtronRegPhaseEnergy, 6)
		if err != nil {
			return 0, err
		}

		var total float64
		for l := range 3 {
			total += float64(binary.BigEndian.Uint32(b[4*l:4*(l+1)])) / 1e3
		}

		return total, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(amtronRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// getPhaseValues returns 3 sequential register values
func (wb *MennekesAmtron) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(reg, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := range res {
		u32 := binary.BigEndian.Uint32(b[4*i:])
		if u32 == math.MaxUint32 {
			u32 = 0
		}
		res[i] = float64(u32) / divider
	}

	return res[0], res[1], res[2], nil
}

// currents implements the api.PhaseCurrents interface
func (wb *MennekesAmtron) currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(amtronRegCurrents, 1e3)
}

// voltages implements the api.PhaseVoltages interface
func (wb *MennekesAmtron) voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(amtronRegVoltages, 1)
}

// identify implements the api.Identifier interface
func (wb *MennekesAmtron) identify() (string, error) {
	if !wb.legacy {
		b, err := wb.conn.ReadHoldingRegisters(amtronRegSmartVehicleDetected, 1)
		if err == nil && binary.BigEndian.Uint16(b) != 0 {
			b, err = wb.conn.ReadHoldingRegisters(amtronRegEVCCID, 6)
		}

		if id := bytesAsString(b); id != "" || err != nil {
			return id, err
		}
	}

	b, err := wb.conn.ReadHoldingRegisters(amtronRegUserID, 10)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

// soc implements the api.Battery interface
func (wb *MennekesAmtron) soc() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(amtronRegSmartVehicleDetected, 1)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint16(b) == 1 {
		b, err = wb.conn.ReadHoldingRegisters(amtronRegEVBatteryState, 1)
		if err != nil {
			return 0, err
		}
		if soc := binary.BigEndian.Uint16(b); soc <= 100 {
			return float64(soc), nil
		}
	}

	return 0, api.ErrNotAvailable
}

var _ api.PhaseSwitcher = (*MennekesAmtron)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *MennekesAmtron) Phases1p3p(phases int) error {
	fmt.Printf("Switching to %d phases\n", phases)
	wb.phases = phases

	b := make([]byte, 2)
	if phases == 1 {
		binary.BigEndian.PutUint16(b, uint16(3500))
	} else if phases == 3 {
		binary.BigEndian.PutUint16(b, uint16(4500))
	}

	_, err := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, b)

	return err
}

var _ api.Diagnosis = (*MennekesAmtron)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *MennekesAmtron) Diagnose() {
	fmt.Printf("\tLegacy:\t\t%t\n", wb.legacy)
	if !wb.legacy {
		if b, err := wb.conn.ReadHoldingRegisters(amtronRegChargePointModel, 10); err == nil {
			fmt.Printf("\tModel:\t%s\n", b)
		}
	}
	if b, err := wb.conn.ReadHoldingRegisters(amtronRegFirmware, 2); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(amtronRegProtocolVersion, 2); err == nil {
		fmt.Printf("\tProtocol:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(amtronRegOcppCpStatus, 1); err == nil {
		fmt.Printf("\tOCPP Status:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if !wb.legacy {
		if b, err := wb.conn.ReadHoldingRegisters(amtronRegSmartVehicleDetected, 1); err == nil {
			fmt.Printf("\tSmart Vehicle:\t%t\n", binary.BigEndian.Uint16(b) != 0)
		}
	}
	if b, err := wb.conn.ReadHoldingRegisters(amtronRegEVCCID, 6); err == nil {
		fmt.Printf("\tEVCCID:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(amtronRegUserID, 10); err == nil {
		fmt.Printf("\tUserID:\t%s\n", b)
	}
}
