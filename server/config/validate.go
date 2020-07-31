package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/util/test"
	"github.com/andig/evcc/vehicle"
)

type Reading = struct {
	Error string      `json:"error"`
	Value interface{} `json:"value,omitempty"`
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

func Validate(yaml string) (string, map[string]interface{}, error) {
	conf, err := test.ConfigFromYAML(yaml)
	if err != nil {
		return "", conf, err
	}

	typ, ok := conf["type"].(string)
	if !ok {
		return "", conf, errors.New("invalid or missing type")
	}

	delete(conf, "type")
	return typ, conf, nil
}

// testDevice executes given configuration
func testDevice(class, typ string, conf map[string]interface{}) (res map[string]Reading, err error) {
	res = make(map[string]Reading, 0)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

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

type Validator struct {
	sync.Mutex
	generator, id int
	err           error
	result        map[string]Reading
}

func (v *Validator) Test(class, typ string, conf map[string]interface{}) int {
	v.Lock()
	defer v.Unlock()

	// generate next test id
	v.generator++
	id := v.generator

	go func(id int) {
		res, error := testDevice(class, typ, conf)

		v.Lock()
		defer v.Unlock()

		// store result if this is still the current test
		if v.id == id {
			v.err = error
			v.result = res
		}
	}(id)

	return id
}

func (v *Validator) TestResult(id int) (res map[string]Reading, err error) {
	v.Lock()
	defer v.Unlock()

	if v.id == id {
		return v.result, v.err
	}

	return res, errors.New("request outdated")
}
