package configure

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/manifoldco/promptui"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
)

// create a yaml configuration
func (c *CmdConfigure) renderConfiguration() ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(configTmpl)
	if err != nil {
		panic(err)
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, c.configuration)

	return bytes.TrimSpace(out.Bytes()), err
}

func (c *CmdConfigure) configureDeviceCategory(deviceCategory string, deviceIndex int) error {
	fmt.Println()
	fmt.Printf("- Configure your %s\n", DeviceCategories[deviceCategory].title)

	device, err := c.processDeviceCategory(deviceCategory, deviceIndex)
	if err != nil && err != ErrItemNotPresent {
		c.log.FATAL.Fatal(err)
	}

	if err != ErrItemNotPresent {
		switch DeviceCategories[deviceCategory].class {
		case DeviceClassCharger:
			c.configuration.Chargers = append(c.configuration.Chargers, device)
		case DeviceClassMeter:
			c.configuration.Meters = append(c.configuration.Meters, device)
			switch DeviceCategories[deviceCategory].usageFilter {
			case UsageChoiceGrid:
				c.configuration.Site.Grid = device.Name
			case UsageChoicePV:
				c.configuration.Site.PVs = append(c.configuration.Site.PVs, device.Name)
			case UsageChoiceBattery:
				c.configuration.Site.Batteries = append(c.configuration.Site.Batteries, device.Name)
			}
		case DeviceClassVehicle:
			c.configuration.Vehicles = append(c.configuration.Vehicles, device)
		}
	}

	return err
}

// let the user select a device item from a list defined by class and filter
func (c *CmdConfigure) processDeviceCategory(deviceCategory string, deviceIndex int) (device, error) {
	var repeat bool = true

	device := device{
		Name:  DeviceCategories[deviceCategory].defaultName,
		Title: "",
		Yaml:  "",
	}

	for ok := true; ok; ok = repeat {
		fmt.Println()
		templateItem := c.selectItem(deviceCategory)
		if templateItem.Description == itemNotPresent {
			return device, ErrItemNotPresent
		}

		// check if we need to setup an EEBUS HEMS
		if DeviceCategories[deviceCategory].class == DeviceClassCharger && templateItem.Requirements.Eebus == true {
			if c.configuration.EEBUS == "" {
				eebusConfig, err := c.eebusCertificate()

				if err != nil {
					return device, fmt.Errorf("error creating EEBUS cert: %s", err)
				}

				err = c.configureEEBus(eebusConfig)
				if err != nil {
					return device, err
				}

				eebusYaml, err := yaml.Marshal(eebusConfig)
				if err != nil {
					return device, err
				}
				c.configuration.EEBUS = string(eebusYaml)
			}

			fmt.Println()
			fmt.Println("You have selected an EEBUS wallbox.")
			fmt.Println("Please pair your wallbox with EVCC in the wallbox web interface")
			fmt.Println("When done, press enter to continue.")
			fmt.Scanln()
		}

		var values map[string]interface{}
		values = c.processConfig(templateItem.Params, deviceCategory, false)
		device.Name = fmt.Sprintf("%s%d", DeviceCategories[deviceCategory].defaultName, deviceIndex)
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
			fmt.Println("Testing configuration...")
			fmt.Println()
			deviceIsValid, err = c.testDevice(deviceCategory, v)
			if deviceCategory == DeviceCategoryCharger {
				if deviceIsValid && err == nil {
					device.ChargerHasMeter = true
				}
			}
		}

		if !deviceIsValid {
			if err != nil {
				fmt.Println("Error: ", err)
			}
			fmt.Println()
			if !c.askYesNo("This device configuration does not work and can not be selected. Do you want to restart the device selection?") {
				fmt.Println()
				return device, err
			}
			continue
		}

		fmt.Println("Success.")

		templateItem.Params = append(templateItem.Params, templates.Param{Name: "name", Value: device.Name})
		b, err := templateItem.RenderProxyWithValues(values)
		if err != nil {
			return device, err
		}

		device.Yaml = string(b)
		repeat = false
	}

	return device, nil
}

// provide all entered name values
func (c *CmdConfigure) enteredNames() []string {
	var names []string

	for _, v := range c.configuration.Chargers {
		names = append(names, v.Name)
	}

	for _, v := range c.configuration.Meters {
		names = append(names, v.Name)
	}

	for _, v := range c.configuration.Vehicles {
		names = append(names, v.Name)
	}

	return names
}

// create a configured device from a template so we can test it
func (c *CmdConfigure) configureDevice(deviceCategory string, device templates.Template, values map[string]interface{}) (interface{}, error) {
	b, err := device.RenderResult(values)
	if err != nil {
		return nil, err
	}

	var instance struct {
		Type  string
		Other map[string]interface{} `yaml:",inline"`
	}

	if err := yaml.Unmarshal(b, &instance); err != nil {
		return nil, err
	}

	var v interface{}

	switch DeviceCategories[deviceCategory].class {
	case DeviceClassMeter:
		v, err = meter.NewFromConfig(instance.Type, instance.Other)
	case DeviceClassCharger:
		v, err = charger.NewFromConfig(instance.Type, instance.Other)
	case DeviceClassVehicle:
		v, err = vehicle.NewFromConfig(instance.Type, instance.Other)
	}
	if err != nil {
		return nil, err
	}

	return v, nil
}

