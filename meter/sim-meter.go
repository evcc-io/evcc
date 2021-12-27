package meter

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type SimMeterCfg struct {
	Usage      string  // grid, pv, home, battery
	Capacity   int64   // [kWh]
	PowerLimit float64 // [W]
	SoC        float64 // [%]
	Power      float64 // [W] > 0 for production, < 0 for consumption
}

type SimMeter struct {
	log         *util.Logger // log
	name        string       // name of the sim
	simType     api.SimType  // type of the sim Sim_grid, Sim_battery, ...
	usage       string       // grid, battery, ...
	capacitykWh int64        // [kWh] for battery
	chargekWh   float64      // [kWh] for battery
	powerLimitW float64      // [W] for battery
	soc         float64      // [%] for battery
	powerW      float64      // [w]
	lastUpdate  time.Time    // last update Timestamp for battery
}

func init() {
	registry.Add("sim-meter", NewSimMeterFromConfig)
}

//go:gen go run ../cmd/tools/decorate.go -f decorateSimMeter -b *SimMeter -r api.Meter -t "api.Battery,SoC,func()(float64, error)"

// NewSimMeterFromConfig creates a simulation meter from configuration with the configured simulation type
func NewSimMeterFromConfig(other map[string]interface{}) (api.Meter, error) {
	cfg := &SimMeterCfg{
		Usage:      "grid",
		Capacity:   0,
		PowerLimit: 0,
		SoC:        0,
		Power:      0,
	}

	if err := util.DecodeOther(other, &cfg); err != nil {
		return nil, err
	}
	return NewSimMeter(cfg)
}

// NewSimMeter creates a simulation meter instance
func NewSimMeter(cfg *SimMeterCfg) (api.Meter, error) {

	usageString := strings.ToLower(cfg.Usage)
	loggerName := usageString

	s := &SimMeter{
		name:        "unnamed",
		usage:       cfg.Usage,
		capacitykWh: cfg.Capacity,
		powerLimitW: cfg.PowerLimit,
		soc:         cfg.SoC,
		chargekWh:   (float64(cfg.Capacity) * cfg.SoC) / 100,
		powerW:      cfg.Power,
		lastUpdate:  time.Time{},
	}

	switch usageString {
	case "grid":
		s.simType = api.Sim_grid
	case "pv":
		s.simType = api.Sim_pv
	case "battery":
		loggerName = "batt"
		s.simType = api.Sim_battery
	case "home":
		s.simType = api.Sim_home
	default:
		return nil, fmt.Errorf("sim-meter: invalid usage: %s", cfg.Usage)

	}
	s.log = util.NewLogger(loggerName)

	s.log.INFO.Printf("SimMeter Usage %s", usageString)
	return s, nil
}

// SimType gives the simulation type of the meter - e.g. Sim_grid
func (s *SimMeter) SimType() (api.SimType, error) {
	return s.simType, nil
}

// SetName sets the name of the sim
func (s *SimMeter) SetName(name string) error {
	s.name = name
	return nil
}

// Name gets the name of the sim
func (s *SimMeter) Name() (string, error) {
	return s.name, nil
}

// CurrentPower provides the current power in [W]
func (s *SimMeter) CurrentPower() (float64, error) {
	return s.powerW, nil
}

// SetCurrentPower sets the current power in [W]
func (s *SimMeter) SetCurrentPower(powerW float64) error {
	s.powerW = powerW
	return nil
}

// SoC provides the SoC of the battery (if this meter represents a battery) in [%]
func (s *SimMeter) SoC() (float64, error) {
	if s.simType == api.Sim_battery {
		return s.soc, nil
	} else {
		return 0, fmt.Errorf("%s: SoC request only allowed for batteries", s.name)
	}
}

// SetSoC sets the current SoC of the battery (if this meter represents a battery) in [%]
func (s *SimMeter) SetSoC(soc float64) error {
	if s.simType == api.Sim_battery {
		s.soc = soc
		s.chargekWh = float64(s.capacitykWh) * s.soc / 100
		return nil
	} else {
		return fmt.Errorf("%s: SetSoc only allowed for batteries", s.name)
	}
}

// UpdateSoc recalculates the SoC of the battery
// returns the power [W] it charges (negative sign) or discharges (positive sign)
// in the next update cycle
func (s *SimMeter) UpdateSoC(availablePowerW float64) (float64, error) {
	if s.lastUpdate.IsZero() {
		s.lastUpdate = time.Now()
	}
	elapsedTime := time.Since(s.lastUpdate)
	// add charged energy - convert from Ws to kWh - dividing by 3.6E6
	s.chargekWh = s.chargekWh + ((-s.powerW)*elapsedTime.Seconds())/(3.6e6)
	if s.chargekWh > float64(s.capacitykWh) {
		// battery fully charged - stop charging
		s.chargekWh = float64(s.capacitykWh)
		s.powerW = 0
	} else if s.chargekWh < 0 {
		// battery fully discharged - stop discharging
		s.chargekWh = 0
		s.powerW = 0
	}
	s.lastUpdate = time.Now()
	if s.capacitykWh > 0 {
		s.soc = (s.chargekWh / float64(s.capacitykWh)) * 100
	} else {
		s.soc = 0
	}

	// calculate charge/discharge power for next update cycle
	if s.soc < 100 && availablePowerW > 0 {
		// charge battery: negative power sign: consumes
		s.powerW = -(math.Min(availablePowerW, s.powerLimitW))
	} else if s.soc > 0 && availablePowerW < 0 {
		// discharge battery: positive power sign: generates
		s.powerW = math.Min(-availablePowerW, s.powerLimitW)
	}

	s.log.DEBUG.Printf("%s: Charge[kWh]:%f, SoC[%%]:%f Power[W]:%f", s.name, s.chargekWh, s.soc, s.powerW)
	return s.powerW, nil
}

// SetCapacity sets the capacity of the battery (if this meter represents a bettery) in [kWh]
func (s *SimMeter) SetCapacity(capacitykWh int64) error {
	if s.simType == api.Sim_battery {
		s.capacitykWh = capacitykWh
		return nil
	} else {
		return fmt.Errorf("%s: SetCapacity only allowed for batteries", s.name)
	}
}

// Capacity gives the capacity of the battery (if this meter represents a battery) in [kWh]
func (s *SimMeter) Capacity() int64 {
	if s.simType == api.Sim_battery {
		return s.capacitykWh
	} else {
		return 0
	}
}

// SetPowerLimit sets the PowerLimit of the battery (if this meter represents a bettery) in [W]
func (s *SimMeter) SetPowerLimit(powerLimitW float64) error {
	if s.simType == api.Sim_battery {
		s.powerLimitW = powerLimitW
		return nil
	} else {
		return fmt.Errorf("%s: SetPowerLimit only allowed for batteries", s.name)
	}
}

// PowerLimit gives the PowerLimit of the battery (if this meter represents a battery) in [W]
func (s *SimMeter) PowerLimit() (float64, error) {
	if s.simType == api.Sim_battery {
		return s.powerLimitW, nil
	} else {
		return 0, fmt.Errorf("%s: PowerLimit request only allowed for batteries", s.name)
	}
}
