package configure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/evcc-io/evcc/templates"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
)

func (c *CmdConfigure) processDeviceSelection(deviceCategory string) (templates.Template, error) {
	templateItem := c.selectItem(deviceCategory)

	if templateItem.Description == itemNotPresent {
		return templateItem, ErrItemNotPresent
	}

	err := c.processDeviceRequirements(templateItem)
	if err != nil {
		return templateItem, err
	}

	return templateItem, nil
}

func (c *CmdConfigure) processDeviceValues(values map[string]interface{}, templateItem templates.Template, device device, deviceCategory string) (device, error) {
	addedDeviceIndex++

	device.Name = fmt.Sprintf("%s%d", DeviceCategories[deviceCategory].defaultName, addedDeviceIndex)
	device.Title = templateItem.Description
	for _, param := range templateItem.Params {
		if param.Name != "title" {
			continue
		}
		if len(param.Value) > 0 {
			device.Title = param.Value
		}
	}

	deviceIsValid := false
	v, err := c.configureDevice(deviceCategory, templateItem, values)
	if err == nil {
		fmt.Println()
		categoryTitle := ""
		if deviceCategory == DeviceCategoryPVMeter || deviceCategory == DeviceCategoryBatteryMeter || deviceCategory == DeviceCategoryGridMeter {
			categoryTitle = "als " + DeviceCategories[deviceCategory].title
		}
		fmt.Println("Teste die " + templateItem.Description + " Konfiguration " + categoryTitle + " ...")
		deviceIsValid, err = c.testDevice(deviceCategory, v)
		if deviceCategory == DeviceCategoryCharger {
			if deviceIsValid && err == nil {
				device.ChargerHasMeter = true
			}
		}
	}

	if !deviceIsValid {
		addedDeviceIndex--
		return device, ErrDeviceNotValid
	}

	templateItem.Params = append(templateItem.Params, templates.Param{Name: "name", Value: device.Name})
	b, err := templateItem.RenderProxyWithValues(values)
	if err != nil {
		addedDeviceIndex--
		return device, err
	}

	device.Yaml = string(b)

	return device, nil
}

// handle device requirements
func (c *CmdConfigure) processDeviceRequirements(templateItem templates.Template) error {
	if len(templateItem.Requirements.Description) > 0 {
		fmt.Println()
		fmt.Println("Das Gerät muss die folgenden Voraussetzungen erfüllen:")
		fmt.Println("  " + templateItem.Requirements.Description)
		if len(templateItem.Requirements.URI) > 0 {
			fmt.Println("  Weitere Informationen: " + templateItem.Requirements.URI)
		}
	}

	// check if sponsorship is required
	if templateItem.Requirements.Sponsorship == true && c.configuration.SponsorToken() == "" {
		fmt.Println()
		fmt.Println("Dieses Gerät benötigt ein Sponsorship von evcc. Wie das funktioniert und was ist, findest du hier: https://docs.evcc.io/docs/sponsorship")
		fmt.Println()
		if !c.askYesNo("Bist du ein Sponsor und möchtest das Sponsortoken eintragen") {
			return ErrItemNotPresent
		}
		sponsortoken := c.askValue(question{
			label:    "Bitte gib das Sponsortoken ein",
			help:     "",
			required: true})
		c.configuration.SetSponsorToken(sponsortoken)
	}

	// check if we need to setup an EEBUS HEMS
	if templateItem.Requirements.Eebus == true {
		if c.configuration.EEBUS() == "" {
			eebusConfig, err := c.eebusCertificate()

			if err != nil {
				return fmt.Errorf("Fehler: Das EEBUS Zertifikat konnte nicht erstellt werden: %s", err)
			}

			err = c.configureEEBus(eebusConfig)
			if err != nil {
				return err
			}

			eebusYaml, err := yaml.Marshal(eebusConfig)
			if err != nil {
				return err
			}
			c.configuration.SetEEBUS(string(eebusYaml))
		}

		fmt.Println()
		fmt.Println("Du hast eine Wallbox ausgewählt, welche über das EEBUS Protokoll angesprochen wird.")
		fmt.Println("Dazu muss die Wallbox nun mit evcc verbunden werden. Dies geschieht üblicherweise auf der Webseite der Wallbox.")
		fmt.Println("Drücke die Enter-Taste, wenn dies abgeschlossen ist.")
		fmt.Scanln()
	}

	return nil
}

// return template items of a given class
func (c *CmdConfigure) fetchElements(deviceCategory string) []templates.Template {
	var items []templates.Template

	for _, tmpl := range templates.ByClass(DeviceCategories[deviceCategory].class) {
		if len(tmpl.Params) == 0 || len(tmpl.Description) == 0 {
			continue
		}

		if deviceCategory == DeviceCategoryGuidedSetup {
			if tmpl.GuidedSetup.Enable {
				items = append(items, tmpl)
			}
		} else {
			if len(DeviceCategories[deviceCategory].usageFilter) == 0 ||
				c.paramChoiceContains(tmpl.Params, templates.ParamUsage, DeviceCategories[deviceCategory].usageFilter) {
				items = append(items, tmpl)
			}
		}
	}

	sort.Slice(items[:], func(i, j int) bool {
		// sort generic templates to the bottom
		if items[i].Generic && !items[j].Generic {
			return false
		}
		if !items[i].Generic && items[j].Generic {
			return true
		}
		return strings.ToLower(items[i].Description) < strings.ToLower(items[j].Description)
	})

	return items
}

