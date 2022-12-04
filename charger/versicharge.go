package charger

// LICENSE

// Copyright (c) 2019-2022 andig => Vertel.go Charger used as basis
//                                  Additional input from other EVCC Charger GO templates (e.g. ABB)
// Copyright (c) 2022 achgut/Flo56958 => Change and adpation to Versicharge Gen 3 Charger

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

//************************************************************************************

// Verwendete Versicharge GEN 3:
  // Versicharge GEN3 FW2.118 oder höher
  // Commercial Version (Reg 22 = 2), One Outlet: (Reg 24 = 1)
  // Integrated MID (Reg 30 = 4)
  // Order Number: 8EM1310-3EJ04-0GA0

  //https://support.industry.siemens.com/cs/attachments/109814359/versicharge_wallbox_modBus_map_en-US-FINAL.pdf

  // Gefundene Fehler:
    // Status Wallbox (A-F): Register 1601 nicht im ModbusMap dokumentiert. 
	// Active Power Phase Sum wird bei Strömen über 10A falsch berechnet (Register 1665)
    // daher Verwendung Apparent Power.
//************************************************************************************

// Steuerung Enable/Enabled durch Pause. Andere Variante über Current, siehe ABB?

// Weitere zukünfitge Themen zu implementieren / testen:

  // Laden 1/3 Phasen
  // 1 und 3 phasiges Laden implentmentiert aber funktioniert nicht // Wallbox reagiert nicht/stürzt ab?

  //RFID
  // Im ModbusTable fehlt das Register welche Karte freigegen wurde (zur Fahrzeugerkennung)
 	//  VersichargeRegRFIDEnable        = 79 // 1 RW disabled: 0 , enabled: 1	
	//	VersichargeRegRFIDCount         = 87 // 1  RO
    //	VersichargeRegRFID_UID0         = 88 // 5  RO
    //	VersichargeRegRFID_UID1         = 93 // 5  RO
    //	VersichargeRegRFID_UID2         = 97 // 5  RO
    //  weitere RFID Karten möglich (bis Register 337)

//  Failsafe Current und Timeout
    //  VersichargeRegFailsafeTimeout    = 1661 // RW 
    //  VersichargeRegFailsafeCurrentSum = 1660 // RW 

  // Time and Energy of charging session	 
    //  VersichargeRegSessionEnergy   = // derzeit nicht vorhanden im Modbus Table
    //	VersichargeRegChargeTime      = // derzeit nicht vorhanden im Modbus Table
	//  nur Total Energy (Gesamtladeleistung Wallbox) vorhanden

// Alive Check / Heartbeat Function (notwendig? aus ABB)
    //  VersichargeRegAlive           = // derzeit nicht vorhanden im Modbus Table

//************************************************************************************

import (
	"encoding/binary"
	"fmt"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
// Info Wallbox, nur Lesen
    VersichargeRegBrand             = 0    // 5   RO ASCII    -> Diagnose
	VersichargeRegProductionDate    = 5    // 2   RO UNIT16[] -> Diagnose
	VersichargeRegSerial            = 7    // 5   RO ASCII    -> Diagnose 
	VersichargeRegModel             = 12   // 10 RO ASCII     -> Diagnose
	VersichargeRegFirmware          = 31   // 10 RO ASCII     -> Diagnose
	VersichargeRegModbusTable       = 41   // 1  RO UINT16    -> Diagnose
	VersichargeRegRatedCurrent      = 28   // 1  RO UINT16    -> Diagnose
	VersichargeRegCurrentDipSwitch  = 29   // 1  RO UNIT16    -> Diagnose
	VersichargeRegMeterType         = 30   // 1  RO UINT16    -> Diagnose
	VersichargeRegTemp				= 1602 // 1  RO INT16     -> Diagnose

// Charger States / Settings / Steuerung
 	VersichargeRegRFIDEnable      =   79 // 1 RW UNIT16  -> disabled: 0 , enabled: 1	
    VersichargeRegChargeStatus    = 1601 // 1 RO INT16?? -> Status 1-5 nicht dokumentiert
    VersichargePause              = 1629 // 1 RW UNIT16  -> On: 1, Off: 2 - AN
    VersichargePhases             = 1642 // 1 RW UNIT16  -> 1Phase: 0 ; 3Phase: 1
	VersichargeRegMaxCurrent      = 1633 // 1 RW UNIT16  -> Max. Charging Current
 // VersichargeRegTotalEnergy     = 1692 // 2 RO Unit32(BigEndian) 
                                         // -> Gesamtleistung Wallbox in WattHours (Mulitplikation mit 0,1)
)

var (
	VersichargeRegCurrents = []uint16{1647, 1648, 1649, 1650}  // L1, L2, L3, SUM in AMP
	VersichargeRegVoltages = []uint16{1651, 1652, 1653}        // L1-N, L2-N, L3-N in V
//	VersichargeRegPower    = []uint16{1662, 1663, 1664, 1665}  // L1, L2, L3, SUM in Watt (Actual Power)
                                                               // SUM (Multiplikation mit 0,1)  
                                                               // WB bringt teilweise falschen Summenwert (bei >10A)
	VersichargeRegPower    = []uint16{1670, 1671, 1672, 1673}  // L1, L2, L3, SUM in Watt (Aparent Power)
)

// Versicharge is an api.Charger implementation for Versicharge wallboxes with Ethernet (SW modells).
// It uses Modbus TCP to communicate with the wallbox at modbus client id 1. 


type Versicharge struct {
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
}

func init() {
	registry.Add("versicharge", NewVersichargeFromConfig)
}

// NewVersichargeFromConfig creates a Versicharge charger from generic config
func NewVersichargeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewVersicharge(cc.URI, cc.ID)
}

