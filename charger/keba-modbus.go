package charger

// LICENSE

// Copyright (c) 2023 andig

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
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// https://www.keba.com/en/emobility/service-support/downloads/Downloads
// https://www.keba.com/download/x/dea7ae6b84/kecontactp30modbustcp_pgen.pdf
// https://www.keba.com/download/x/4a24e19f80/kecontactp40modbustcp_pgen.pdf

// Keba is an api.Charger implementation
type Keba struct {
	*embed
	log          *util.Logger
	conn         *modbus.Connection
	current      uint16
	regEnable    uint16
	energyFactor float64
}

const (
	kebaRegChargingState        = 1000
	kebaRegCableState           = 1004
	kebaRegCurrents             = 1008 // 6 regs, mA
	kebaRegSerial               = 1014 // leading zeros trimmed
	kebaRegProduct              = 1016
	kebaRegFirmware             = 1018
	kebaRegPower                = 1020 // mW
	kebaRegEnergy               = 1036 // Wh
	kebaRegVoltages             = 1040 // 6 regs, V
	kebaRegRfid                 = 1500 // hex
	kebaRegSessionEnergy        = 1502 // Wh
	kebaRegPhaseSource          = 1550
	kebaRegPhaseState           = 1552
	kebaRegFailsafeTimeout      = 1602
	kebaRegMaxCurrent           = 5004 // mA
	kebaRegEnable               = 5014
	kebaRegWriteFailsafeTimeout = 5018 //unit16!
	kebaRegTriggerPhase         = 5052
)

func init() {
	registry.AddCtx("keba-modbus", NewKebaFromConfig)
}

//go:generate go tool decorate -f decorateKeba -b *Keba -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.StatusReasoner,StatusReason,func() (api.Reason, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewKebaFromConfig creates a new Keba ModbusTCP charger
func NewKebaFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed              `mapstructure:",squash"`
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewKeba(ctx, cc.embed, cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	// optional features
	var (
		currentPower, totalEnergy func() (float64, error)
		currents                  func() (float64, float64, float64, error)
		identify                  func() (string, error)
		reason                    func() (api.Reason, error)
		phasesS                   func(int) error
		phasesG                   func() (int, error)
	)

	b, err := wb.conn.ReadHoldingRegisters(kebaRegProduct, 2)
	if err != nil {
		return nil, err
	}

	productCodeStr := fmt.Sprintf("%d", binary.BigEndian.Uint32(b))

	var hasEnergyMeter bool
	var hasRFID bool

	if len(productCodeStr) == 6 && productCodeStr[0] == '3' {
		// P30
		hasEnergyMeter = productCodeStr[4] != '0'
		hasRFID = productCodeStr[5] == '1'
	} else if len(productCodeStr) == 7 && productCodeStr[0] == '4' {
		// P40
		wb.regEnable = kebaRegMaxCurrent
		hasEnergyMeter = productCodeStr[4] != '0'
		hasRFID = productCodeStr[5] == '1'

		b, err := wb.conn.ReadHoldingRegisters(kebaRegFirmware, 2)
		if err != nil {
			return nil, err
		}

		// software version
		if binary.BigEndian.Uint32(b) < 10201 {
			// In software versions below 1.2.1 the registers 1502 and 1036
			// falsely report the value in “Wh” instead of “0.1 Wh”.
			wb.energyFactor = 1e3
		}
	}

	if hasEnergyMeter {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
	}

	if hasRFID {
		identify = wb.identify
		reason = wb.statusReason
	}

	// phases
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegPhaseSource, 2); err == nil {
		if source := binary.BigEndian.Uint32(b); source == 3 {
			phasesS = wb.phases1p3p
			phasesG = wb.getPhases
		}
	}

	// failsafe
	b, err = wb.conn.ReadHoldingRegisters(kebaRegFailsafeTimeout, 2)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}

	if u := binary.BigEndian.Uint32(b); u > 0 {
		go wb.heartbeat(ctx, u)
	}

	return decorateKeba(wb, currentPower, totalEnergy, currents, identify, reason, phasesS, phasesG), nil
}

// NewKeba creates a new charger
func NewKeba(ctx context.Context, embed embed, uri string, slaveID uint8) (*Keba, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("keba")
	conn.Logger(log.TRACE)

	wb := &Keba{
		embed:        &embed,
		log:          log,
		conn:         conn,
		regEnable:    kebaRegEnable,
		energyFactor: 1e4,
	}

	return wb, err
}

func (wb *Keba) heartbeat(ctx context.Context, u uint32) {
	timeout := time.Duration(u) * time.Second / 2

	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if _, err := wb.conn.WriteSingleRegister(kebaRegWriteFailsafeTimeout, uint16(u)); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *Keba) isConnected() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegCableState, 2)
	if err != nil {
		return false, err
	}

	// 0: No cable is plugged.
	// 1: Cable is connected to the charging station (not to the electric vehicle).
	// 3: Cable is connected to the charging station and locked (not to the electric vehicle).
	// 5: Cable is connected to the charging station and the electric vehicle (not locked).
	// 7: Cable is connected to the charging station and the electric vehicle and locked (charging).

	return binary.BigEndian.Uint32(b)&(1<<2) != 0, err
}