// helper function to check if a param choice contains a given value
func (c *CmdConfigure) paramChoiceContains(params []templates.Param, name, filter string) bool {
	nameFound, choices := c.paramChoiceValues(params, name)

	if !nameFound {
		return false
	}

	for _, choice := range choices {
		if choice == filter {
			return true
		}
	}

	return false

	/*
		for _, item := range params {
			if item.Name != name {
				continue
			}

			filterFound = true
			if item.Choice == nil || len(item.Choice) == 0 {
				return true
			}

			for _, choice := range item.Choice {
				if choice == filter {
					return true
				}
			}
		}

		if !filterFound && considerEmptyAsTrue {
			return true
		}

		return false
	*/
}

func (c *CmdConfigure) paramChoiceValues(params []templates.Param, name string) (bool, []string) {
	nameFound := false

	choices := []string{}

	for _, item := range params {
		if item.Name != name {
			continue
		}

		nameFound = true

		for _, choice := range item.Choice {
			choices = append(choices, choice)
		}
	}

	return nameFound, choices
}

// Process an EVCC configuration item
// Returns
//   a map with param name and values
func (c *CmdConfigure) processConfig(paramItems []templates.Param, deviceCategory string, includeAdvanced bool) map[string]interface{} {
	usageFilter := DeviceCategories[deviceCategory].usageFilter

	additionalConfig := make(map[string]interface{})
	selectedModbusKey := ""

	fmt.Println()
	fmt.Println("Führe folgende Einstellungen durch:")
	fmt.Println()

	for _, param := range paramItems {
		if param.Name == "modbus" {
			choices := []string{}
			choiceKeys := []string{}
			for _, choice := range param.Choice {
				switch choice {
				case ModbusChoiceRS485:
					choices = append(choices, "Serial (USB-RS485 Adapter)")
					choiceKeys = append(choiceKeys, ModbusKeyRS485Serial)
					choices = append(choices, "Serial (Ethernet-RS485 Adapter)")
					choiceKeys = append(choiceKeys, ModbusKeyRS485TCPIP)
				case ModbusChoiceTCPIP:
					choices = append(choices, "TCP/IP")
					choiceKeys = append(choiceKeys, ModbusKeyTCPIP)
				}
			}

			if len(choices) > 0 {
				// ask for modbus address
				id := c.askValue(question{
					label:        "ID",
					help:         "Modbus ID",
					defaultValue: 1,
					valueType:    templates.ParamValueTypeInt,
					required:     true})
				additionalConfig[ModbusParamNameId] = id

				// ask for modbus interface type
				index := 0
				if len(choices) > 1 {
					index, _ = c.askChoice("Wähle die ModBus Schnittstelle aus", choices)
				}
				selectedModbusKey = choiceKeys[index]
				switch selectedModbusKey {
				case ModbusKeyRS485Serial:
					device := c.askValue(question{
						label:        "Device",
						help:         "USB-RS485 Adapter Adresse",
						exampleValue: ModbusParamValueDevice,
						required:     true})
					additionalConfig[ModbusParamNameDevice] = device

					baudrate := c.askValue(question{
						label:        "Baudrate",
						defaultValue: ModbusParamValueBaudrate,
						valueType:    templates.ParamValueTypeInt,
						required:     true})
					additionalConfig[ModbusParamNameBaudrate] = baudrate

					comset := c.askValue(question{
						label:        "ComSet",
						defaultValue: ModbusParamValueComset,
						required:     true})
					additionalConfig[ModbusParamNameComset] = comset

				case ModbusKeyRS485TCPIP, ModbusKeyTCPIP:
					if selectedModbusKey == ModbusKeyRS485TCPIP {
						additionalConfig[ModbusParamNameRTU] = "true"
					}
					host := c.askValue(question{
						label:        "Host",
						exampleValue: ModbusParamValueHost,
						required:     true})
					additionalConfig[ModbusParamNameHost] = host

					port := c.askValue(question{
						label:        "Port",
						defaultValue: ModbusParamValuePort,
						valueType:    templates.ParamValueTypeInt,
						required:     true})
					additionalConfig[ModbusParamNamePort] = port
				}
			}
		} else if param.Name != templates.ParamUsage {
			if !includeAdvanced && param.Advanced {
				continue
			}

			valueType := templates.ParamValueTypeString
			if param.ValueType != "" && funk.ContainsString(templates.ParamValueTypes, param.ValueType) {
				valueType = param.ValueType
			}

			label, help := c.userFriendlyLabelHelp(param.Name, param.Help)

			value := c.askValue(question{
				label:        label,
				defaultValue: param.Default,
				exampleValue: param.Example,
				help:         help,
				valueType:    valueType,
				mask:         param.Mask,
				required:     param.Required})
			additionalConfig[param.Name] = value
		} else if param.Name == templates.ParamUsage {
			if usageFilter != "" {
				additionalConfig[param.Name] = usageFilter
			}
		}
	}

	return additionalConfig
}
