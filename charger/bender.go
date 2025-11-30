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

// Supports all chargers based on Bender CC612/613 controller series
// * The 'Modbus TCP Server for energy management systems' must be enabled.
// * The setting 'Register Address Set' must NOT be set to 'Phoenix', 'TQ-DM100' or 'ISE/IGT Kassel'.
//   -> Use the third selection labeled 'Ebee', 'Bender', 'MENNEKES' etc.
// * Set 'Allow UID Disclose' to On

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/semp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

type sempHandler struct {
	deviceID string
	conn     *semp.Connection
	deviceG  util.Cacheable[semp.Device2EM]
	phases   int
}

// BenderCC charger implementation
type BenderCC struct {
	conn    *modbus.Connection
	current uint16
	regCurr uint16
	legacy  bool
	log     *util.Logger
	semp    sempHandler
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
	bendRegHemsCurrentLimit   = 1000 // HEMS Current Limit (A). Only available on Mennekes Amtron 4You / 4Business chargers.
	bendRegHemsCurrentLimit10 = 1001 // HEMS Current Limit 1/10 (0.1 A). Only available on Mennekes Amtron 4You / 4Business chargers.
	bendRegHemsPowerLimit     = 1002 // HEMS Power Limit (W). Only available on Mennekes Amtron 4You / 4Business chargers.

	bendRegFirmware             = 100 // Application version number
	bendRegOcppCpStatus         = 104 // Charge Point status according to the OCPP spec. enumaration
	bendRegProtocolVersion      = 120 // Ebee Modbus TCP Server Protocol Version number
	bendRegRelayState           = 140 // State of the internal relay (0: off, 1: 3 phases active 5: 1 phase active)
	bendRegChargePointModel     = 142 // ChargePoint Model. Bytes 0 to 19.
	bendRegSmartVehicleDetected = 740 // Returns 1 if an EV currently connected is a smart vehicle, or 0 if no EV connected or it is not a smart vehicle

	// unused
	// bendRegChargedEnergyLegacy    = 705 // Sum of charged energy for the current session (Wh)
	// bendRegChargingDurationLegacy = 709 // Duration since beginning of charge (Seconds)
	// bendRegChargedEnergy          = 716 // Sum of charged energy for the current session (Wh)
	// bendRegChargingDuration       = 718 // Duration since beginning of charge (Seconds)

	powerLimit1pMennekes uint16 = 3725 // 207V * 3p * 6A - 1W
	powerLimit3pMennekes uint16 = 0xffff
)

func init() {
	registry.AddCtx("bender", NewBenderCCFromConfig)
}

// NewBenderCCFromConfig creates a BenderCC charger from generic config
func NewBenderCCFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		modbus.TcpSettings `mapstructure:",squash"`
		Cache              time.Duration
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 255, // default
		},
		Cache: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewBenderCC(ctx, cc.URI, cc.ID, cc.Cache)
}

// NewBenderCC creates BenderCC charger
//
//go:generate go tool decorate -f decorateBenderCC -b *BenderCC -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.Battery,Soc,func() (float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.ChargerEx,MaxCurrentMillis,func(float64) error" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"
func NewBenderCC(ctx context.Context, uri string, id uint8, cache time.Duration) (api.Charger, error) {
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
		regCurr: bendRegHemsCurrentLimit,
		log:     log,
	}

	// check legacy register set
	if _, err := wb.conn.ReadHoldingRegisters(bendRegChargePointModel, 10); err != nil {
		wb.legacy = true
	}

	var (
		currentPower     func() (float64, error)
		currents         func() (float64, float64, float64, error)
		voltages         func() (float64, float64, float64, error)
		totalEnergy      func() (float64, error)
		soc              func() (float64, error)
		identify         func() (string, error)
		maxCurrentMillis func(float64) error
		phases1p3p       func(int) error
		getPhases        func() (int, error)
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

	// check feature mA
	if _, err := wb.conn.ReadHoldingRegisters(bendRegHemsCurrentLimit10, 1); err == nil {
		maxCurrentMillis = wb.maxCurrentMillis
		wb.regCurr = bendRegHemsCurrentLimit10
	}

	// check feature modbus power control/1p3p fpr Mennekes 4you / 4business chargers
	if _, err := wb.conn.ReadHoldingRegisters(bendRegHemsPowerLimit, 1); err == nil {
		phases1p3p = wb.phases1p3pMennekes
		getPhases = wb.getPhasesMennekes
	}

	// check feature semp phase switching
	if phases1p3p == nil {
		if wb.supportsSEMPPhaseSwitching(uri, cache) {
			// set initial SEMP power limit to max so modbus control from 6 to 16 A is possible
			if err := wb.semp.conn.SendDeviceControl(wb.semp.deviceID, 0xffff); err == nil {
				phases1p3p = wb.phases1p3pSEMP
				getPhases = wb.getPhases
				// start heartbeat to keep connection alive
				go wb.heartbeat(ctx)
			} else {
				log.ERROR.Println("SEMP phase switching: could not set initial SEMP power limit:", err)
			}
		}
	}

	// check feature rfid
	if _, err := wb.identify(); err == nil {
		identify = wb.identify
	}

	return decorateBenderCC(wb, currentPower, currents, voltages, totalEnergy, soc, identify, maxCurrentMillis, phases1p3p, getPhases), nil
}

