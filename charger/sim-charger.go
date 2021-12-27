package charger

import (
	"fmt"
	"math"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	voltage1p float64 = 230.0
	voltage3p float64 = 400.0
)

type SimChargerCfg struct {
	ChargeState api.ChargeStatus // charger status
	Switch1p3p  string           // 1p3p switch for this charger
}

type SimCharger struct {
	log            *util.Logger        // logger for this component
	name           string              // name of the charger in the sim program
	simType        api.SimType         // type of the sim Sim_grid, Sim_battery, ...
	powerW         float64             // [w] positive sign: vehicle is charged
	status         api.ChargeStatus    // A-F
	enabled        bool                // true when enabled
	maxCurrent     int64               // [A]
	vehicle        api.SimVehicle      // nil: no vehicle connected, otherwise: vehicle reference
	switch1p3p     api.SimChargePhases // nil: no 1p3p switch connected
	switch1p3pName string              // name of the 1p3p switch
}

func init() {
	registry.Add("sim-charger", NewSimChargerFromConfig)
}

////go:generate go run ../cmd/tools/decorate.go -f decorateSimCharger -b *SimCharger -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)"

// NewSimChargerFromConfig creates a simCharger from configuration
func NewSimChargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cfg := struct {
		Status     api.ChargeStatus
		Switch1p3p string
	}{
		Status:     api.StatusA,
		Switch1p3p: "",
	}

	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, err
	}

	return NewSimCharger(cfg.Status, cfg.Switch1p3p)
}

// NewSimCharger creates a simCharger instance
func NewSimCharger(chargeStatus api.ChargeStatus, switch1p3pName string) (api.Charger, error) {

	trace := util.NewLogger("chrg")
	trace.INFO.Printf("SimCharger create")
	c := &SimCharger{
		log:            trace,
		name:           "unnamed",
		simType:        api.Sim_charger,
		status:         chargeStatus,
		vehicle:        nil,
		switch1p3pName: switch1p3pName,
		switch1p3p:     nil,
	}
	return c, nil
}

// SimType gives the simulation type of the charger: Sim_charger
func (c *SimCharger) SimType() (api.SimType, error) {
	return c.simType, nil
}

// SetName sets the name of the sim
func (c *SimCharger) SetName(name string) error {
	c.name = name
	return nil
}

// Name gets the name of the sim
func (c *SimCharger) Name() (string, error) {
	return c.name, nil
}

// Enabled returns true when the carger is enabled
func (c *SimCharger) Enabled() (bool, error) {
	return c.enabled, nil
}

// Enable enables or disables the charger
func (c *SimCharger) Enable(enable bool) error {
	c.log.DEBUG.Printf("%s: enable:%t", c.name, enable)
	c.enabled = enable
	return nil

}

// MaxCurrent sets the max current the charger may feed to the vehicle
// the total power depends on the phases
func (c *SimCharger) MaxCurrent(current int64) error {
	c.maxCurrent = current
	return nil
}

// GetMaxCurrent gives the max current the charger may feed to the vehicle
func (c *SimCharger) GetMaxCurrent() (int64, error) {
	return c.maxCurrent, nil
}

// Currents implements the MeterCurrent interface
func (c *SimCharger) Currents() (float64, float64, float64, error) {

	var phases int = 1
	var err error = nil
	var i1 float64 = 0
	var i2 float64 = 0
	var i3 float64 = 0

	if c.switch1p3p != nil {
		phases, err = c.switch1p3p.GetPhases1p3p()
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if phases == 1 {
		// I = P / U
		i1 = c.powerW / voltage1p
	} else {
		// I = P / (U * SQRT(3))
		i1 = c.powerW / (voltage3p * math.Sqrt(3))
		i2 = i1
		i3 = i1
	}
	return i1, i2, i3, nil
}

// Status gives the status of the charger (car connected yes/no, charging yes/no, ...)
func (c *SimCharger) Status() (api.ChargeStatus, error) {
	return c.status, nil
}

// SetStatus sets the status of the charger (car connected yes/no, charging yes/no, ...)
func (c *SimCharger) SetStatus(status api.ChargeStatus) error {
	c.status = status
	return nil
}

// CurrentPower implements the api.Meter interface
// gives the current charger power in [W]. Positive sign: vehicle is charged
func (c *SimCharger) CurrentPower() (float64, error) {
	return c.powerW, nil
}

// Connect connects the given vehicle to this charger
func (c *SimCharger) Connect(vehicle api.SimVehicle) error {
	if c.vehicle != nil {
		newName, err := vehicle.Name()
		if err != nil {
			return err
		}
		currentName, err := c.vehicle.Name()
		if err != nil {
			return err
		}
		return fmt.Errorf("%s: unable to connect new vehicle: %vs another vehicle already connected: %s", c.name, newName, currentName)
	}
	c.vehicle = vehicle
	c.status = api.StatusB
	return nil
}

// Disconnect disconnects a vehicle from the charger
// switches the charger status back to "A"
// Can also be called if no vehicle is connected
func (c *SimCharger) Disconnect() error {
	if c.vehicle != nil {
		if _, err := c.vehicle.UpdateSoC(0); err != nil {
			return err
		}
	}
	c.powerW = 0
	c.vehicle = nil
	c.status = api.StatusA
	return nil
}

// Switch1p3pName gives the name of the configured 1p3p switch
// returns an empty string "" if no 1p3p switch is configured
func (c *SimCharger) Switch1p3pName() (string, error) {
	return c.switch1p3pName, nil
}

// SetSwitch1p3p sets the 1p3p switch that belongs to this charger
func (c *SimCharger) SetSwitch1p3p(switch1p3p api.SimChargePhases) error {
	if c.switch1p3p != nil {
		return fmt.Errorf("%s: unable to set 1p3p switch - switch already set", c.name)
	}
	c.switch1p3p = switch1p3p
	return nil
}

// Update updates the sim charger status and the connected vehicle
// returns the current power
func (c *SimCharger) Update() (float64, error) {
	var availablePowerW float64
	var phases int = 1
	var err error
	if c.switch1p3p != nil {
		phases, err = c.switch1p3p.GetPhases1p3p()
		if err != nil {
			return 0, err
		}
	} else {
		// todo default phases from config!
		phases = 1

	}
	// calculate the maximum available power
	if c.enabled && c.maxCurrent > 0 {
		if phases == 1 {
			// P = U * I
			availablePowerW = voltage1p * float64(c.maxCurrent)
		} else if phases == 3 {
			// P = U * I * SQRT(3)
			availablePowerW = voltage3p * float64(c.maxCurrent) * math.Sqrt(3)
		} else {
			return 0, fmt.Errorf("%s: invalid number of phases: %d", c.name, phases)
		}
	} else {
		availablePowerW = 0
	}

	// tell the vehicle to charge with the available power (or its maximum)
	if c.vehicle != nil {
		c.powerW, err = c.vehicle.UpdateSoC(availablePowerW)
		if err != nil {
			return 0, err
		}
	} else {
		c.powerW = 0
	}

	// recalculate the charging status
	if c.powerW != 0 {
		c.status = api.StatusC // connected and charging
	} else {
		if c.vehicle != nil {
			c.status = api.StatusB // connected not charging
		} else {
			c.status = api.StatusA // not connected, not charging
		}
	}

	c.log.DEBUG.Printf("%s: availablePower[W]:%f, used[W]:%f, Status:%s", c.name, availablePowerW, c.powerW, c.status)

	return c.powerW, nil
}
