package templates

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed defaults.yaml
var defaults []byte

type configDefaults struct {
	Params  []Param // Default values for common parameters
	Presets map[string]struct {
		Params []Param
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

// read the actual config into the struct, but only once
func (c *configDefaults) Load() {
	// if params are initialized, defaults have been loaded
	if c.Params != nil {
		return
	}

	if err := yaml.Unmarshal(defaults, &c); err != nil {
		panic(fmt.Errorf("failed to parse deviceGroupListDefinition: %v", err))
	}

	// resolve modbus param references
	for k := range c.Modbus.Types {
		for i, p := range c.Modbus.Types[k].Params {
			// if this is a reference, get the referenced values and then overwrite it with the values defined here
			if p.IsReference() {
				finalName := p.Name
				referencedItemName := p.Name
				if p.ReferenceName != "" {
					referencedItemName = p.ReferenceName
				}
				_, referencedParam := c.ParamByName(referencedItemName)
				referencedParam.OverwriteProperties(p)
				referencedParam.Name = finalName
				p = referencedParam
				c.Modbus.Types[k].Params[i] = p
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