func (wb *Keba) getChargingState() (uint32, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegChargingState, 2)
	if err != nil {
		return 0, err
	}

	// 0: Start-up of the charging station
	// 1: The charging station is not ready for charging. The charging station is not connected to an electric vehicle, it is locked by the authorization function or another mechanism.
	// 2: The charging station is ready for charging and waits for a reaction from the electric vehicle.
	// 3: A charging process is active.
	// 4: An error has occurred.
	// 5: The charging process is temporarily interrupted because the temperature is too high or the wallbox is in suspended mode.

	return binary.BigEndian.Uint32(b), nil
}

// Status implements the api.Charger interface
func (wb *Keba) Status() (api.ChargeStatus, error) {
	if connected, err := wb.isConnected(); err != nil || !connected {
		return api.StatusA, err
	}

	s, err := wb.getChargingState()
	if err != nil {
		return api.StatusA, err
	}
	if s == 3 {
		return api.StatusC, nil
	}
	return api.StatusB, nil
}

// statusReason implements the api.StatusReasoner interface
func (wb *Keba) statusReason() (api.Reason, error) {
	if connected, err := wb.isConnected(); err != nil || !connected {
		return api.ReasonUnknown, err
	}

	if s, err := wb.getChargingState(); err != nil || s != 1 {
		return api.ReasonUnknown, err
	}

	return api.ReasonWaitingForAuthorization, nil
}

// Enabled implements the api.Charger interface
func (wb *Keba) Enabled() (bool, error) {
	// P40
	if wb.regEnable == kebaRegMaxCurrent {
		b, err := wb.conn.ReadHoldingRegisters(kebaRegMaxCurrent, 1)
		if err != nil {
			return false, err
		}
		return binary.BigEndian.Uint16(b) != 0, err
	}

	// P30
	s, err := wb.getChargingState()
	if err != nil {
		return false, err
	}

	return !(s == 5 || s == 1), nil
}

// Enable implements the api.Charger interface
func (wb *Keba) Enable(enable bool) error {
	var u uint16
	if enable {
		if wb.regEnable == kebaRegMaxCurrent {
			// P40
			u = wb.current
		} else {
			// P30
			u = 1
		}
	}

	_, err := wb.conn.WriteSingleRegister(wb.regEnable, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Keba) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Keba)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Keba) MaxCurrentMillis(current float64) error {
	curr := uint16(current * 1000)

	_, err := wb.conn.WriteSingleRegister(kebaRegMaxCurrent, curr)
	if err == nil {
		wb.current = curr
	}

	return err
}

// currentPower implements the api.Meter interface
func (wb *Keba) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Keba) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / wb.energyFactor, nil
}

// chargedEnergy is not supported since Keba does not reset it when plugging in a new car

// currents implements the api.PhaseCurrents interface
func (wb *Keba) currents() (float64, float64, float64, error) {
	var res [3]float64
	for i := range res {
		// does not support reading across register boundaries
		b, err := wb.conn.ReadHoldingRegisters(kebaRegCurrents+2*uint16(i), 2)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = float64(binary.BigEndian.Uint32(b)) / 1e3
	}

	return res[0], res[1], res[2], nil
}

// identify implements the api.Identifier interface
func (wb *Keba) identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegRfid, 2)
	if err != nil {
		return "", err
	}

	id := hex.EncodeToString(b)
	if id == "00000000" {
		id = ""
	}

	return id, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Keba) phases1p3p(phases int) error {
	var u uint16
	if phases == 3 {
		u = 1
	}

	_, err := wb.conn.WriteSingleRegister(kebaRegTriggerPhase, u)
	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *Keba) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegPhaseState, 2)
	if err != nil {
		return 0, err
	}
	if binary.BigEndian.Uint32(b) == 0 {
		return 1, nil
	}
	return 3, nil
}

var _ api.Diagnosis = (*Keba)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Keba) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegSerial, 2); err == nil {
		fmt.Printf("\tSerial:\t%s\n", strings.TrimLeft(strconv.Itoa(int(binary.BigEndian.Uint32(b))), "0"))
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegFirmware, 2); err == nil {
		fmt.Printf("\tFirmware:\t%d.%d.%d\n", b[0], b[1], b[2])
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegProduct, 2); err == nil {
		fmt.Printf("\tProduct:\t%6d\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegPhaseSource, 2); err == nil {
		fmt.Printf("\tPhases source:\t%d\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegPhaseState, 2); err == nil {
		fmt.Printf("\tPhases state:\t%d\n", binary.BigEndian.Uint32(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegFailsafeTimeout, 2); err == nil {
		fmt.Printf("\tFailsafe timeout:\t%ds\n", binary.BigEndian.Uint32(b))
	}
}
