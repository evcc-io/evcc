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
	enabled bool
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

	// write heartbeat once for command line testing
	if _, err := wb.conn.WriteSingleRegister(tqRegLifeBit, 1); err != nil {
		return nil, fmt.Errorf("heartbeat: %w", err)
	}

	// get failsafe timeout from charger
	b, err := wb.conn.ReadHoldingRegisters(tqRegComTimeout, 1)
	if err != nil {
		return nil, fmt.Errorf("failsafe timeout: %w", err)
	}
	if u := binary.BigEndian.Uint16(b); u > 0 {
		go wb.heartbeat(time.Duration(u) * time.Second / 2)
	}

	return wb, err
}

func (wb *WebastoNext) heartbeat(timeout time.Duration) {
	for range time.Tick(timeout) {
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
	case 0:
		return api.StatusA, nil
	case 1, 4:
		return api.StatusB, nil
	case 3:
		return api.StatusC, nil
	case 7:
		return api.StatusNone, fmt.Errorf("charging error: status %d", sb)
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", sb)
	}
}

// Enable implements the api.Charger interface
func (wb *WebastoNext) Enable(enable bool) error {
	b := make([]byte, 2)
	if enable {
		binary.BigEndian.PutUint16(b, wb.current)
	}

	_, err := wb.conn.WriteMultipleRegisters(tqRegChargeCurrent, 1, b)
	if err == nil {
		wb.enabled = enable
	}

	return err
}

// Enabled implements the api.Charger interface
func (wb *WebastoNext) Enabled() (bool, error) {
	return verifyEnabled(wb, wb.enabled)

	// b, err := wb.conn.ReadHoldingRegisters(1104, 1)
	// if err != nil {
	// 	return false, err
	// }

	// return binary.BigEndian.Uint16(b) > 0, nil
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

// ChargeDuration implements the api.ChargeTimer interface
func (wb *WebastoNext) ChargeDuration() (time.Duration, error) {
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

var _ api.PhaseCurrents = (*WebastoNext)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *WebastoNext) Currents() (float64, float64, float64, error) {
	var res [3]float64
	for l := uint16(0); l < 3; l++ {
		b, err := wb.conn.ReadInputRegisters(tqRegCurrents+2*l, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		res[l] = float64(binary.BigEndian.Uint16(b)) / 1e3
	}

	return res[0], res[1], res[2], nil
}

var _ api.Identifier = (*WebastoNext)(nil)

// Identify implements the api.Identifier interface
func (wb *WebastoNext) Identify() (string, error) {
	b, err := wb.conn.ReadHoldingRegisters(tqRegUserID, 10)
	if err != nil {
		return "", err
	}

	return bytesAsString(b), nil
}

var _ api.Diagnosis = (*WebastoNext)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *WebastoNext) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(tqRegSmartVehicleDetected, 1); err == nil {
		fmt.Printf("Smart vehicle:\t%t\n", binary.BigEndian.Uint16(b) != 0)
	}

	if b, err := wb.conn.ReadHoldingRegisters(tqRegUserID, 10); err == nil {
		fmt.Printf("UserID:\t%s\n", b)
	}

	if b, err := wb.conn.ReadHoldingRegisters(tqRegChargePointState, 1); err == nil {
		fmt.Printf("Charge point state:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1001, 1); err == nil {
		fmt.Printf("Charge state:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1002, 1); err == nil {
		fmt.Printf("Evse state:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1004, 1); err == nil {
		fmt.Printf("Cable state:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1102, 1); err == nil {
		fmt.Printf("Min current:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1100, 1); err == nil {
		fmt.Printf("Max current:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1104, 1); err == nil {
		fmt.Printf("Max evse current:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1106, 1); err == nil {
		fmt.Printf("Max cable current:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1108, 1); err == nil {
		fmt.Printf("Max ev current:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(tqRegComTimeout, 1); err == nil {
		fmt.Printf("Com timeout:\t%v\n", binary.BigEndian.Uint16(b))
	}

	if b, err := wb.conn.ReadHoldingRegisters(1006, 1); err == nil {
		fmt.Printf("Error code:\t%v\n", binary.BigEndian.Uint16(b))
	}
}
