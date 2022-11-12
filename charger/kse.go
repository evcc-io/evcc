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

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// KSE charger implementation
type KSE struct {
	log  *util.Logger
	conn *modbus.Connection
	curr uint16
	rfid bool
}

const (
	kseRegSetMaxCurrent       = 0x03 // Externe Stromvorgabe via Bussystem / Ladefreigabe
	kseRegChargeMode          = 0x0E // Lademodus
	kseRegVehicleState        = 0x10 // State der Statemachine
	kseRegCurrents            = 0x14 // Phasenstrom (3)
	kseRegCurrentLoadedEnergy = 0x17 // Zwischen anstecken und abstecken geladene Energie (10 Wh)
	kseRegActualPower         = 0x18 // Aktuelle Ladeleistung (W)
	kseRegFirmwareVersion     = 0x30 // Firmware Version
	kseRegRFIDinstalled       = 0x31 // RFID-Leser vorhanden
	kseRegNFCTransactionID    = 0x67 // Tag ID (8 Bytes)
	// kseRegRelayMode           = 0x35 // Umschalten 1 phasiges oder 3 phasiges Laden

)

func init() {
	registry.Add("kse", NewKSEFromConfig)
}

// NewKSEFromConfig creates a KSE charger from generic config
func NewKSEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.Settings{
		ID:       100,
		Baudrate: 9600,
		Comset:   "8E1",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewKSE(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateKSE -b *KSE -r api.Charger -t "api.Identifier,Identify,func() (string, error)"

// NewKSE creates KSE charger
func NewKSE(uri, device, comset string, baudrate int, slaveID uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, modbus.Rtu, slaveID)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("kse")
	conn.Logger(log.TRACE)

	wb := &KSE{
		log:  log,
		conn: conn,
		curr: 6, // assume min current
	}

	// check presence of rfid
	b, err := wb.conn.ReadDiscreteInputs(kseRegRFIDinstalled, 1)
	if err == nil && b[0] != 0 {
		wb.rfid = true
		return decorateKSE(wb, wb.identify), err
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *KSE) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(kseRegVehicleState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	s := binary.BigEndian.Uint16(b)

	switch s {
	case 0, 1, 3:
		return api.StatusA, nil
	case 4:
		return api.StatusB, nil
	case 5:
		return api.StatusC, nil
	case 6, 7, 8:
		return api.StatusE, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *KSE) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(kseRegSetMaxCurrent, 1)
	if err != nil {
		return false, err
	}

	cur := binary.BigEndian.Uint16(b)

	return cur != 0, nil
}

// Enable implements the api.Charger interface
func (wb *KSE) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.curr
	}

	_, err := wb.conn.WriteSingleRegister(kseRegSetMaxCurrent, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *KSE) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(kseRegSetMaxCurrent, uint16(current))
	if err == nil {
		wb.curr = uint16(current)
	}

	return err
}

var _ api.Meter = (*KSE)(nil)

// CurrentPower implements the api.Meter interface
func (wb *KSE) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(kseRegActualPower, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), err
}

var _ api.ChargeRater = (*KSE)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *KSE) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(kseRegCurrentLoadedEnergy, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)) / 100, err
}

var _ api.MeterCurrent = (*KSE)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *KSE) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(kseRegCurrents, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var curr [3]float64
	for l := 0; l < 3; l++ {
		curr[l] = float64(binary.BigEndian.Uint16(b[2*l:])) / 1e3
	}

	return curr[0], curr[1], curr[2], nil
}

// var _ api.PhaseSwitcher = (*KSE)(nil)

// // Phases1p3p implements the api.PhaseSwitcher interface
// func (wb *KSE) Phases1p3p(phases int) error {
// 	var b uint16 = 0
// 	if phases == 1 {
// 		b = 1 // 1p
// 	}

// 	_, err := wb.conn.WriteSingleRegister(kseRegRelayMode, b)
// 	return err
// }

// Identify implements the api.Identifier interface
func (wb *KSE) identify() (string, error) {
	id, err := wb.conn.ReadHoldingRegisters(kseRegNFCTransactionID, 4)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(id)), nil
}

var _ api.Diagnosis = (*KSE)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *KSE) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(kseRegFirmwareVersion, 1); err == nil {
		fmt.Printf("\tFirmware:\t%d.%d\n", b[0], b[1])
	}
	if b, err := wb.conn.ReadInputRegisters(kseRegChargeMode, 1); err == nil {
		fmt.Printf("\tCharge Mode:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(kseRegVehicleState, 1); err == nil {
		fmt.Printf("\tState:\t%d\n", binary.BigEndian.Uint16(b))
	}
	if wb.rfid {
		if b, err := wb.conn.ReadHoldingRegisters(kseRegNFCTransactionID, 4); err == nil {
			fmt.Printf("\tNFC ID:\t%s\n", b)
		}
	}
}
