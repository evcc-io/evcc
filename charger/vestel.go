package charger

// LICENSE

// Copyright (c) 2019-2022 andig

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
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/hashicorp/go-version"
)

const (
	vestelRegSerial          = 100 // 25
	vestelRegBrand           = 190 // 10
	vestelRegModel           = 210 // 5
	vestelRegFirmware        = 230 // 50
	vestelRegNumberPhases    = 404
	vestelRegPhasesSwitch    = 405
	vestelRegChargeStatus    = 1001
	vestelRegCableStatus     = 1004
	vestelRegChargeTime      = 1508
	vestelRegMaxCurrent      = 5004
	vestelRegPower           = 1020
	vestelRegTotalEnergy     = 1036
	vestelRegSessionEnergy   = 1502
	vestelRegRFID            = 1516
	vestelRegFailsafeTimeout = 2002
	vestelRegAlive           = 6000
	// vestelRegChargepointState = 1000
)

var (
	vestelRegCurrents = []uint16{1008, 1010, 1012} // non-continuous uint16 registers!
	vestelRegVoltages = []uint16{1014, 1016, 1018} // non-continuous uint16 registers!
)

// Vestel is an api.Charger implementation for Vestel/Hymes wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type Vestel struct {
	log     *util.Logger
	conn    *modbus.Connection
	enabled bool
	current uint16
}

func init() {
	registry.AddCtx("vestel", NewVestelFromConfig)
}

//go:generate go tool decorate -f decorateVestel -b *Vestel -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)" -t "api.Identifier,Identify,func() (string, error)"

// NewVestelFromConfig creates a Vestel charger from generic config
func NewVestelFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVestel(ctx, cc.URI, cc.ID)
}

// NewVestel creates a Vestel charger
func NewVestel(ctx context.Context, uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("vestel")
	conn.Logger(log.TRACE)

	wb := &Vestel{
		log:     log,
		conn:    conn,
		current: 6,
	}

	var (
		phasesS func(int) error
		phasesG func() (int, error)
	)
	if b, err := wb.conn.ReadInputRegisters(vestelRegNumberPhases, 1); err == nil && binary.BigEndian.Uint16(b) == 1 {
		phasesS = wb.phases1p3p
		phasesG = wb.getPhases
	}

	// compare firmware version to determine if RFID is available
	var identify func() (string, error)

	b, err := wb.conn.ReadInputRegisters(vestelRegFirmware, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to read firmware version: %w", err)
	}

	fw := strings.TrimPrefix(bytesAsString(b), "v")
	if v, err := version.NewSemver(fw); err == nil {
		if v.GreaterThanOrEqual(version.Must(version.NewSemver("3.156.0"))) {
			// firmware >= v3.156.0 supports RFID according to https://github.com/evcc-io/evcc/issues/21359
			identify = wb.identify
		}
	} else {
		log.WARN.Printf("failed to parse firmware version %q: %v", string(b), err)
	}

	// get failsafe timeout from charger
	b, err = wb.conn.ReadHoldingRegisters(vestelRegFailsafeTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	timeout := 5 * time.Second // 20s/4
	if u := binary.BigEndian.Uint16(b); u > 0 {
		timeout = time.Duration(u) * time.Second / 4
	}
	if timeout < time.Second {
		timeout = time.Second
	}
	go wb.heartbeat(ctx, timeout)

	return decorateVestel(wb, phasesS, phasesG, identify), err
}

func (wb *Vestel) heartbeat(ctx context.Context, timeout time.Duration) {
	for tick := time.Tick(timeout); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if _, err := wb.conn.WriteSingleRegister(vestelRegAlive, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *Vestel) Status() (api.ChargeStatus, error) {
	res := api.StatusA

	b, err := wb.conn.ReadInputRegisters(vestelRegCableStatus, 1)
	if err == nil && binary.BigEndian.Uint16(b) >= 2 {
		res = api.StatusB

		b, err = wb.conn.ReadInputRegisters(vestelRegChargeStatus, 1)
		if err == nil && binary.BigEndian.Uint16(b) == 1 {
			res = api.StatusC
		}
	}

	return res, err
}

// Enabled implements the api.Charger interface
func (wb *Vestel) Enabled() (bool, error) {
	return verifyEnabled(wb, wb.enabled)
}

// Enable implements the api.Charger interface
func (wb *Vestel) Enable(enable bool) error {
	var u uint16
	if enable {
		u = wb.current
	}

	_, err := wb.conn.WriteSingleRegister(vestelRegMaxCurrent, u)
	if err == nil {
		wb.enabled = enable
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Vestel) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current)
	_, err := wb.conn.WriteSingleRegister(vestelRegMaxCurrent, u)
	if err == nil {
		wb.current = u
	}

	return err
}

var _ api.CurrentGetter = (*Vestel)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (wb *Vestel) GetMaxCurrent() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(vestelRegMaxCurrent, 1)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), nil
}

var _ api.ChargeTimer = (*Vestel)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (wb *Vestel) ChargeDuration() (time.Duration, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegChargeTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.Meter = (*Vestel)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Vestel) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*Vestel)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Vestel) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegTotalEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

var _ api.ChargeRater = (*Vestel)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Vestel) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegSessionEnergy, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, err
}

// getPhaseValues returns 3 sequential register values
func (wb *Vestel) getPhaseValues(regs []uint16, divider float64) (float64, float64, float64, error) {
	var res [3]float64
	for i, reg := range regs {
		b, err := wb.conn.ReadInputRegisters(reg, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = float64(binary.BigEndian.Uint16(b)) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Vestel)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Vestel) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(vestelRegCurrents, 1e3)
}

var _ api.PhaseVoltages = (*Vestel)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Vestel) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(vestelRegVoltages, 1)
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *Vestel) phases1p3p(phases int) error {
	_, err := wb.conn.WriteSingleRegister(vestelRegPhasesSwitch, uint16((phases-1)>>1))
	return err
}

// getPhases implements the api.PhaseGetter interface
func (wb *Vestel) getPhases() (int, error) {
	b, err := wb.conn.ReadHoldingRegisters(vestelRegPhasesSwitch, 1)
	if err != nil {
		return 0, err
	}
	return 1 + int(binary.BigEndian.Uint16(b))<<1, nil
}

// Identify implements the api.Identifier interface
func (wb *Vestel) identify() (string, error) {
	b, err := wb.conn.ReadInputRegisters(vestelRegRFID, 15)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

var _ api.Diagnosis = (*Vestel)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Vestel) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(vestelRegBrand, 10); err == nil {
		fmt.Printf("Brand:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegModel, 5); err == nil {
		fmt.Printf("Model:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegSerial, 25); err == nil {
		fmt.Printf("Serial:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegFirmware, 50); err == nil {
		fmt.Printf("Firmware:\t%s\n", b)
	}
	if b, err := wb.conn.ReadHoldingRegisters(vestelRegFailsafeTimeout, 1); err == nil {
		fmt.Printf("Failsafe timeout:\t%#x\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadInputRegisters(vestelRegNumberPhases, 1); err == nil {
		fmt.Printf("Number of phases:\t%#x\n", binary.BigEndian.Uint16(b))
	}
	if b, err := wb.conn.ReadHoldingRegisters(vestelRegPhasesSwitch, 1); err == nil {
		fmt.Printf("Phase switch:\t%#x\n", binary.BigEndian.Uint16(b))
	}
}
