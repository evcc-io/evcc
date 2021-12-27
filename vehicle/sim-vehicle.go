package vehicle

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type SimVehicleCfg struct {
	Capacity      int64    // [kWh]
	PowerLimit    float64  // [W]
	SoC           float64  // [%]
	Title         string   // title of this vehicle
	Bidirectional bool     // true when bidirectional charging is supported
	Identifiers   []string // list of identifiers for this vehicle
}

type SimVehicle struct {
	log           *util.Logger
	name          string
	simType       api.SimType      // type of the sim Sim_grid, Sim_battery, ...
	capacitykWh   int64            // [kWh] for vehicle battery
	chargekWh     float64          // [kWh] for vehicle battery
	powerLimitW   float64          // [W] for battery
	soc           float64          // [%] for battery
	powerW        float64          // [w]
	lastUpdate    time.Time        // last update Timestamp for vehicle battery
	title         string           // title of the vehicle
	identifiers   []string         // array of identifiers for this vehicle
	onIdentified  api.ActionConfig // on identified action
	bidirectional bool             // true when the vehicle supports bidirectional charging
}

func init() {
	registry.Add("sim-vehicle", NewSimVehicleFromConfig)
}

// NewSimVehicleFromConfig returns a vehicle instance generated from the configuration
func NewSimVehicleFromConfig(other map[string]interface{}) (api.Vehicle, error) {

	cfg := &SimVehicleCfg{
		Capacity:      0,
		PowerLimit:    0,
		SoC:           0,
		Title:         "untitled",
		Bidirectional: false,
		Identifiers:   []string{""},
	}

	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, err
	}

	return NewSimVehicle(cfg)
}

// NewSimVehicle creates a vehicle instance
func NewSimVehicle(cfg *SimVehicleCfg) (api.Vehicle, error) {
	trace := util.NewLogger("vhcl")
	trace.INFO.Println("SimVehicle create")
	v := &SimVehicle{
		log:           trace,
		name:          "unnamed",
		simType:       api.Sim_vehicle,
		capacitykWh:   cfg.Capacity,
		powerLimitW:   cfg.PowerLimit,
		soc:           cfg.SoC,
		chargekWh:     (float64(cfg.Capacity) * cfg.SoC) / 100,
		title:         cfg.Title,
		bidirectional: cfg.Bidirectional,
		identifiers:   cfg.Identifiers,
	}
	return v, nil
}

// SimType gives the simulation type of the meter - e.g. Sim_grid
func (v *SimVehicle) SimType() (api.SimType, error) {
	return v.simType, nil
}

// SetName sets the name of the sim - used mainly for logging
func (v *SimVehicle) SetName(name string) error {
	v.name = name
	return nil
}

// Name gets the name of the sim
func (v *SimVehicle) Name() (string, error) {
	return v.name, nil
}

// CurrentPower provides the current power in [W]
// charging: positive sign, discharging: negative sign
func (v *SimVehicle) CurrentPower() (float64, error) {
	return v.powerW, nil
}

// SetCurrentPower sets the current power in [W]
// positive sign: charge vehickle, negative sign: discharge vehicle
func (v *SimVehicle) SetCurrentPower(powerW float64) error {
	v.powerW = powerW
	return nil
}

// SoC provides the SoC of the battery (if this meter represents a battery) in [%]
func (v *SimVehicle) SoC() (float64, error) {
	return v.soc, nil
}

// SetSoC sets the current SoC of the battery (if this meter represents a battery) in [%]
func (v *SimVehicle) SetSoC(soc float64) error {
	v.soc = soc
	v.chargekWh = float64(v.capacitykWh) * v.soc / 100
	return nil
}

// UpdateSoc recalculates the SoC of the vehicle battery
// returns the power [W] it charges (positive sign) or discharges (negative sign)
// in the next update cycle
func (v *SimVehicle) UpdateSoC(availablePowerW float64) (float64, error) {
	if v.lastUpdate.IsZero() {
		v.lastUpdate = time.Now()
	}
	elapsedTime := time.Since(v.lastUpdate)
	// add charged energy - convert from Ws to kWh - dividing by 3.6E6
	v.chargekWh = v.chargekWh + (v.powerW*elapsedTime.Seconds())/(3.6e6)
	if v.chargekWh > float64(v.capacitykWh) {
		// battery fully charged - stop charging
		v.chargekWh = float64(v.capacitykWh)
		v.powerW = 0
	} else if v.chargekWh < 0 {
		// battery fully discharged - stop discharging
		v.chargekWh = 0
		v.powerW = 0
	}
	v.lastUpdate = time.Now()
	if v.capacitykWh > 0 {
		v.soc = (v.chargekWh / float64(v.capacitykWh)) * 100
	} else {
		v.soc = 0
	}

	// calculate charge/discharge power for next update cycle
	if v.soc < 100 && availablePowerW > 0 {
		// charge battery: positive power sign
		v.powerW = math.Min(availablePowerW, v.powerLimitW)
	} else if v.soc > 0 && availablePowerW < 0 && v.bidirectional {
		// discharge battery: negative power sign
		v.powerW = -(math.Min(-availablePowerW, v.powerLimitW))
	} else if availablePowerW == 0 {
		v.powerW = 0
	}

	v.log.DEBUG.Printf("%s: Charge[kWh]:%f, SoC[%%]:%f Power[W]:%f", v.name, v.chargekWh, v.soc, v.powerW)
	return v.powerW, nil
}

// SetCapacity sets the capacity of the vehicle in [kWh]
func (v *SimVehicle) SetCapacity(capacitykWh int64) error {
	v.capacitykWh = capacitykWh
	return nil
}

// Capacity gives the capacity of the vehicle in [kWh]
func (v *SimVehicle) Capacity() int64 {
	return v.capacitykWh
}

// SetPowerLimit sets the PowerLimit of the battery (if this meter represents a bettery) in [W]
func (v *SimVehicle) SetPowerLimit(powerLimitW float64) error {
	v.powerLimitW = powerLimitW
	return nil
}

// PowerLimit gives the PowerLimit of the battery (if this meter represents a battery) in [W]
func (v *SimVehicle) PowerLimit() (float64, error) {
	return v.powerLimitW, nil
}

// Title gives the title of the vehicle
func (v *SimVehicle) Title() string {
	return v.title
}

// SetTitle sets the title of the vehicle
func (v *SimVehicle) SetTitle(title string) error {
	v.title = title
	return nil
}

// Identifiers gives the list of identifiers for this vehicle
func (v *SimVehicle) Identifiers() []string {
	return v.identifiers
}

// SetIdentifiers sets the list of identifiers for this vehicle
func (v *SimVehicle) SetIdentifiers(identifiers []string) error {
	v.identifiers = identifiers
	return nil
}

// OnIdentified gives the actionConfig for the onIdentified event
func (v *SimVehicle) OnIdentified() api.ActionConfig {
	v.log.DEBUG.Printf("%s: OnIdentified", v.name)
	return v.onIdentified
}

// OnIdentified gives the actionConfig for the onIdentified event
func (v *SimVehicle) SetOnIdentified(onIdentified api.ActionConfig) error {
	v.onIdentified = onIdentified
	return nil
}

// Bidirectional returns true when the vehicle supports bidirectional charging
func (v *SimVehicle) Bidirectional() (bool, error) {
	return v.bidirectional, nil
}

// SetBidirectional sets the bidirectional charging capability of the vehicle
func (v *SimVehicle) SetBidirectional(isBidirectional bool) error {
	v.bidirectional = isBidirectional
	return nil
}
