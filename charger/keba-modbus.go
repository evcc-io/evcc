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

// Keba is an api.Charger implementation
type Keba struct {
	log  *util.Logger
	conn *modbus.Connection
}

const (
	kebaRegChargingState   = 1000
	kebaRegCableState      = 1004
	kebaRegCurrents        = 1008 // 6 regs, mA
	kebaRegSerial          = 1014 // leading zeros trimmed
	kebaRegProduct         = 1016
	kebaRegFirmware        = 1018
	kebaRegPower           = 1020 // mW
	kebaRegEnergy          = 1036 // Wh
	kebaRegVoltages        = 1040 // 6 regs, V
	kebaRegRfid            = 1500 // hex
	kebaRegSessionEnergy   = 1502 // Wh
	kebaRegPhaseSource     = 1550
	kebaRegPhaseState      = 1552
	kebaRegFailsafeTimeout = 1602
	kebaRegMaxCurrent      = 5004 // mA
	kebaRegEnable          = 5014
	kebaRegTriggerPhase    = 5052
)

func init() {
	registry.Add("keba-modbus", NewKebaFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateKeba -b *Keba -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewKebaFromConfig creates a new Keba ModbusTCP charger
func NewKebaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewKeba(cc.URI, cc.ID)
	if err != nil {
		return nil, err
	}

	// features
	var (
		currentPower, totalEnergy func() (float64, error)
		currents                  func() (float64, float64, float64, error)
		identify                  func() (string, error)
	)

	b, err := wb.conn.ReadHoldingRegisters(kebaRegProduct, 2)
	if err != nil {
		return nil, err
	}

	if features := binary.BigEndian.Uint32(b); (features/10)%10 > 0 {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
	}

	if features := binary.BigEndian.Uint32(b); features%10 > 0 {
		identify = wb.identify
	}

	// phases
	var phases func(int) error
	if b, err := wb.conn.ReadHoldingRegisters(kebaRegPhaseSource, 2); err == nil {
		if source := binary.BigEndian.Uint32(b); source == 3 {
			phases = wb.phases1p3p
		}
	}

	// failsafe
	b, err = wb.conn.ReadHoldingRegisters(kebaRegFailsafeTimeout, 2)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}

	if u := binary.BigEndian.Uint32(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Second / 2)
	}

	return decorateKeba(wb, currentPower, totalEnergy, currents, identify, phases), nil
}

// NewKeba creates a new charger
func NewKeba(uri string, slaveID uint8) (*Keba, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("keba")
	conn.Logger(log.TRACE)

	// per Keba docs
	// conn.Delay(500 * time.Millisecond)

	wb := &Keba{
		log:  log,
		conn: conn,
	}

	return wb, err
}

func (wb *Keba) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
		if _, err := wb.Enabled(); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Keba) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegCableState, 2)
	if err != nil {
		return api.StatusNone, err
	}

	switch status := binary.BigEndian.Uint32(b); status {
	case 0, 1, 3:
		return api.StatusA, nil

	case 5:
		return api.StatusB, nil

	case 7:
		b, err := wb.conn.ReadHoldingRegisters(kebaRegChargingState, 2)
		if err != nil {
			return api.StatusNone, err
		}
		if binary.BigEndian.Uint32(b) == 3 {
			return api.StatusC, err
		}
		return api.StatusB, nil

	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", status)
	}
}

// Enabled implements the api.Charger interface
func (wb *Keba) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(kebaRegChargingState, 2)
	if err != nil {
		return false, err
	}
	status := binary.BigEndian.Uint32(b)
	return !(status == 5 || status == 1), nil
}

// Enable implements the api.Charger interface
func (wb *Keba) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}
	_, err := wb.conn.WriteSingleRegister(kebaRegEnable, u)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Keba) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Keba)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *Keba) MaxCurrentMillis(current float64) error {
	u := uint16(current * 1000)
	_, err := wb.conn.WriteSingleRegister(kebaRegMaxCurrent, u)
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

	return float64(binary.BigEndian.Uint32(b)) / 1e4, nil
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
