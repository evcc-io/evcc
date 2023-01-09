package meter

import (
	"encoding/binary"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

const (
	cfosRegEnergy = 8058 // energy reading
	cfosRegPower  = 8062 // power reading
)

// var cfosRegCurrents = []uint16{8064, 8066, 8068} // current readings

// CfosPowerBrain is a meter implementation for cFos PowerBrain wallboxes.
// It uses Modbus TCP to communicate at modbus client id 1 and power meters at id 2 and 3.
// https://www.cfos-emobility.de/en-gb/cfos-power-brain/modbus-registers.htm
type CfosPowerBrain struct {
	conn *modbus.Connection
}

func init() {
	registry.Add("cfos", NewCfosPowerBrainFromConfig)
}

// NewCfosPowerBrainFromConfig creates a cFos meter from generic config
func NewCfosPowerBrainFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := modbus.TcpSettings{
		ID: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewCfosPowerBrain(cc.URI, cc.ID)
}

// NewCfosPowerBrain creates a cFos meter
func NewCfosPowerBrain(uri string, id uint8) (*CfosPowerBrain, error) {
	conn, err := modbus.NewConnection(uri, "", "", 0, modbus.Tcp, id)
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

// CurrentPower implements the api.Meter interface
func (wb *CfosPowerBrain) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegPower, 2)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint32(b)), err
}

var _ api.MeterEnergy = (*CfosPowerBrain)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *CfosPowerBrain) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(cfosRegEnergy, 4)
	if err != nil {
		return 0, err
	}

	return float64(binary.BigEndian.Uint64(b)) / 1e3, err
}

// var _ api.PhaseCurrents = (*CfosPowerBrain)(nil)

// // Currents implements the api.PhaseCurrents interface
// func (wb *CfosPowerBrain) Currents() (float64, float64, float64, error) {
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
