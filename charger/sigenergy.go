package charger

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
)

// Sigenergy charger implementation
// Based on https://github.com/TypQxQ/Sigenergy-Local-Modbus/tree/main/custom_components/sigen
type Sigenergy struct {
	*embed
	log     *util.Logger
	conn    *modbus.Connection
	current uint16
}

const (
	// AC Charger Running Info Registers
	sigenACChargerSystemState         = 32000 // System states according to IEC61851-1 definition
	sigenACChargerTotalEnergyConsumed = 32001 // kWh, total energy consumed during charging
	sigenACChargerChargingPower       = 32003 // kW, instantaneous charging power
	
	// AC Charger Parameter Registers
	sigenACChargerOutputCurrent       = 42001 // Amperes, R/W, charger output current ([6, X] X is the smaller value between the rated current and the AC-Charger input breaker rated current.)
)

func init() {
	registry.AddCtx("sigenergy", NewSigenergyFromConfig)
}


// NewSigenergyFromConfig creates a new Sigenergy ModbusTCP charger
func NewSigenergyFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		embed              `mapstructure:",squash"`
		modbus.TcpSettings `mapstructure:",squash"`
	}{
		TcpSettings: modbus.TcpSettings{
			ID: 1,
		},
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSigenergy(ctx, cc.embed, cc.URI, cc.ID)
}

// NewSigenergy creates a new charger
func NewSigenergy(ctx context.Context, embed embed, uri string, slaveID uint8) (*Sigenergy, error) {
	conn, err := modbus.NewConnection(ctx, uri, "", "", 0, modbus.Tcp, slaveID)
	if err != nil {
		return nil, err
	}

	log := util.NewLogger("sigenergy")
	conn.Logger(log.TRACE)

	wb := &Sigenergy{
		embed: &embed,
		log:   log,
		conn:  conn,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Sigenergy) Status() (api.ChargeStatus, error) {
	b, err := wb.conn.ReadHoldingRegisters(sigenACChargerSystemState, 1)
	if err != nil {
		return api.StatusNone, err
	}

	state := binary.BigEndian.Uint16(b)

	// System states according to IEC61851-1 definition:
	// 0: Initializing
	// 1: Not Connected
	// 2: Reserving  
	// 3: Preparing
	// 4: EV Ready
	// 5: Charging
	// 6: Fault
	// 7: Error

	switch state {
	case 5: // Charging
		return api.StatusC, nil
	case 4: // EV Ready
		return api.StatusB, nil
	case 1: // Not Connected
		return api.StatusA, nil
	case 6, 7: // Fault, Error
		return api.StatusF, nil
	default: // Initializing, Reserving, Preparing
		return api.StatusB, nil
	}
}

// Enabled implements the api.Charger interface
func (wb *Sigenergy) Enabled() (bool, error) {
	b, err := wb.conn.ReadHoldingRegisters(sigenACChargerOutputCurrent, 2)
	if err != nil {
		return false, err
	}

	// U32 register with gain 100, check if != 0
	current := binary.BigEndian.Uint32(b)
	return current != 0, nil
}

// Enable implements the api.Charger interface
func (wb *Sigenergy) Enable(enable bool) error {
	var u uint32
	if enable {
		u = uint32(wb.current) * 100 // Apply gain 100
	} else {
		u = 0
	}

	// Write U32 value to register (2 registers)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, u)
	_, err := wb.conn.WriteMultipleRegisters(sigenACChargerOutputCurrent, 2, b)
	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *Sigenergy) MaxCurrent(current int64) error {
	curr := uint16(current)
	u := uint32(curr) * 100 // Apply gain 100
	
	// Write U32 value to register (2 registers)
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, u)
	_, err := wb.conn.WriteMultipleRegisters(sigenACChargerOutputCurrent, 2, b)
	if err == nil {
		wb.current = curr
	}

	return err
}

// CurrentPower implements the api.Meter interface
func (wb *Sigenergy) CurrentPower() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(sigenACChargerChargingPower, 2)
	if err != nil {
		return 0, err
	}

	// S32 register with gain 1000, convert directly to W
	return float64(int32(binary.BigEndian.Uint32(b))), nil
}

// TotalEnergy implements the api.MeterEnergy interface  
func (wb *Sigenergy) TotalEnergy() (float64, error) {
	b, err := wb.conn.ReadHoldingRegisters(sigenACChargerTotalEnergyConsumed, 2)
	if err != nil {
		return 0, err
	}

	// U32 register with gain 100, divide by 100 to get kWh
	return float64(binary.BigEndian.Uint32(b)) / 100, nil
}

var _ api.Meter = (*Sigenergy)(nil)
var _ api.MeterEnergy = (*Sigenergy)(nil)
var _ api.Diagnosis = (*Sigenergy)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Sigenergy) Diagnose() {
	if b, err := wb.conn.ReadHoldingRegisters(sigenACChargerSystemState, 1); err == nil {
		state := binary.BigEndian.Uint16(b)
		stateNames := []string{"Initializing", "Not Connected", "Reserving", "Preparing", "EV Ready", "Charging", "Fault", "Error"}
		stateName := "Unknown"
		if int(state) < len(stateNames) {
			stateName = stateNames[state]
		}
		fmt.Printf("\tSystem State:\t%d (%s)\n", state, stateName)
	}
	
	if b, err := wb.conn.ReadHoldingRegisters(sigenACChargerOutputCurrent, 2); err == nil {
		current := float64(binary.BigEndian.Uint32(b)) / 100
		fmt.Printf("\tOutput Current:\t%.1fA\n", current)
	}
	
	if b, err := wb.conn.ReadHoldingRegisters(sigenACChargerChargingPower, 2); err == nil {
		powerKW := float64(int32(binary.BigEndian.Uint32(b))) / 1000
		fmt.Printf("\tCharging Power:\t%.1fkW\n", powerKW)
	}
	
	if b, err := wb.conn.ReadHoldingRegisters(sigenACChargerTotalEnergyConsumed, 2); err == nil {
		energy := float64(binary.BigEndian.Uint32(b)) / 100
		fmt.Printf("\tTotal Energy:\t%.1fkWh\n", energy)
	}
}