// heartbeat ensures that SEMP device control updates are sent about once per minute
func (wb *BenderCC) heartbeat(ctx context.Context) {
	for tick := time.Tick(5 * time.Second); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}
		if time.Since(wb.semp.conn.Updated()) >= time.Minute {
			// Send a very high power value to allow full control between 6 and 16A via modbus
			// Note: This will not trigger a phase switch, as the value is above the max. power consumption
			if err := wb.semp.conn.SendDeviceControl(wb.semp.deviceID, 0xffff); err != nil {
				wb.log.ERROR.Printf("heartbeat: failed to send update: %v", err)
			}
		}
	}
}

// supportsSEMPPhaseSwitching checks if SEMP phase switching is supported by querying device info
func (wb *BenderCC) supportsSEMPPhaseSwitching(uri string, cache time.Duration) bool {
	wb.semp.conn = semp.NewConnection(wb.log, "http://"+strings.Split(uri, ":")[0]+":8888/SimpleEnergyManagementProtocol")
	wb.semp.deviceG = util.ResettableCached(func() (semp.Device2EM, error) {
		return wb.semp.conn.GetDeviceXML()
	}, cache)

	doc, err := wb.semp.deviceG.Get()
	if err != nil {
		wb.log.DEBUG.Println("SEMP phase switching: cannot get XML", err)
		return false
	}
	if len(doc.DeviceInfo) == 0 {
		wb.log.DEBUG.Println("SEMP phase switching: no devices found")
		return false
	}

	// Use first device ID found
	wb.semp.deviceID = doc.DeviceInfo[0].Identification.DeviceID
	wb.log.DEBUG.Printf("SEMP phase switching: found device ID: %s", wb.semp.deviceID)

	// Check if device supports phase switching by checking power characteristics
	info, err := wb.getDeviceInfo()
	if err != nil {
		wb.log.DEBUG.Println("SEMP phase switching: cannot get device info:", err)
		return false
	}

	// Assume Phase switching support if MinPowerConsumption < 4140W and MaxPowerConsumption > 4600W
	if info.Characteristics.MinPowerConsumption > 0 && info.Characteristics.MinPowerConsumption < 4140 &&
		info.Characteristics.MaxPowerConsumption > 4600 {
		return true
	}

	wb.log.DEBUG.Println("SEMP phase switching: not supported")
	return false
}

// getDeviceInfo retrieves device info from cached document
func (wb *BenderCC) getDeviceInfo() (semp.DeviceInfo, error) {
	doc, err := wb.semp.deviceG.Get()
	if err != nil {
		return semp.DeviceInfo{}, err
	}

	for _, info := range doc.DeviceInfo {
		if info.Identification.DeviceID == wb.semp.deviceID {
			return info, nil
		}
	}

	return semp.DeviceInfo{}, fmt.Errorf("device %s not found in info response", wb.semp.deviceID)
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
	b, err := wb.conn.ReadHoldingRegisters(wb.regCurr, 1)
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

	_, err := wb.conn.WriteMultipleRegisters(wb.regCurr, 1, b)

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

// maxCurrentMillis implements the api.ChargerEx interface (Wallbe Firmware only)
func (wb *BenderCC) maxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	curr := uint16(current * 10) // 0.1A Steps

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, curr)

	_, err := wb.conn.WriteMultipleRegisters(bendRegHemsCurrentLimit10, 1, b)
	if err == nil {
		wb.current = curr
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

// phases1p3pMennekes implements the api.PhaseSwitcher interface for Mennekes AMTRON 4You / 4Business chargers
func (wb *BenderCC) phases1p3pMennekes(phases int) error {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, powerLimit3pMennekes)

	if phases == 1 {
		binary.BigEndian.PutUint16(b, powerLimit1pMennekes)
	}

	_, err := wb.conn.WriteMultipleRegisters(bendRegHemsPowerLimit, 1, b)

	return err
}

// getPhases implements the api.PhaseGetter interface for Mennekes AMTRON 4You / 4Business chargers
func (wb *BenderCC) getPhasesMennekes() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(bendRegHemsPowerLimit, 1)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint16(b) <= powerLimit1pMennekes {
		return 1, nil
	}

	return 3, nil
}

// phases1p3pSEMP implements the api.PhaseSwitcher interface via SEMP
func (wb *BenderCC) phases1p3pSEMP(phases int) error {
	// to switch to 3 phases, we have to uese a power value that is reachable with 3 phases
	// between 207 and 253V, but never with just 1 phase
	phaseSwitchPower := 9936 // 207V * 3p * 16A
	if phases == 1 {
		// to switch to 1 phase, we have to use a power value that is reachable with 1 phase
		// between 207 and 253V, but never with 3 phases
		phaseSwitchPower = 1518 // 253 * 1p * 6A
	}

	if err := wb.semp.conn.SendDeviceControl(wb.semp.deviceID, phaseSwitchPower); err != nil {
		return err
	}

	wb.semp.phases = phases

	wb.semp.deviceG.Reset()

	return nil
}

// getPhases implements the api.PhaseGetter interface for semp phase switching by reading the relay state through modbus
func (wb *BenderCC) getPhases() (int, error) {
	// check relay register
	b, err := wb.conn.ReadHoldingRegisters(bendRegRelayState, 1)
	if err != nil {
		return 0, err
	}

	if binary.BigEndian.Uint16(b) == 5 {
		return 1, nil
	}

	if binary.BigEndian.Uint16(b) == 1 {
		return 3, nil
	}

	return wb.semp.phases, nil
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

// soc implements the api.Battery interface
func (wb *BenderCC) soc() (float64, error) {
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
	if b, err := wb.conn.ReadHoldingRegisters(wb.regCurr, 1); err == nil {
		fmt.Printf("\tCurrent Limit:\t%d\n", binary.BigEndian.Uint16(b))
	}
}
