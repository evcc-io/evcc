package server

import (
	"errors"
	"fmt"

	"github.com/andig/evcc/charger"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/util/test"
	"github.com/andig/evcc/vehicle"
)

type configSample = struct {
	Name   string `json:"name"`
	Sample string `json:"template"`
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

// TestConfiguration executes given configuration
func TestConfiguration(class, yaml string) error {
	conf, err := test.ConfigFromYAML(yaml)
	if err != nil {
		return err
	}

	typI, ok := conf["type"]
	if !ok {
		return errors.New("missing type")
	}
	typ, ok := typI.(string)
	if !ok {
		return errors.New("invalid type")
	}
	delete(conf, "type")

	switch class {
	case "meter":
		_, err := meter.NewFromConfig(typ, conf)
		return err
	case "charger":
		_, err := charger.NewFromConfig(typ, conf)
		return err
	case "vehicle":
		_, err := vehicle.NewFromConfig(typ, conf)
		return err
	default:
		return fmt.Errorf("invalid type: %s", typ)
	}
}
