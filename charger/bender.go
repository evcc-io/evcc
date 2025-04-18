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

// Supports all chargers based on Bender CC612/613 controller series
// * The 'Modbus TCP Server for energy management systems' must be enabled.
// * The setting 'Register Address Set' must NOT be set to 'Phoenix', 'TQ-DM100' or 'ISE/IGT Kassel'.
//   -> Use the third selection labeled 'Ebee', 'Bender', 'MENNEKES' etc.
// * Set 'Allow UID Disclose' to On

// Supports dynamic phase switching for Mennekes Amtron 4You 5xx Series and 4Business 7xx (same charger type, but with Eichrecht)

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// BenderCC charger implementation
type BenderCC struct {
	conn    *modbus.Connection
	current float64
	legacy  bool
	model   string
	phases  int
}

const (
	// all holding type registers
	bendRegChargePointState   = 122  // Vehicle (Control Pilot) state
	bendRegPhaseEnergy        = 200  // Phase energy from primary meter (Wh)
	bendRegCurrents           = 212  // Currents from primary meter (mA)
	bendRegTotalEnergy        = 218  // Total Energy from primary meter (Wh)
	bendRegActivePower        = 220  // Active Power from primary meter (W)
	bendRegVoltages           = 222  // Voltages of the ocpp meter (V)
	bendRegUserID             = 720  // User ID (OCPP IdTag) from the current session. Bytes 0 to 19.
	bendRegEVBatteryState     = 730  // EV Battery State (% 0-100)
	bendRegEVCCID             = 741  // ASCII representation of the Hex. Values corresponding to the EVCCID. Bytes 0 to 11.
	bendRegHemsCurrentLimit   = 1000 // Current limit of the HEMS module (A)
	amtronRegHemsCurrentLimit = 1001 // Current limit of the HEMS module (0.1 A) only used for Amtron 4You
	amtronRegHemsPowerLimit   = 1002 // Power limit of the HEMS module (W) only used for Amtron 4You (SW >=1.1)

	bendRegFirmware             = 100 // Application version number
	bendRegOcppCpStatus         = 104 // Charge Point status according to the OCPP spec. enumaration
	bendRegProtocolVersion      = 120 // Ebee Modbus TCP Server Protocol Version number
	bendRegChargePointModel     = 142 // ChargePoint Model. Bytes 0 to 19.
	bendRegSmartVehicleDetected = 740 // Returns 1 if an EV currently connected is a smart vehicle, or 0 if no EV connected or it is not a smart vehicle

	// unused
	// bendRegChargedEnergyLegacy    = 705 // Sum of charged energy for the current session (Wh)
	// bendRegChargingDurationLegacy = 709 // Duration since beginning of charge (Seconds)
	// bendRegChargedEnergy          = 716 // Sum of charged energy for the current session (Wh)
	// bendRegChargingDuration       = 718 // Duration since beginning of charge (Seconds)
)

func init() {
	registry.AddCtx("bender", NewBenderCCFromConfig)
}

// NewBenderCCFromConfig creates a BenderCC charger from generic config
func NewBenderCCFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewBenderCC(ctx, cc.URI, cc.ID)
}

