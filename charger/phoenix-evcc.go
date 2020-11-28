package charger

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/modbus"
)

const (
	phEVCCRegEnConfig   =  4000 // Holding, Remanent
	phEVCCRegEnable     = 20000 // Coil
	phEVCCRegOUT        = 23000 // Holding
	phEVCCRegMaxCurrent = 22000 // Holding
	phEVCCRegStatus     = 24000 // Input
)

// PhoenixEVCC is an api.ChargeController implementation for Phoenix EV-CC-AC1-M wallboxes.
// It uses Modbus TCP to communicate with the wallbox at modbus client id 255.
type PhoenixEVCC struct { 
	log *util.Logger 
	conn *modbus.Connection
}

func init() {
	registry.Add("phoenix-evcc", NewPhoenixEVCCFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -p charger -f decoratePhoenixEVCC -b api.Charger -o phoenix-evcc_decorators -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.MeterCurrent,Currents,func() (float64, float64, float64, error)"

// NewPhoenixEVCCFromConfig creates a Phoenix charger from generic config
func NewPhoenixEVCCFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		modbus.Settings `mapstructure:",squash"`
	}{
		Settings: modbus.Settings{
			ID: 255,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewPhoenixEVCC(cc.URI, cc.Device, cc.Comset, cc.Baudrate, cc.ID)
}

// NewPhoenixEVCC creates a Phoenix charger
func NewPhoenixEVCC(uri, device, comset string, baudrate int, id uint8) (*PhoenixEVCC, error) {
	conn, err := modbus.NewConnection(uri, device, comset, baudrate, true, id)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("evcc")
	conn.Logger(log.TRACE)

	wb := &PhoenixEVCC{ 
		log: log, 
		conn: conn,
	}

	return wb, nil
}

// Status implements the Charger.Status interface
func (wb *PhoenixEVCC) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadInputRegisters(phEVCCRegStatus, 1)
	if err != nil {
		return api.StatusNone, err
	}

	return api.ChargeStatus(string(b[0])), nil
}

// Enabled implements the Charger.Enabled interface
func (wb *PhoenixEVCC) Enabled() (bool, error) {
        var EN bool
	r, err := wb.conn.ReadInputRegisters(phEVCCRegEnConfig, 1)
        if err != nil {
                return false, err
        }
	if r[1] == 1 { 
 		b, err := wb.conn.ReadInputRegisters(phEVCCRegOUT, 1)
       		if err != nil {
        	        return false, err
 	       }
 	        wb.log.TRACE.Printf("Register 23000 is %d)", b)

		EN = b[1] == 1
	}else if r[1] == 3 {
		b, err := wb.conn.ReadCoils(phEVCCRegEnable, 1)
                if err != nil {
                        return false, err
               }
                wb.log.TRACE.Printf("Register 20000 is %d)", b)

                 EN = b[0] == 1
 	}else {
		wb.log.WARN.Printf("Register 4000 setting (%d) invalid",r[1])
	}
	if err != nil {
		return false, err
	}
	wb.log.TRACE.Printf("enabled: %t  (Register 4000 setting is %d)", EN, r[1])

	return EN, nil
}

// Enable implements the Charger.Enable interface
func (wb *PhoenixEVCC) Enable(enable bool) error {
	var u uint16
	if enable {
		u = 0x0001
	}
	//High-signal on pin OUT of the EV_CC_AC1-M  board (wire bridge between OUT and ENABLE necessary!!) 
	_, err := wb.conn.WriteSingleRegister(phEVCCRegOUT, u)

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (wb *PhoenixEVCC) MaxCurrent(current int64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %d", current)
	}

	_, err := wb.conn.WriteSingleRegister(phEVCCRegMaxCurrent, uint16(current))

	return err
}
