package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/vehicle"
)

// Reading wraps the test result
type Reading = struct {
	Error string      `json:"error,omitempty"`
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
		}
	case "charger":
		if i, err = charger.NewFromConfig(typ, conf); err == nil {
			testCharger(res, i)
		}
	case "vehicle":
		if i, err = vehicle.NewFromConfig(typ, conf); err == nil {
			testVehicle(res, i)
		}
	default:
		err = fmt.Errorf("invalid device class: %s", typ)
	}

	if err != nil {
		err = fmt.Errorf("creating device failed: %v", err)
	}

	return res, err
}

// Validator manages test execution
type Validator struct {
	sync.Mutex
	generator, id int
	running       bool
	err           error
	result        map[string]Reading
}

// Test schedules the test for execution and returns unique test id
func (v *Validator) Test(class, typ string, conf map[string]interface{}) int {
	v.Lock()
	defer v.Unlock()

	// generate next test id
	v.generator++
	id := v.generator
	v.id = id

	// mark test as running
	v.running = true

	go func(id int) {
		res, err := testDevice(class, typ, conf)

		v.Lock()
		defer v.Unlock()

		// store result if this is still the current test
		if v.id == id {
			v.err = err
			v.result = res

			// mark test completed
			v.running = false
		}
	}(id)

	return id
}

// TestResult returns test results by test id
func (v *Validator) TestResult(id int) (completed bool, res map[string]Reading, err error) {
	v.Lock()
	defer v.Unlock()

	if v.id == id {
		if v.running {
			return false, nil, nil
		}

		return true, v.result, v.err
	}

	return true, res, errors.New("request outdated")
}