// return template items of a given class
func (c *CmdConfigure) fetchElements(deviceCategory string) []templates.Template {
	var items []templates.Template

	for _, tmpl := range templates.ByClass(DeviceCategories[deviceCategory].class) {
		if len(tmpl.Params) == 0 || len(tmpl.Description) == 0 {
			continue
		}

		if len(DeviceCategories[deviceCategory].usageFilter) == 0 ||
			c.paramChoiceContains(tmpl.Params, templates.ParamUsage, DeviceCategories[deviceCategory].usageFilter, true) {
			items = append(items, tmpl)
		}
	}

	sort.Slice(items[:], func(i, j int) bool {
		return strings.ToLower(items[i].Description) < strings.ToLower(items[j].Description)
	})

	return items
}

// helper function to check if a param choice contains a given value
func (c *CmdConfigure) paramChoiceContains(params []templates.Param, name, filter string, considerEmptyAsTrue bool) bool {
	filterFound := false
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
}

// PromptUI: select item from list
func (c *CmdConfigure) selectItem(deviceCategory string) templates.Template {
	promptuiTemplates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ .Description }}",
		Inactive: "   {{ .Description }}",
		Selected: fmt.Sprintf("%s: {{ .Description }}", DeviceCategories[deviceCategory].class),
	}

	var emptyItem templates.Template
	emptyItem.Description = itemNotPresent

	items := c.fetchElements(deviceCategory)
	items = append(items, emptyItem)

	prompt := promptui.Select{
		Label:     fmt.Sprintf("Select your %s", DeviceCategories[deviceCategory].title),
		Items:     items,
		Templates: promptuiTemplates,
		Size:      10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return items[index]
}

// PromptUI: select item from list
func (c *CmdConfigure) askChoice(label string, choices []string) (int, string) {
	promptuiTemplates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ . }}",
		Inactive: "   {{ . }}",
		Selected: "   {{ . }}",
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     choices,
		Templates: promptuiTemplates,
		Size:      10,
	}

	index, result, err := prompt.Run()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return index, result
}

// PromptUI: ask yes/no question, return true if yes is selected
func (c *CmdConfigure) askYesNo(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()

	return !errors.Is(err, promptui.ErrAbort)
}

// PromputUI: ask for input
func (c *CmdConfigure) askValue(label, exampleValue, hint string, invalidValues []string, mask, required bool) string {
	promptuiTemplates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	validate := func(input string) error {
		if invalidValues != nil && funk.ContainsString(invalidValues, input) {
			return errors.New("Value '" + input + "' is already used")
		}

		if required && len(input) == 0 {
			return errors.New("Value may not be empty")
		}

		return nil
	}

	if hint != "" {
		fmt.Println(hint)
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: promptuiTemplates,
		Default:   exampleValue,
		Validate:  validate,
		AllowEdit: true,
	}

	if mask {
		prompt.Mask = '*'
	}

	result, err := prompt.Run()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return result
}

// Process an EVCC configuration item
// Returns
//   a map with param name and values
func (c *CmdConfigure) processConfig(paramItems []templates.Param, deviceCategory string, includeAdvanced bool) map[string]interface{} {
	usageFilter := DeviceCategories[deviceCategory].usageFilter

	additionalConfig := make(map[string]interface{})
	selectedModbusKey := ""

	fmt.Println("Enter the configuration values:")

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
				id := c.askValue("ID", "1", "Modbus ID", nil, false, true)
				additionalConfig[ModbusParamNameId] = id

				// ask for modbus interface type
				index := 0
				if len(choices) > 1 {
					index, _ = c.askChoice("Select the Modbus interface", choices)
				}
				selectedModbusKey = choiceKeys[index]
				switch selectedModbusKey {
				case ModbusKeyRS485Serial:
					device := c.askValue("Device", ModbusParamValueDevice, "USB-RS485 Adapter address", nil, false, true)
					additionalConfig[ModbusParamNameDevice] = device
					baudrate := c.askValue("Baudrate", ModbusParamValueBaudrate, "", nil, false, true)
					additionalConfig[ModbusParamNameBaudrate] = baudrate
					comset := c.askValue("ComSet", ModbusParamValueComset, "", nil, false, true)
					additionalConfig[ModbusParamNameComset] = comset
				case ModbusKeyRS485TCPIP, ModbusKeyTCPIP:
					if selectedModbusKey == ModbusKeyRS485TCPIP {
						additionalConfig[ModbusParamNameRTU] = "true"
					}
					host := c.askValue("Host", ModbusParamValueHost, "IP address or hostname", nil, false, true)
					additionalConfig[ModbusParamNameHost] = host
					port := c.askValue("Port", ModbusParamValuePort, "Port address", nil, false, true)
					additionalConfig[ModbusParamNamePort] = port
				}
			}
		} else if param.Name != templates.ParamUsage {
			if !includeAdvanced && param.Advanced {
				continue
			}
			exampleValue := param.Example
			if exampleValue == "" && param.Default != "" {
				exampleValue = param.Default
			}
			value := c.askValue(param.Name, exampleValue, param.Hint, nil, param.Mask, param.Required)
			additionalConfig[param.Name] = value
		} else if param.Name == templates.ParamUsage {
			if usageFilter != "" {
				additionalConfig[param.Name] = usageFilter
			}
		}
	}

	return additionalConfig
}
