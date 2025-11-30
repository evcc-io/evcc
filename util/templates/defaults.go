package templates

import (
	"bytes"
	_ "embed"
	"slices"

	"go.yaml.in/yaml/v4"
)

//go:embed defaults.yaml
var defaults []byte

type configDefaults struct {
	Params  []Param // Default values for common parameters
	Presets map[string][]Param
	Modbus  struct { // Details about possible ModbusInterfaces and ModbusConnectionTypes
		Definitions []Param
		Interfaces  map[string][]string // Information about physical modbus interface types (rs485, tcpip)
		Types       map[string]struct { // Details about different ways to connect to a ModbusInterface and its defaults
			Description TextLanguage
			Params      []Param
		}
	}
	DeviceGroups map[string]TextLanguage // Default device groups
}

// read the actual config into the struct, but only once
func (c *configDefaults) Load() {
	// if params are initialized, defaults have been loaded
	if c.Params != nil {
		return
	}

	// panic on unknown fields
	dec := yaml.NewDecoder(bytes.NewReader(defaults))
	dec.KnownFields(true)

	if err := dec.Decode(&c); err != nil {
		panic("failed to parse config defaults: " + err.Error())
	}

	// resolve modbus param references
	for typ := range c.Modbus.Types {
		for i, p := range c.Modbus.Types[typ].Params {
			if idx := slices.IndexFunc(c.Modbus.Definitions, func(pd Param) bool {
				return pd.Name == p.Name
			}); idx >= 0 {
				p.OverwriteProperties(c.Modbus.Definitions[idx])
				c.Modbus.Types[typ].Params[i] = p
			}
		}
	}
}

// return the param with the given name
func (c *configDefaults) ParamByName(name string) (int, Param) {
	for i, param := range c.Params {
		if param.Name == name {
			return i, param
		}
	}
	return -1, Param{}
}
