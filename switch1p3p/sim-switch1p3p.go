package switch1p3p

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type SimSwitch1p3p struct {
	log     *util.Logger // log
	name    string       // name of the sim
	simType api.SimType  // type of the sim Sim_grid, Sim_battery, ...
	phases  int          // currently switched phases - 1, or 3
}

func init() {
	registry.Add("sim-switch1p3p", NewSimSwitch1p3pFromConfig)
}

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
		log:     trace,
		name:    "unnamed",
		simType: api.Sim_switch1p3p,
		phases:  phases,
	}
	return sw, nil
}

// SimType gives the simulation type of the meter - e.g. Sim_grid
func (s *SimSwitch1p3p) SimType() (api.SimType, error) {
	return s.simType, nil
}

// SetName sets the name of the sim - used mainly for logging
func (s *SimSwitch1p3p) SetName(name string) error {
	s.name = name
	return nil
}

// Name gets the name of the sim
func (s *SimSwitch1p3p) Name() (string, error) {
	return s.name, nil
}

// Phases1p3p sets the phases the switch should switch to
func (sw *SimSwitch1p3p) Phases1p3p(phases int) error {
	sw.phases = phases
	return nil
}

// GetPhases1p3p gets the phases to which the switch is switched
func (sw *SimSwitch1p3p) GetPhases1p3p() (int, error) {
	return sw.phases, nil
}