//go:generate go tool decorate -f decorateBenderCC -b *BenderCC -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewBenderCC creates BenderCC charger
func NewBenderCC(ctx context.Context, uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("bender")
	conn.Logger(log.TRACE)

	wb := &BenderCC{
		conn:    conn,
		current: 6, // assume min current
		model:   "bender",
		phases:  3,
	}

	// check legacy register set and if the wb is a Mennekes Amtron 4You
	bModel, err := wb.conn.ReadHoldingRegisters(bendRegChargePointModel, 10)
	if err != nil {
		wb.legacy = true
		wb.model = "bender"
	} else {
		if strings.Contains(strings.ToLower(string(bModel[:])), "4you") ||
			strings.Contains(strings.ToLower(string(bModel[:])), "4business") {
			wb.model = "4you"
		}
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
	reg := uint16(bendRegActivePower)
	if wb.legacy {
		reg = bendRegPhaseEnergy
	}

	if b, err := wb.conn.ReadHoldingRegisters(reg, 2); err == nil && binary.BigEndian.Uint32(b) != math.MaxUint32 {
		currentPower = wb.currentPower
		currents = wb.currents
		totalEnergy = wb.totalEnergy

		// check presence of "ocpp meter"
		if b, err := wb.conn.ReadHoldingRegisters(bendRegVoltages, 2); err == nil && binary.BigEndian.Uint32(b) > 0 {
			voltages = wb.voltages
		}

		if !wb.legacy {
			if _, err := wb.conn.ReadHoldingRegisters(bendRegEVBatteryState, 1); err == nil {
				soc = wb.soc
			}
		}
	}

	// check rfid
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	// decorate phases - phases1p3p is only available for Amtron 4You
	// option should not be available in GUI for other models
	var phases1p3p func(int) error
	if wb.model == "4you" {
		phases1p3p = wb.phases1p3p
	}

	return decorateBenderCC(wb, currentPower, currents, voltages, totalEnergy, soc, identify, phases1p3p), nil
}

// Status implements the api.Charger interface
func (wb *BenderCC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(bendRegChargePointState, 1)
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
func (wb *BenderCC) Enabled() (bool, error) {

	if wb.model != "4you" {
		b, err := wb.conn.ReadHoldingRegisters(bendRegHemsCurrentLimit, 1)
		if err != nil {
			return false, err
		}

		return binary.BigEndian.Uint16(b) != 0, nil
	} else {

		// Check if the charger is enabled by reading the HEMS Power and Current limits.
		// If both limit are non-zero, the charger is enabled.
		// If either limit is zero, the charger is disabled.
		bPower, errPower := wb.conn.ReadHoldingRegisters(amtronRegHemsPowerLimit, 1)
		if errPower != nil {
			return false, errPower
		}

		bCurrent, errCurrent := wb.conn.ReadHoldingRegisters(bendRegHemsCurrentLimit, 1)
		if errCurrent != nil {
			return false, errCurrent
		}

		return binary.BigEndian.Uint16(bPower) != 0 && binary.BigEndian.Uint16(bCurrent) != 0, nil
	}
}

// calculate the power limit for the HEMS module
func (wb *BenderCC) CalculatePowerLimit(current float64) (float64, error) {
	v1, v2, v3, err := wb.voltages()
	if err != nil {
		return 0, fmt.Errorf("error reading voltages: %v", err)
	}

	// Calculate the power limit for the AMTRON 4You charger.
	// Ensure that the resulting power corresponds to at least 6A charging current
	// to guarantee charging functionality.
	// Use the maximum voltage among the three phases,
	// applying a 2% tolerance to account for voltage fluctuations.
	maxVoltage := math.Max(v1, math.Max(v2, v3))
	maxVoltageTol := maxVoltage * 1.02

	powerLimit := current * maxVoltageTol * float64(wb.phases)

	return powerLimit, nil
}

// Enable implements the api.Charger interface
func (wb *BenderCC) Enable(enable bool) error {

	if wb.model != "4you" {
		b := make([]byte, 2)
		if enable {
			binary.BigEndian.PutUint16(b, uint16(wb.current))
		}

		_, err := wb.conn.WriteMultipleRegisters(bendRegHemsCurrentLimit, 1, b)

		return err

	} else {
		// Ensure the current limit is set to address potential issues with undefined states
		// that may occur after charger timeouts or reboots.

		// Calculate the power limit based on the current setting.
		powerlimit, err1 := wb.CalculatePowerLimit(wb.current)

		if err1 != nil {
			return fmt.Errorf("error calculating power limit: %v", err1)
		}

		bp := make([]byte, 2)
		bc := make([]byte, 2)

		// If enabling, set HEMS Power to the calculated power limit and HEMS Current to 16A
		// If disabling, set both HEMS Power and HEMS Current to 0
		if enable {
			binary.BigEndian.PutUint16(bp, uint16(powerlimit)) // Set power limit
			binary.BigEndian.PutUint16(bc, uint16(16))         // Set current limit to 16A
		} else {
			binary.BigEndian.PutUint16(bp, uint16(0)) // Disable power limit
			binary.BigEndian.PutUint16(bc, uint16(0)) // Disable current limit
		}

		_, err_p := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, bp)
		_, err_c := wb.conn.WriteMultipleRegisters(bendRegHemsCurrentLimit, 1, bc)

		// If there are errors in either of the writes, return a combined error message
		if err_p != nil || err_c != nil {
			return fmt.Errorf("Error setting HEMS Power Limit: %v, Error setting HEMS Current Limit: %v", err_p, err_c)
		}

		// Return nil if both operations succeed
		return nil

	}
}

var _ api.ChargerEx = (*BenderCC)(nil)

// MaxCurrent implements the api.Charger interface
func (wb *BenderCC) MaxCurrentMillis(current float64) error {

	// Calculate the power limit based on the current setting.
	powerlimit, err1 := wb.CalculatePowerLimit(current)
	if err1 != nil {
		return fmt.Errorf("error calculating power limit: %v", err1)
	}

	bp := make([]byte, 2)
	binary.BigEndian.PutUint16(bp, uint16(powerlimit))

	_, err_sp := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, bp)

	if err_sp != nil {
		return fmt.Errorf("Error setting HEMS power: %v", err_sp)
	}
	wb.current = current
	return nil

}

