package charger

// LICENSE

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Details on the Peblar modbus server obtained from: https://developer.peblar.com/modbus-api

import (
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/volkszaehler/mbmd/encoding"
)

// Peblar charger implementation
type Peblar struct {
	log     *util.Logger
	conn    *modbus.Connection
	curr    uint32
	enabled bool
	phases  uint16
}

const (
	// Meter addresses
	peblarEnergyTotalAddress   = 30000
	peblarSessionEnergyAddress = 30004
	peblarPowerPhase1Address   = 30008
	peblarPowerPhase2Address   = 30010
	peblarPowerPhase3Address   = 30012
	peblarPowerTotalAddress    = 30014
	peblarVoltagePhase1Address = 30016
	peblarVoltagePhase2Address = 30018
	peblarVoltagePhase3Address = 30020
	peblarCurrentPhase1Address = 30022
	peblarCurrentPhase2Address = 30024
	peblarCurrentPhase3Address = 30026

	// Config addresses
	peblarSerialNumberAddress  = 30050
	peblarProductNumberAddress = 30062
	peblarFwIdentifierAddress  = 30074
	peblarPhaseCountAddress    = 30092
	peblarIndepRelayAddress    = 30093

	// Control addresses
	peblarCurrentLimitSourceAddress = 30112
	peblarCurrentLimitActualAddress = 30113
	peblarModbusCurrentLimitAddress = 40000
	peblarForce1PhaseAddress        = 40002

	// Diagnostic addresses
	peblarCpStateAddress = 30110
)

func init() {
	registry.Add("peblar", NewPeblarFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decoratePeblar -b *Peblar -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewPeblarFromConfig creates a Peblar charger from generic config
func NewPeblarFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPeblar(cc.URI, cc.ID)
}

// NewPeblar creates Peblar charger
func NewPeblar(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("peblar")
	conn.Logger(log.TRACE)

	// Register contains the physically connected phases
	b, err := conn.ReadInputRegisters(peblarPhaseCountAddress, 1)
	if err != nil {
		return nil, err
	}
	log.DEBUG.Println("detected connected phases:", binary.BigEndian.Uint16(b))

	wb := &Peblar{
		log:    log,
		conn:   conn,
		curr:   6000, // assume min current
		phases: binary.BigEndian.Uint16(b),
	}

	c, err := conn.ReadInputRegisters(peblarIndepRelayAddress, 1)
	if err != nil {
		log.DEBUG.Println("failed to read independent relays register")
		return nil, err
	}
	var indepRelays uint16 = binary.BigEndian.Uint16(c)

	var phasesS func(int) error

	if indepRelays == 0 {
		log.DEBUG.Println("detected 1x4 or 1x2-pole relay")
	} else {
		log.DEBUG.Println("detected 2x2-pole relay")
		phasesS = wb.phases1p3p
	}

	return decoratePeblar(wb, phasesS), err
}

// Status implements the api.Charger interface
func (wb *Peblar) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(peblarCpStateAddress, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := encoding.Uint16(b); s {
	case 'A':
		return api.StatusA, nil
	case 'B':
		return api.StatusB, nil
	case 'C':
		return api.StatusC, nil
	case 'D':
		return api.StatusD, nil
	case 'E':
		return api.StatusE, nil
	case 'F':
		return api.StatusF, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Peblar) Enabled() (bool, error) {
	return verifyEnabled(wb, wb.enabled)
}

// Enable implements the api.Charger interface
func (wb *Peblar) Enable(enable bool) error {
	var current uint32
	if enable {
		current = wb.curr
	}

	return wb.setCurrent(current)
}

// setCurrent writes the current limit in mA
func (wb *Peblar) setCurrent(current uint32) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, current)

	_, err := wb.conn.WriteMultipleRegisters(peblarModbusCurrentLimitAddress, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Peblar) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Peblar)(nil)

// MaxCurrent implements the api.ChargerEx interface
func (wb *Peblar) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.1f", current)
	}

	wb.curr = uint32(current * 1e3)

	return wb.setCurrent(wb.curr)
}

var _ api.Meter = (*Peblar)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Peblar) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(peblarPowerTotalAddress, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.ChargeRater = (*Peblar)(nil)

// ChargedEnergy implements the api.MeterEnergy interface
func (wb *Peblar) ChargedEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(peblarSessionEnergyAddress, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int64(b)) / 1e3, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Peblar) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadInputRegisters(peblarEnergyTotalAddress, 4)
	if err != nil {
		return 0, err
	}

	return float64(encoding.Int64(b)) / 1e3, nil
}

// getPhaseValues returns 1..3 sequential register values
func (wb *Peblar) getPhaseValues(reg uint16, divider float64) (float64, float64, float64, error) {
	b, err := wb.conn.ReadInputRegisters(reg, wb.phases*2)
	if err != nil {
		return 0, 0, 0, err
	}

	var res [3]float64
	for i := 0; i < int(wb.phases); i++ {
		res[i] = float64(binary.BigEndian.Uint32(b[4*i:])) / divider
	}

	return res[0], res[1], res[2], nil
}

var _ api.PhaseCurrents = (*Peblar)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Peblar) Currents() (float64, float64, float64, error) {
	return wb.getPhaseValues(peblarCurrentPhase1Address, 1e3)
}

var _ api.PhaseVoltages = (*Peblar)(nil)

// Voltages implements the api.PhaseVoltages interface
func (wb *Peblar) Voltages() (float64, float64, float64, error) {
	return wb.getPhaseValues(peblarVoltagePhase1Address, 1)
}

// phases1p3p implements the api.PhaseSwitcher interface via the decorator
func (wb *Peblar) phases1p3p(phases int) error {
	var b uint16 = 0

	wb.log.DEBUG.Println("attempt to change phases to:", phases)

	if phases == 1 {
		b = 1
	}

	_, err := wb.conn.WriteSingleRegister(peblarForce1PhaseAddress, b)
	return err
}

var _ api.PhaseGetter = (*Peblar)(nil)

// GetPhases implements the api.PhaseGetter interface
func (wb *Peblar) GetPhases() (int, error) {
	c, err := wb.conn.ReadHoldingRegisters(peblarForce1PhaseAddress, 1)
	if err != nil {
		return 0, err
	}

	var force1p uint16 = binary.BigEndian.Uint16(c)
	wb.log.DEBUG.Println("forced to 1 phase:", force1p)
	var phases uint16 = wb.phases
	wb.log.DEBUG.Println("connected phases:", phases)

	if wb.phases > 1 && force1p == 1 {
		phases = 1
	}

	return int(phases), nil
}

var _ api.Diagnosis = (*Peblar)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Peblar) Diagnose() {
	if b, err := wb.conn.ReadInputRegisters(peblarSerialNumberAddress, 12); err == nil {
		fmt.Printf("\tSN:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(peblarProductNumberAddress, 12); err == nil {
		fmt.Printf("\tPN:\t%s\n", b)
	}
	if b, err := wb.conn.ReadInputRegisters(peblarFwIdentifierAddress, 12); err == nil {
		fmt.Printf("\tFirmware:\t%s\n", b)
	}
}
