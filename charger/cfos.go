package charger

import (
	"encoding/binary"
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	cfosRegStatus     = 8092 // Input
	cfosRegMaxCurrent = 8093 // Holding
	cfosRegEnable     = 8094 // Coil
	cfosRegEnergy     = 8058 // energy reading
	cfosRegPower      = 8062 // power reading
	cfosRegMeter      = 8096 // has meter
)

var cfosRegCurrents = []uint16{8064, 8066, 8068} // current readings

// CfosPowerBrain is an api.ChargeController implementation for cFos PowerBrain wallboxes.
// It uses Modbus TCP to communicate at modbus client id 1 and power meters at id 2 and 3.
// https://www.cfos-emobility.de/en-gb/cfos-power-brain/modbus-registers.htm
type CfosPowerBrain struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("cfos", NewCfosPowerBrainFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateCfosPowerBrain -o cfos_decorators -b *CfosPowerBrain -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)"

// NewCfosPowerBrainFromConfig creates a cFos charger from generic config
func NewCfosPowerBrainFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		ID    uint8
		Meter struct {
			Power, Energy, Currents bool
		}
	}{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewCfosPowerBrain(cc.URI, cc.ID)
	if err != nil {
		return wb, err
	}

	ok, err := wb.hasMeter()
	if ok && err == nil {
		// var currentPower func() (float64, error)
		// if cc.Meter.Power {
		// 	currentPower = wb.currentPower
		// }

		// var totalEnergy func() (float64, error)
		// if cc.Meter.Energy {
		// 	totalEnergy = wb.totalEnergy
		// }

		// var currents func() (float64, float64, float64, error)
		// if cc.Meter.Currents {
		// 	currents = wb.currents
		// }

		// return decorateCfosPowerBrain(wb, wb.currentPower, wb.totalEnergy, wb.currents), err
		return decorateCfosPowerBrain(wb, wb.currentPower, wb.totalEnergy), err
	}

	return wb, err
}

// NewCfosPowerBrain creates a cFos charger
func NewCfosPowerBrain(uri string, id uint8) (*CfosPowerBrain, error) {
	uri = util.DefaultPort(uri, 4701)

	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.TcpFormat, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("cfos")
	conn.Logger(log.TRACE)

	wb := &CfosPowerBrain{
		conn: conn,
	}

	return wb, nil
}

func (wb *CfosPowerBrain) hasMeter() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegMeter, 1)
	if err != nil {
		return false, err
	}
	return b[0] == 1, nil
}

// Status implements the Charger.Status interface
func (wb *CfosPowerBrain) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	switch b[0] {
	case 0: // warten
		return api.StatusA, nil
	case 1: // Fahrzeug erkannt
		return api.StatusB, nil
	case 2: // laden
		return api.StatusC, nil
	case 3: // laden mit Kühlung
		return api.StatusD, nil
	case 4: // kein Strom
		return api.StatusE, nil
	case 5: // Fehler
		return api.StatusF, nil
	default:
		return api.StatusNone, errors.New("invalid response")
	}
}

// Enabled implements the Charger.Enabled interface
func (wb *CfosPowerBrain) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegEnable, 1)
	if err != nil {
		return false, err
	}

	return b[0] == 1, nil
}

// Enable implements the Charger.Enable interface
func (wb *CfosPowerBrain) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 1
	}

	_, err := wb.conn.WriteSingleCoil(cfosRegEnable, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *CfosPowerBrain) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*CfosPowerBrain)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *CfosPowerBrain) MaxCurrentMillis(current float64) error {
	_, err := wb.conn.WriteSingleRegister(cfosRegMaxCurrent, uint16(current*10))
	return err
}

// CurrentPower implements the api.Meter interface
func (wb *CfosPowerBrain) currentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)) / 10, err
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *CfosPowerBrain) totalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint64(b)), err
}

// currents implements the api.MeterCurrent interface
// not used as currents are only calculated from S0 meter
// func (wb *CfosPowerBrain) currents() (float64, float64, float64, error) {
// 	var currents []float64
// 	for _, regCurrent := range cfosRegCurrents {
// 		b, err := wb.conn.ReadHoldingRegisters(regCurrent, 2)
// 		if err != nil {
// 			return 0, 0, 0, err
// 		}

// 		currents = append(currents, float64(binary.BigEndian.Uint32(b))/10)
// 	}

// 	return currents[0], currents[1], currents[2], nil
// }