// NewVersicharge creates a Versicharge charger
func NewVersicharge(uri string, id uint8) (*Versicharge, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("versicharge")
	conn.Logger(log.TRACE)

	wb := &Versicharge{
		log:     log,
		conn:    conn,
		current: 7,
	}

	return wb, nil
}


// ------------------------------------------------------------------------------------------------------
// Charger
// ------------------------------------------------------------------------------------------------------

// Status implements the api.Charger interface (Charging State A-F)
func (wb *Versicharge) Status() (api.ChargeStatus, error) {
	s, err := wb.conn.ReadHoldingRegisters(VersichargeRegChargeStatus, 1)
		if err != nil {
		return api.StatusNone, err
	}

	switch binary.BigEndian.Uint16(s) {
	case 1: // State A: Idle, Power on
		return api.StatusA, nil
	case 2: // State B: EV Plug in, pending authorization
		return api.StatusB, nil
	case 3: // Charging
		return api.StatusC, nil
	case 4: // Charging? kommt nur kurzzeitg beim Starten, dann Rückfall auf 3
		return api.StatusC, nil
	case 5: // Other: Session stopped (Pause) 
	        //weitere Option, noch zu implementieren: Test auf ChargingStrom -> C, bei 0 -> B
		b, err := wb.conn.ReadHoldingRegisters(VersichargePause, 1) // Abfrage Pausiert?
		if err != nil {
			return api.StatusNone, err
		}
		if binary.BigEndian.Uint16(b) == 0x1 {  //Pause ON
			return api.StatusB, nil
		}
	default: // Other
		return api.StatusNone, fmt.Errorf("invalid status: %0x", s)
	}
	return api.StatusNone, err
}

// Enabled implements the api.Charger interface -> Über Pause
func (wb *Versicharge) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(VersichargePause, 1)
	if err != nil {
		return false, err
	}

	return binary.BigEndian.Uint16(b) == 2, nil
}

// Enable implements the api.Charger interface
// Enable mit Einstellung auf MinCurrent sinnvoll?
func (wb *Versicharge) Enable(enable bool) error {
    var u uint16
	u = 1
	if enable == true {
		u = 2
		}
	_, err := wb.conn.WriteSingleRegister(VersichargePause, u)

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Versicharge) MaxCurrent(current int64) error {
	if current < 7 {
		return fmt.Errorf("invalid current %d", current)
	}

	u := uint16(current)
	_, err := wb.conn.WriteSingleRegister(VersichargeRegMaxCurrent, u)
	if err == nil {
		wb.current = u
	}

	return err
}

// var _ api.PhaseSwitcher = (*Versicharge)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface 1Phase: 0 ; 3Phase: 1
// func (wb *Versicharge) Phases1p3p(phases int) error {
// 	fmt.Printf("%d Phases \n", phases)
// 	
// 	if phases == 1 {
// 		_, err := wb.conn.WriteSingleRegister(VersichargePhases, uint16(0)) // 1 Phase = 0
// 		return err
// 	}
// 
// 	if phases == 3 {
// 		_, err := wb.conn.WriteSingleRegister(VersichargePhases, uint16(1)) // 3 Phasen = 1
// 		return err
// 	}
// 
// 	return nil 
// }

// ------------------------------------------------------------------------------------------------------
// Meter
// ------------------------------------------------------------------------------------------------------
var _ api.Meter = (*Versicharge)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Versicharge) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(VersichargeRegPower[3], 1)
	if err != nil {
	  return 0, err
	}

	return float64(binary.BigEndian.Uint16(b)), err 
}

var _ api.MeterCurrent = (*Versicharge)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Versicharge) Currents() (float64, float64, float64, error) {
	var currents []float64
	for _, regCurrent := range VersichargeRegCurrents {
		b, err := wb.conn.ReadHoldingRegisters(regCurrent, 1)
		if err != nil {
			return 0, 0, 0, err
		}

		currents = append(currents, float64(binary.BigEndian.Uint16(b))) // in Ampere
	}

	fmt.Printf("%f %f %f %f Currents \n", currents[0],currents[1],currents[2],currents[3])	
	return currents[0], currents[1], currents[2], nil
}

// ------------------------------------------------------------------------------------------------------
// Diagnoses
// ------------------------------------------------------------------------------------------------------

var _ api.Diagnosis = (*Versicharge)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Versicharge) Diagnose() {

	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegBrand, 5); err == nil {
		fmt.Printf("Brand:\t%s\n", b) // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegModel, 10); err == nil {
		fmt.Printf("Model:\t%s\n", b) // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegSerial, 5); err == nil {
		fmt.Printf("Serial:\t%s\n", b) // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegFirmware, 5); err == nil {
		fmt.Printf("Firmware:\t%s\n", b) // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegProductionDate, 2); err == nil {
		fmt.Printf("Production Date:\t%s\n", b)  // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegModbusTable, 1); err == nil {
		fmt.Printf("Modbus Table:\t%s\n", b)  // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegRatedCurrent, 1); err == nil {
		fmt.Printf("Rated Current:\t%s\n", b)  // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegCurrentDipSwitch, 1); err == nil {
		fmt.Printf("Current (DIP Switch):\t%s\n", b)  // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegMeterType, 1); err == nil {
		fmt.Printf("Meter Type:\t%s\n", b)  // Ausgabeformat bearbeiten?
	}
	if b, err := wb.conn.ReadHoldingRegisters(VersichargeRegTemp, 1); err == nil {
	    fmt.Printf("Temperature PCB (°C):\t%s\n", b)  // Ausgabeformat bearbeiten?
    }
}