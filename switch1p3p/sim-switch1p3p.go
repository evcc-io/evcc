package switch1p3p

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type SimSwitch1p3p struct {
	log             *util.Logger // log
	name            string       // name of the sim
	simType         api.SimType  // type of the sim Sim_grid, Sim_battery, ...
	phases          int          // currently switched phases - 1, or 3
	enabled         bool         // true when enabled, false otherwise. Phase switching is only allowed when enabled
	lockPhases      int          // lock value for phases
	isLockRequested bool         // true when the phases are locked
	isLockTriggered bool         // true when the lock is requested and a phases request is called
}

func init() {
	registry.Add("sim-switch1p3p", NewSimSwitch1p3pFromConfig)
}

// NewSimSwitch1p3pFromConfig creates a simulated 1p3p switch from the given configuration
func NewSimSwitch1p3pFromConfig(other map[string]interface{}) (api.ChargePhases, error) {
	cfg := struct {
		Phases int
	}{
		Phases: 1,
	}

	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, err
	}

	return NewSimSwitch1p3p(cfg.Phases)
}

func NewSimSwitch1p3p(phases int) (api.ChargePhases, error) {

	trace := util.NewLogger("switch")
	trace.INFO.Println("SimSwitch1p3p create")

	sw := &SimSwitch1p3p{
		log:             trace,
		name:            "unnamed",
		simType:         api.Sim_switch1p3p,
		phases:          phases,
		enabled:         false,
		lockPhases:      1,
		isLockRequested: false,
		isLockTriggered: false,
	}
	return sw, nil
}

// SimType gives the simulation type of the meter - e.g. Sim_grid
func (sw *SimSwitch1p3p) SimType() (api.SimType, error) {
	return sw.simType, nil
}

// SetName sets the name of the sim - used mainly for logging
func (sw *SimSwitch1p3p) SetName(name string) error {
	sw.name = name
	return nil
}

// Name gets the name of the sim
func (sw *SimSwitch1p3p) Name() (string, error) {
	return sw.name, nil
}

// LockPhases1p3p arms the "phases lock" trap. It is triggered with the next "phases1p3p" request
func (sw *SimSwitch1p3p) LockPhases1p3p(phases int) error {
	sw.lockPhases = phases
	sw.isLockRequested = true
	return nil
}

// UnlockPhases1p3p disarms the "Phases lock" trap and switches everything back to normal
func (sw *SimSwitch1p3p) UnlockPhases1p3p() error {
	sw.isLockRequested = false
	sw.isLockTriggered = false
	return nil
}

// Phases1p3p sets the phases the switch should switch to
func (sw *SimSwitch1p3p) Phases1p3p(phases int) error {
	if sw.enabled {
		return fmt.Errorf("%s: switching phases only allowed when disabled", sw.name)
	}
	sw.isLockTriggered = sw.isLockRequested
	sw.phases = phases
	return nil
}

// GetPhases1p3p gets the phases to which the switch is switched
func (sw *SimSwitch1p3p) GetPhases1p3p() (int, error) {
	if sw.isLockTriggered {
		return sw.lockPhases, nil
	}
	return sw.phases, nil
}

// Enable implements the ChargeEnable interface to support enabling and disabling the connected charger
func (sw *SimSwitch1p3p) Enable(enable bool) error {
	sw.log.DEBUG.Printf("%s: enable:%t", sw.name, enable)
	sw.enabled = enable
	return nil
}

// Enabled implements the ChargeEnable interface.
// Gives the current enabled status of the 1p3p switch
func (sw *SimSwitch1p3p) Enabled() (bool, error) {
	return sw.enabled, nil
}
