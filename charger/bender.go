package charger

// LICENSE

// Copyright (c) 2022 premultiply

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

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// BenderCC charger implementation
type BenderCC struct {
	conn    *modbus.Connection
	current uint16
	legacy  bool
}

const (
	// all holding type registers
	bendRegChargePointState       = 122  // Vehicle (Control Pilot) state
	bendRegPhaseEnergy            = 200  // Phase energy from primary meter (Wh)
	bendRegCurrents               = 212  // Currents from primary meter (mA)
	bendRegTotalEnergy            = 218  // Total Energy from primary meter (Wh)
	bendRegActivePower            = 220  // Active Power from primary meter (W)
	bendRegVoltages               = 222  // Voltages of the ocpp meter (V)
	bendRegChargedEnergyLegacy    = 705  // Sum of charged energy for the current session (Wh)
	bendRegChargingDurationLegacy = 709  // Duration since beginning of charge (Seconds)
	bendRegChargedEnergy          = 716  // Sum of charged energy for the current session (Wh)
	bendRegChargingDuration       = 718  // Duration since beginning of charge (Seconds)
	bendRegUserID                 = 720  // User ID (OCPP IdTag) from the current session. Bytes 0 to 19.
	bendRegEVCCID                 = 741  // ASCII representation of the Hex. Values corresponding to the EVCCID. Bytes 0 to 11.
	bendRegHemsCurrentLimit       = 1000 // Current limit of the HEMS module (A)

	bendRegFirmware             = 100 // Application version number
	bendRegOcppCpStatus         = 104 // Charge Point status according to the OCPP spec. enumaration
	bendRegProtocolVersion      = 120 // Ebee Modbus TCP Server Protocol Version number
	bendRegChargePointModel     = 142 // ChargePoint Model. Bytes 0 to 19.
	bendRegSmartVehicleDetected = 740 // Returns 1 if an EV currently connected is a smart vehicle, or 0 if no EV connected or it is not a smart vehicle
)

func init() {
	registry.Add("bender", NewBenderCCFromConfig)
}

// NewBenderCCFromConfig creates a BenderCC charger from generic config
func NewBenderCCFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewBenderCC(cc.URI, cc.ID)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateBenderCC -b *BenderCC -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.ChargeRater,ChargedEnergy,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Identifier,Identify,func() (string, error)"

// NewBenderCC creates BenderCC charger
func NewBenderCC(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
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
	}

	// check legacy register set
	if _, err := wb.conn.ReadHoldingRegisters(bendRegChargePointModel, 10); err != nil {
		wb.legacy = true
	}

	var (
		currentPower  func() (float64, error)
		currents      func() (float64, float64, float64, error)
		voltages      func() (float64, float64, float64, error)
		chargedEnergy func() (float64, error)
		totalEnergy   func() (float64, error)
		identify      func() (string, error)
	)

	// check presence of metering
	reg := uint16(bendRegActivePower)
	if wb.legacy {
		reg = bendRegPhaseEnergy
	}

	if b, err := wb.conn.ReadHoldingRegisters(reg, 2); err == nil && binary.BigEndian.Uint32(b) != math.MaxUint32 {
		currentPower = wb.currentPower
		currents = wb.currents
		voltages = wb.voltages
		chargedEnergy = wb.chargedEnergy
		totalEnergy = wb.totalEnergy
	}

	// check rfid
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	return decorateBenderCC(wb, currentPower, currents, voltages, chargedEnergy, totalEnergy, identify), nil
}

// Status implements the api.Charger interface
func (wb *BenderCC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(bendRegChargePointState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch sb := binary.BigEndian.Uint16(b); sb {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 4:
		return api.StatusD, nil
	case 5:
		return api.StatusE, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", sb)
	}
}

// Enabled implements the api.Charger interface
func (wb *BenderCC) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(bendRegHemsCurrentLimit, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) != 0, nil
}

// Enable implements the api.Charger interface
func (wb *BenderCC) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(bendRegHemsCurrentLimit, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *BenderCC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(bendRegHemsCurrentLimit, 1, b)
	if err == nil {
		wb.current = uint16(current)
	}

	return err
}

var _ api.ChargeTimer = (*BenderCC)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *BenderCC) ChargingTime() (time.Duration, error) {
	if wb.legacy {
		b, err := wb.conn.ReadHoldingRegisters(bendRegChargingDurationLegacy, 1)
		if err != nil {
			return 0, err
		}

		return time.Duration(binary.BigEndian.Uint16(b)) * time.Second, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(bendRegChargingDuration, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

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

// ChargedEnergy implements the api.ChargeRater interface
func (wb *BenderCC) chargedEnergy() (float64, error) {
	if wb.legacy {
		b, err := wb.conn.ReadHoldingRegisters(bendRegChargedEnergyLegacy, 1)
		if err != nil {
			return 0, err
		}

		return float64(binary.BigEndian.Uint16(b)) / 1e3, nil
	}

	b, err := wb.conn.ReadHoldingRegisters(bendRegChargedEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *BenderCC) totalEnergy() (float64, error) {
	if wb.legacy {
		b, err := wb.conn.ReadHoldingRegisters(bendRegPhaseEnergy, 6)
		if err != nil {
			return 0, err
		}

		var total float64
		for l := 0; l < 3; l++ {
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

// currents implements the api.PhaseCurrents interface
func (wb *BenderCC) currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(bendRegCurrents, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var curr [3]float64
	for l := 0; l < 3; l++ {
		curr[l] = float64(binary.BigEndian.Uint32(b[4*l:4*(l+1)])) / 1e3
	}

	return curr[0], curr[1], curr[2], nil
}

// voltages implements the api.PhaseVoltages interface
func (wb *BenderCC) voltages() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(bendRegVoltages, 6)
	if err != nil {
		return 0, 0, 0, err
	}

	var volt [3]float64
	for l := 0; l < 3; l++ {
		volt[l] = float64(binary.BigEndian.Uint32(b[4*l : 4*(l+1)]))
	}

	return volt[0], volt[1], volt[2], nil
}

// identify implements the api.Identifier interface
func (wb *BenderCC) identify() (string, error) {
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
