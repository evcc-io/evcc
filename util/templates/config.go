package templates

import (
	"fmt"

	"github.com/evcc-io/evcc/templates/definition"
	"gopkg.in/yaml.v3"
)

type ConfigDefaultsDefinition struct {
	Params  []Param // Default values for common parameters
	Presets map[string]struct {
		Params []Param
		Render string
	}
	Modbus struct { // Details about possible ModbusInterfaces and ModbusConnectionTypes
		Interfaces map[string][]string // Information about physical modbus interface types (rs485, tcpip)
		Types      map[string]struct { // Details about different ways to connect to a ModbusInterface and its defaults
			Description TextLanguage
			Params      []Param
		}
	}
	DeviceGroups map[string]TextLanguage // Default device groups
}

type ConfigDefaults struct {
	read bool

	Config ConfigDefaultsDefinition
}

// read the actual config into the struct, but only once
func (c *ConfigDefaults) LoadDefaults() {
	if c.read {
		return
	}

	if err := yaml.Unmarshal([]byte(definition.DefaultsContent), &c.Config); err != nil {
		panic(fmt.Errorf("Error: failed to parse deviceGroupListDefinition: %v\n", err))
	}
	c.read = true
}

// return the param with the given name
func (c *ConfigDefaults) ParamByName(name string) (int, Param) {
	for i, param := range c.Config.Params {
		if param.Name == name {
			return i, param
		}
	}
	return -1, Param{}
}
