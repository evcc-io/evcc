package server

import (
	"fmt"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/util/test"
	"github.com/andig/evcc/vehicle"
)

type configSample = struct {
	Name   string `json:"name"`
	Sample string `json:"template"`
}

type Reading = struct {
	Error string      `json:"error"`
	Value interface{} `json:"value,omitempty"`
}

// ConfigurationSamplesByClass returns a slice of configuration templates
func ConfigurationSamplesByClass(class string) []configSample {
	res := make([]configSample, 0)
	for _, conf := range test.ConfigTemplates(class) {
		typedSample := fmt.Sprintf("type: %s\n%s", conf.Type, conf.Sample)
		t := configSample{
			Name:   conf.Name,
			Sample: typedSample,
		}
		res = append(res, t)
	}
	return res
}

func testMeter(res map[string]Reading, i interface{}) {
	if i, ok := i.(api.Meter); ok {
		r := Reading{}
		if power, err := i.CurrentPower(); err != nil {
			r.Error = err.Error()
		} else {
			r.Value = power
		}
		res["power"] = r
	}

	if i, ok := i.(api.MeterEnergy); ok {
		r := Reading{}
		if energy, err := i.TotalEnergy(); err != nil {
			r.Error = err.Error()
		} else {
			r.Value = energy
		}
		res["energy"] = r
	}

	if i, ok := i.(api.MeterCurrent); ok {
		r := Reading{}
		if i1, i2, i3, err := i.Currents(); err != nil {
			r.Error = err.Error()
		} else {
			r.Value = []float64{i1, i2, i3}
		}
		res["current"] = r
	}
}

func testCharger(res map[string]Reading, i interface{}) {
	if i, ok := i.(api.ChargeRater); ok {
		r := Reading{}
		if energy, err := i.ChargedEnergy(); err != nil {
			r.Error = err.Error()
		} else {
			r.Value = energy
		}
		res["energy"] = r
	}

	if i, ok := i.(api.ChargeTimer); ok {
		r := Reading{}
		if duration, err := i.ChargingTime(); err != nil {
			r.Error = err.Error()
		} else {
			r.Value = duration
		}
		res["duration"] = r
	}
}

func testVehicle(res map[string]Reading, i interface{}) {
	if i, ok := i.(api.Vehicle); ok {
		r := Reading{}
		if soc, err := i.ChargeState(); err != nil {
			r.Error = err.Error()
		} else {
			r.Value = soc
		}
		res["soc"] = r
	}
}

// TestConfiguration executes given configuration
func TestConfiguration(class, yaml string) (res map[string]Reading, err error) {
	res = make(map[string]Reading, 0)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic due to %v", r)
		}
	}()

	conf, err := test.ConfigFromYAML(yaml)
	if err != nil {
		return res, fmt.Errorf("parsing failed: %v", err)
	}

	typ, ok := conf["type"].(string)
	if !ok {
		return res, fmt.Errorf("parsing failed: invalid or missing type")
	}
	delete(conf, "type")

	var i interface{}

	switch class {
	case "meter":
		if i, err = meter.NewFromConfig(typ, conf); err == nil {
			testMeter(res, i)
		} else {
			err = fmt.Errorf("creating device failed: %v", err)
		}
		return res, err
	case "charger":
		if i, err := charger.NewFromConfig(typ, conf); err == nil {
			testCharger(res, i)
		} else {
			err = fmt.Errorf("creating device failed: %v", err)
		}
		return res, err
	case "vehicle":
		if i, err := vehicle.NewFromConfig(typ, conf); err == nil {
			testVehicle(res, i)
		} else {
			err = fmt.Errorf("creating device failed: %v", err)
		}
		return res, err
	default:
		return res, fmt.Errorf("invalid device class: %s", typ)
	}
}