func (wb *BenderCC) MaxCurrent(current int64) error {

	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	// use power setpoint based on milliampere values for AMTRON 4You
	if wb.model == "4you" {
		return wb.MaxCurrentMillis(float64(current))
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(bendRegHemsCurrentLimit, 1, b)
	if err == nil {
		wb.current = float64(uint16(current))
	}

	return err

}

// removed: https://github.com/evcc-io/evcc/issues/13555
// var _ api.ChargeTimer = (*BenderCC)(nil)

// CurrentPower implements the api.Meter interface
func (wb *BenderCC) currentPower() (float64, error) {
	if wb.legacy {
		l1, l2, l3, err := wb.currents()
		return 230 * (l1 + l2 + l3), err
	}

	b, err := wb.conn.ReadHoldingRegisters(bendRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

// removed: https://github.com/evcc-io/evcc/issues/13726
// var _ api.ChargeRater = (*BenderCC)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *BenderCC) totalEnergy() (float64, error) {
	if wb.legacy {
		b, err := wb.conn.ReadHoldingRegisters(bendRegPhaseEnergy, 6)
		if err != nil {
			return 0, err
		}

		var total float64
		for l := range 3 {
			total += float64(binary.BigEndian.Uint32(b[4*l:4*(l+1)])) / 1e3
		}

		return total, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(bendRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// getPhaseValues returns 3 sequential register values
func (wb *BenderCC) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
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
func (wb *BenderCC) currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(bendRegCurrents, 1e3)
}

// voltages implements the api.PhaseVoltages interface
func (wb *BenderCC) voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(bendRegVoltages, 1)
}

func (wb *BenderCC) phases1p3p(phases int) error {
	if wb.model == "4you" {
		fmt.Printf("Switching to %d phases\n", phases)

		b := make([]byte, 2)
		if phases == 1 {
			// Set the power limit to a minimum value for single-phase operation (1500W).
			// This includes an offset to ensure the charger remains safely in single-phase mode.
			// Helps avoid unexpected behavior in "PV+Min" mode when PV power is below 6A.
			binary.BigEndian.PutUint16(b, uint16(1500))
		} else {
			// Set the power limit to a minimum value for three-phase operation (4500W).
			// This includes an offset to ensure the charger remains safely in three-phase mode.
			binary.BigEndian.PutUint16(b, uint16(4500))
		}

		_, err := wb.conn.WriteMultipleRegisters(amtronRegHemsPowerLimit, 1, b)

		if err == nil {
			wb.phases = phases
		}

		return err
	}
	return api.ErrNotAvailable
}

// identify implements the api.Identifier interface
func (wb *BenderCC) identify() (string, error) {
	if wb.model == "bender" {
		if !wb.legacy {
			b, err := wb.conn.ReadHoldingRegisters(bendRegSmartVehicleDetected, 1)
			if err == nil && binary.BigEndian.Uint16(b) != 0 {
				b, err = wb.conn.ReadHoldingRegisters(bendRegEVCCID, 6)
			}

			if id := bytesAsString(b); id != "" || err != nil {
				return id, err
			}
		}

		b, err := wb.conn.ReadHoldingRegisters(bendRegUserID, 10)
		if err != nil {
			return "", err
		}

		return bytesAsString(b), nil
	}
	return "N/A", api.ErrNotAvailable
}

// soc implements the api.Battery interface
func (wb *BenderCC) soc() (float64, error) {
	if wb.model == "bender" {
		b, err := wb.conn.ReadHoldingRegisters(bendRegSmartVehicleDetected, 1)
		if err != nil {
			return 0, err
		}

		if binary.BigEndian.Uint16(b) == 1 {
			b, err = wb.conn.ReadHoldingRegisters(bendRegEVBatteryState, 1)
			if err != nil {
				return 0, err
			}
			if soc := binary.BigEndian.Uint16(b); soc <= 100 {
				return float64(soc), nil
			}
		}
	}
	return 0, api.ErrNotAvailable
}

var _ api.Diagnosis = (*BenderCC)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *BenderCC) Diagnose() {
	fmt.Printf("\tLegacy:\t\t%t\n", wb.legacy)
	if !wb.legacy {
		if b, err := wb.conn.ReadHoldingRegisters(bendRegChargePointModel, 10); err == nil {
			fmt.Printf("\tModel:\t%s\n", b)
		}
	}
	if b, err := wb.conn.ReadHoldingRegisters(bendRegFirmware, 2); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(bendRegProtocolVersion, 2); err == nil {
		fmt.Printf("\tProtocol:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(bendRegOcppCpStatus, 1); err == nil {
		fmt.Printf("\tOCPP Status:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if !wb.legacy {
		if b, err := wb.conn.ReadHoldingRegisters(bendRegSmartVehicleDetected, 1); err == nil {
			fmt.Printf("\tSmart Vehicle:\t%t\n", binary.BigEndian.Uint16(b) != 0)
		}
	}
	if b, err := wb.conn.ReadHoldingRegisters(bendRegEVCCID, 6); err == nil {
		fmt.Printf("\tEVCCID:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(bendRegUserID, 10); err == nil {
		fmt.Printf("\tUserID:\t%s\n", b)
	}
}
