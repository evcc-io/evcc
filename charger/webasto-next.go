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
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
)

// WebastoNext charger implementation
type WebastoNext struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
}

const (
	// all holding type registers
	tqRegChargePointState     = 1000 // State of the charging device
	tqRegCurrents             = 1008 // Charging current (mA)
	tqRegActivePower          = 1020 // Sum of active charging power (W)
	tqRegEnergyMeter          = 1036 // Meter reading of the charging station (Wh)
	tqRegChargingTime         = 1508 // Duration since beginning of charge (Seconds)
	tqRegUserID               = 1600 // User ID (OCPP IdTag) from the current session. Bytes 0 to 19.
	tqRegSmartVehicleDetected = 1620 // Returns 1 if an EV currently connected is a smart vehicle, or 0 if no EV connected or it is not a smart vehicle
	tqRegComTimeout           = 2002 // Communication timeout
	tqRegChargeCurrent        = 5004 // (A)
	tqRegLifeBit              = 6000 // Communication monitoring 0/1 Toggle-Bit

	tqTimeout = 60
)

func init() {
	registry.Add("webasto-next", NewWebastoNextFromConfig)
}

// NewWebastoNextFromConfig creates a WebastoNext charger from generic config
func NewWebastoNextFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 255,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewWebastoNext(cc.URI, cc.ID)
}

// NewWebastoNext creates WebastoNext charger
func NewWebastoNext(uri string, id uint8) (api.Charger, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	log := util.NewLogger("webasto-next")
	conn.Logger(log.TRACE)

	wb := &WebastoNext{
		log:     log,
		conn:    conn,
		current: 6, // assume min current
	}

	// get failsafe timeout
	b := make([]byte, 2)
	if b, err = wb.conn.ReadHoldingRegisters(tqRegComTimeout, 1); err != nil {
		return nil, fmt.Errorf("could not get failsafe timeout: %v", err)
	}

	go wb.heartbeat(time.Duration(binary.BigEndian.Uint16(b)/2-1) * time.Second)

	return wb, err
}

func (wb *WebastoNext) heartbeat(timeout time.Duration) {
	for range time.NewTicker(timeout).C {
		if _, err := wb.conn.WriteSingleRegister(tqRegLifeBit, 1); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

// Status implements the api.Charger interface
func (wb *WebastoNext) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegChargePointState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	sb := binary.BigEndian.Uint16(b)

	switch sb {
	case 0, 8:
		return api.StatusA, nil
	case 1, 4:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 7:
		return api.StatusE, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", sb)
	}
}

// Enabled implements the api.Charger interface
func (wb *WebastoNext) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegChargeCurrent, 1)
	if err != nil {
		return false, err
	}

	cur := binary.BigEndian.Uint16(b)

	return cur != 0, nil
}

// Enable implements the api.Charger interface
func (wb *WebastoNext) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(tqRegChargeCurrent, 1, b)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *WebastoNext) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(current))

	_, err := wb.conn.WriteMultipleRegisters(tqRegChargeCurrent, 1, b)
	if err == nil {
		wb.current = uint16(current)
	}

	return err
}

var _ api.ChargeTimer = (*WebastoNext)(nil)

// ChargingTime implements the api.ChargeTimer interface
func (wb *WebastoNext) ChargingTime() (time.Duration, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegChargingTime, 2)
	if err != nil {
		return 0, err
	}

	return time.Duration(binary.BigEndian.Uint32(b)) * time.Second, nil
}

var _ api.Meter = (*WebastoNext)(nil)

// CurrentPower implements the api.Meter interface
func (wb *WebastoNext) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegActivePower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), nil
}

var _ api.MeterEnergy = (*WebastoNext)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WebastoNext) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegEnergyMeter, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 1e3, nil
}

var _ api.MeterCurrent = (*WebastoNext)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *WebastoNext) Currents() (float64, float64, float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegCurrents, 3)
	if err != nil {
		return 0, 0, 0, err
	}

	var curr [3]float64
	for l := 0; l < 3; l++ {
		curr[l] = float64(binary.BigEndian.Uint16(b[2*l:2*(l+1)])) / 1e3
	}

	return curr[0], curr[1], curr[2], nil
}

var _ api.Identifier = (*WebastoNext)(nil)

// Identify implements the api.Identifier interface
func (wb *WebastoNext) Identify() (string, error) {
	id, err := wb.conn.ReadHoldingRegisters(tqRegUserID, 10)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(id)), nil
}

var _ api.Diagnosis = (*WebastoNext)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *WebastoNext) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(tqRegSmartVehicleDetected, 1); err == nil {
		fmt.Printf("\tSmart Vehicle:\t%t\n", binary.BigEndian.Uint16(b) != 0)
	}

	if b, err := wb.conn.ReadHoldingRegisters(tqRegUserID, 10); err == nil {
		fmt.Printf("\tUserID:\t%s\n", b)
	}
}
