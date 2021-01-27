package config

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// Result is a single test result, either value or error
type Result struct {
	Value interface{} `json:"value,omitempty"`
	Error error       `json:"error,omitempty"`
}

// MarshalJSON implements the json marshaler for the result type
func (r Result) MarshalJSON() ([]byte, error) {
	var err string
	if r.Error != nil {
		err = r.Error.Error()
	}

	return json.Marshal(&struct {
		Value interface{} `json:"value,omitempty"`
		Error string      `json:"error,omitempty"`
	}{
		Value: r.Value,
		Error: err,
	})
}

func testDevice(v interface{}) map[string]Result {
	res := make(map[string]Result)

	// meter

	if v, ok := v.(api.Meter); ok {
		power, err := v.CurrentPower()
		res["power"] = Result{power, err}
	}

	if v, ok := v.(api.MeterEnergy); ok {
		energy, err := v.TotalEnergy()
		res["energy"] = Result{energy, err}
	}

	if v, ok := v.(api.MeterCurrent); ok {
		i1, i2, i3, err := v.Currents()
		res["currents"] = Result{[]float64{i1, i2, i3}, err}
	}

	if v, ok := v.(api.Battery); ok {
		soc, err := v.SoC()
		res["soc"] = Result{soc, err}
	}

	// charger

	if v, ok := v.(api.Charger); ok {
		enabled, err := v.Enabled()
		res["enabled"] = Result{enabled, err}

		status, err := v.Status()
		res["status"] = Result{status, err}
	}

	if v, ok := v.(api.ChargeRater); ok {
		energy, err := v.ChargedEnergy()
		res["energy"] = Result{energy, err}
	}

	if v, ok := v.(api.ChargeTimer); ok {
		duration, err := v.ChargingTime()
		res["duration"] = Result{duration, err}
	}

	// vehicle

	if v, ok := v.(api.Vehicle); ok {
		res["capacity"] = Result{v.Capacity(), nil}

		soc, err := v.ChargeState()
		res["soc"] = Result{soc, err}
	}

	if v, ok := v.(api.VehicleRange); ok {
		rng, err := v.Range()
		res["range"] = Result{rng, err}
	}

	if v, ok := v.(api.VehicleStatus); ok {
		status, err := v.Status()
		res["status"] = Result{status, err}
	}

	if v, ok := v.(api.ChargeFinishTimer); ok {
		ft, err := v.FinishTime()
		res["ft"] = Result{ft, err}
	}

	// if v, ok := v.(api.Climater); ok {
	// 	 active, ot, tt, err := v.Climater();
	// 			fmt.Fprintf(w, "Outside temp:\t%.1f°C\n", ot)
	// 		}
	// 		if !math.IsNaN(tt) {
	// 			fmt.Fprintf(w, "Target temp:\t%.1f°C\n", tt)
	// 		}
	// 	}
	// }

	return res
}

// Test creates the device from the given configuration file and tries to read all supported readings
func Test(class string, body io.Reader) (interface{}, error) {
	// decode json configuration
	var req map[string]interface{}
	err := json.NewDecoder(body).Decode(&req)
	if err != nil {
		return nil, err
	}

	// split device type and settings
	var config struct {
		Type  string
		Other map[string]interface{} `mapstructure:",remain"`
	}
	err = util.DecodeOther(req, &config)
	if err != nil {
		return nil, err
	}

	// find the type definition
	typeDef, err := typeDefinition(class, config.Type)
	if err != nil {
		return nil, err
	}

	// validate the configuration
	configStruct := reflect.New(reflect.TypeOf(typeDef.Config)).Elem().Interface()
	if err := util.DecodeOther(config.Other, &configStruct); err != nil {
		return nil, err
	}
	if err := util.Validate(configStruct); err != nil {
		return nil, err
	}

	// create the device
	dev, err := typeDef.Factory(config.Other)
	if err != nil {
		return nil, fmt.Errorf("creating %s failed: %w", class, err)
	}

	// test the device
	return testDevice(dev), nil
}
