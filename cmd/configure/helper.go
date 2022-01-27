package configure

import (
	"fmt"
	"sort"
	"strings"

	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/thoas/go-funk"
	stripmd "github.com/writeas/go-strip-markdown"
	"gopkg.in/yaml.v3"
)

// processDeviceSelection processes the the user selected device, check if it is an actual device and make sure the requirements are set
func (c *CmdConfigure) processDeviceSelection(deviceCategory DeviceCategory) (templates.Template, error) {
	templateItem := c.selectItem(deviceCategory)

	if templateItem.Title() == c.localizedString("ItemNotPresent", nil) {
		return templateItem, c.errItemNotPresent
	}

	if err := c.processDeviceRequirements(templateItem); err != nil {
		return templateItem, err
	}

	return templateItem, nil
}

// processDeviceValues processes the user provided values, create a device configuration and check if it is a valid device
func (c *CmdConfigure) processDeviceValues(values map[string]interface{}, templateItem templates.Template, device device, deviceCategory DeviceCategory) (device, error) {
	c.addedDeviceIndex++

	device.Name = fmt.Sprintf("%s%d", DeviceCategories[deviceCategory].defaultName, c.addedDeviceIndex)
	device.Title = templateItem.Title()
	for item, value := range values {
		if strings.ToLower(item) != "title" {
			continue
		}
		if len(value.(string)) > 0 {
			device.Title = value.(string)
		}
	}

	categoryWithUsage := deviceCategory == DeviceCategoryPVMeter || deviceCategory == DeviceCategoryBatteryMeter || deviceCategory == DeviceCategoryGridMeter

	fmt.Println()
	if categoryWithUsage {
		fmt.Println(c.localizedString("TestingDevice_TitleUsage", localizeMap{"Device": templateItem.Title(), "Usage": deviceCategory.String()}))
	} else {
		fmt.Println(c.localizedString("TestingDevice_Title", localizeMap{"Device": templateItem.Title()}))
	}

	deviceTest := DeviceTest{
		DeviceCategory: deviceCategory,
		Template:       templateItem,
		ConfigValues:   values,
	}

	testResult, err := deviceTest.Test()
	if err != nil {
		fmt.Println("  ", c.localizedString("Error", localizeMap{"Error": err}))
		fmt.Println()

		question := c.localizedString("TestingDevice_AddFailed", localizeMap{"Device": templateItem.Title()})
		if categoryWithUsage {
			question = c.localizedString("TestingDevice_AddFailedUsage", localizeMap{"Device": templateItem.Title(), "Usage": deviceCategory.String()})
		}
		if !c.askYesNo(question) {
			c.addedDeviceIndex--
			return device, c.errDeviceNotValid
		}
	} else {
		if deviceCategory == DeviceCategoryCharger && testResult == DeviceTestResultValid {
			device.ChargerHasMeter = true
		}
	}

	templateItem.Params = append(templateItem.Params, templates.Param{Name: "name", Value: device.Name})
	if !c.expandedMode {
		b, err := templateItem.RenderProxyWithValues(values, c.lang)
		if err != nil {
			c.addedDeviceIndex--
			return device, err
		}

		device.Yaml = string(b)
	} else {
		for _, p := range templateItem.Params {
			if p.Name == "name" {
				values["name"] = p.Value
				templateItem.Render = fmt.Sprintf("name: {{ .name }}\n%s", templateItem.Render)
			}
		}
		b, _, err := templateItem.RenderResult(templates.TemplateRenderModeInstance, values)
		if err != nil {
			c.addedDeviceIndex--
			return device, err
		}

		device.Yaml = string(b)
	}

	return device, nil
}

func (c *CmdConfigure) processDeviceCapabilities(capabilitites []string) {
	if funk.ContainsString(capabilitites, templates.CapabilitySMAHems) {
		c.capabilitySMAHems = true
	}
}

// processDeviceRequirements handles device requirements
func (c *CmdConfigure) processDeviceRequirements(templateItem templates.Template) error {
	requirementDescription := stripmd.Strip(templateItem.Requirements.Description.String(c.lang))
	if len(requirementDescription) > 0 {
		fmt.Println()
		fmt.Println("-------------------------------------------------")
		fmt.Println(c.localizedString("Requirements_Title", nil))
		fmt.Println(requirementDescription)
		if len(templateItem.Requirements.URI) > 0 {
			fmt.Println("  " + c.localizedString("Requirements_More", nil) + " " + templateItem.Requirements.URI)
		}
		fmt.Println("-------------------------------------------------")
	}

	// check if sponsorship is required
	if funk.ContainsString(templateItem.Requirements.EVCC, templates.RequirementSponsorship) && c.configuration.config.SponsorToken == "" {
		if err := c.askSponsortoken(true); err != nil {
			return err
		}
	}

	// check if we need to setup an MQTT broker
	if funk.ContainsString(templateItem.Requirements.EVCC, templates.RequirementMQTT) {
		if c.configuration.config.MQTT == "" {
			mqttConfig, err := c.configureMQTT()
			if err != nil {
				return err
			}

			mqttYaml, err := yaml.Marshal(mqttConfig)
			if err != nil {
				return err
			}

			c.configuration.config.MQTT = string(mqttYaml)
		}
	}

	// check if we need to setup an EEBUS HEMS
	if funk.ContainsString(templateItem.Requirements.EVCC, templates.RequirementEEBUS) {
		if c.configuration.config.EEBUS == "" {
			fmt.Println()
			fmt.Println("-- EEBUS -----------------------------------")
			fmt.Println()
			eebusConfig, err := c.eebusCertificate()
			if err != nil {
				return fmt.Errorf("%s: %s", c.localizedString("Requirements_EEBUS_Cert_Error", nil), err)
			}

			err = c.configureEEBus(eebusConfig)
			if err != nil {
				return err
			}

			eebusYaml, err := yaml.Marshal(eebusConfig)
			if err != nil {
				return err
			}
			c.configuration.config.EEBUS = string(eebusYaml)
			fmt.Println()
			fmt.Println("--------------------------------------------")
		}

		fmt.Println()
		fmt.Println(c.localizedString("Requirements_EEBUS_Pairing", nil))
		fmt.Scanln()
	}

	return nil
}

func (c *CmdConfigure) askSponsortoken(required bool) error {
	fmt.Println("-- Sponsorship -----------------------------")
	if required {
		fmt.Println()
		fmt.Println(c.localizedString("Requirements_Sponsorship_Title", nil))
	} else {
		fmt.Println()
		fmt.Println(c.localizedString("Requirements_Sponsorship_Optional_Title", nil))
	}
	fmt.Println()
	if !c.askYesNo(c.localizedString("Requirements_Sponsorship_Token", nil)) {
		return c.errItemNotPresent
	}

	sponsortoken := c.askValue(question{
		label:    c.localizedString("Requirements_Sponsorship_Token_Input", nil),
		mask:     true,
		required: true})
	c.configuration.config.SponsorToken = sponsortoken
	sponsor.Subject = sponsortoken
	fmt.Println()
	fmt.Println("--------------------------------------------")

	return nil
}

func (c *CmdConfigure) configureMQTT() (map[string]interface{}, error) {
	fmt.Println()
	fmt.Println("-- MQTT Broker ----------------------------")

	var err error

	for ok := true; ok; {
		fmt.Println()
		host := c.askValue(question{
			label:    c.localizedString("UserFriendly_Host_Name", nil),
			mask:     false,
			required: true})

		port := c.askValue(question{
			label:    c.localizedString("UserFriendly_Port_Name", nil),
			mask:     false,
			required: true})

		user := c.askValue(question{
			label:    c.localizedString("UserFriendly_User_Name", nil),
			mask:     false,
			required: false})

		password := c.askValue(question{
			label:    c.localizedString("UserFriendly_Password_Name", nil),
			mask:     true,
			required: false})

		fmt.Println()
		fmt.Println("-------------------------------------------")

		broker := fmt.Sprintf("%s:%s", host, port)

		mqttConfig := map[string]interface{}{
			"broker":   broker,
			"user":     user,
			"password": password,
		}

		log := util.NewLogger("mqtt")

		if mqtt.Instance, err = mqtt.RegisteredClient(log, broker, user, password, "", 1); err == nil {
			return mqttConfig, nil
		}

		fmt.Println()
		question := c.localizedString("TestingMQTTFailed", nil)
		if !c.askYesNo(question) {
			return nil, fmt.Errorf("failed configuring mqtt: %w", err)
		}
	}

	return nil, fmt.Errorf("failed configuring mqtt: %w", err)
}

// fetchElements returns template items of a given class
func (c *CmdConfigure) fetchElements(deviceCategory DeviceCategory) []templates.Template {
	var items []templates.Template
	for _, tmpl := range templates.ByClass(DeviceCategories[deviceCategory].class.String()) {
		if len(tmpl.Params) == 0 {
			continue
		}

		for _, t := range tmpl.Titles(c.lang) {
			titleTmpl := templates.Template{
				TemplateDefinition: tmpl.TemplateDefinition,
				ConfigDefaults:     tmpl.ConfigDefaults,
				Lang:               c.lang,
			}
			title := t
			groupTitle := titleTmpl.GroupTitle()
			if groupTitle != "" {
				title += " [" + groupTitle + "]"
			}
			titleTmpl.SetTitle(title)

			if deviceCategory == DeviceCategoryGuidedSetup {
				if tmpl.GuidedSetup.Enable {
					items = append(items, titleTmpl)
				}
			} else {
				if len(DeviceCategories[deviceCategory].categoryFilter) == 0 ||
					c.paramChoiceContains(tmpl.Params, templates.ParamUsage, DeviceCategories[deviceCategory].categoryFilter.String()) {
					items = append(items, titleTmpl)
				}
			}
		}
	}

	sort.Slice(items[:], func(i, j int) bool {
		// sort generic templates to the bottom
		if items[i].Group != "" && items[j].Group == "" {
			return false
		}
		if items[i].Group == "" && items[j].Group != "" {
			return true
		}
		if items[i].Group != items[j].Group {
			return strings.ToLower(items[i].GroupTitle()) < strings.ToLower(items[j].GroupTitle())
		}
		return strings.ToLower(items[i].Title()) < strings.ToLower(items[j].Title())
	})

	return items
}

// paramChoiceContains is a helper function to check if a param choice contains a given value
func (c *CmdConfigure) paramChoiceContains(params []templates.Param, name, filter string) bool {
	choices := c.paramChoiceValues(params, name)

	return funk.ContainsString(choices, filter)
}

// paramChoiceValues provides a list of possible values for a param choice
func (c *CmdConfigure) paramChoiceValues(params []templates.Param, name string) []string {
	choices := []string{}

	for _, item := range params {
		if item.Name != name {
			continue
		}

		choices = append(choices, item.Choice...)
	}

	return choices
}

// processConfig processes an EVCC configuration item
// Returns:
//   a map with param name and values
func (c *CmdConfigure) processConfig(templateItem templates.Template, deviceCategory DeviceCategory) map[string]interface{} {
	fmt.Println()
	fmt.Println(c.localizedString("Config_Title", nil))
	fmt.Println()

	c.processModbusConfig(&templateItem, deviceCategory)

	return c.processParams(templateItem, templateItem.Params, deviceCategory)
}

// process a list of params
func (c *CmdConfigure) processParams(templateItem templates.Template, params []templates.Param, deviceCategory DeviceCategory) map[string]interface{} {
	usageFilter := DeviceCategories[deviceCategory].categoryFilter

	additionalConfig := make(map[string]interface{})

	for _, param := range params {
		if param.Dependencies != nil {
			valid := true
			for _, dep := range param.Dependencies {
				i, valueParam := templateItem.ParamByName(dep.Name)
				if i == -1 {
					break
				}

				value := valueParam.Value
				switch dep.Check {
				case templates.DependencyCheckEmpty:
					if additionalConfig[dep.Name] != nil {
						valid = additionalConfig[dep.Name] == ""
					} else {
						valid = value == ""
					}
				case templates.DependencyCheckNotEmpty:
					if additionalConfig[dep.Name] != nil {
						valid = additionalConfig[dep.Name] != ""
					} else {
						valid = value != ""
					}
				case templates.DependencyCheckEqual:
					if additionalConfig[dep.Name] != nil {
						valid = additionalConfig[dep.Name] == dep.Value
					} else {
						valid = value == dep.Value
					}
				}
				if !valid {
					break
				}
			}
			if !valid {
				continue
			}
		}

		switch param.Name {
		case templates.ParamModbus:
			additionalConfig[param.Name] = param.Value

		case templates.ParamUsage:
			if usageFilter != "" {
				additionalConfig[param.Name] = usageFilter.String()
			}
		default:
			if !c.advancedMode && param.Advanced {
				continue
			}

			if param.Deprecated {
				continue
			}

			if param.Hidden && param.Default != "" {
				additionalConfig[param.Name] = param.Default
				continue
			}

			switch param.ValueType {
			case templates.ParamValueTypeStringList:
				additionalConfig[param.Name] = c.processListInputConfig(param)
			default:
				additionalConfig[param.Name] = c.processInputConfig(param)
			}
		}
	}

	return additionalConfig
}

// handle user input of multiple items in a list
func (c *CmdConfigure) processListInputConfig(param templates.Param) []string {
	var values []string

	// ask for values until the user decides to stop
	for ok := true; ok; {
		newValue := c.processInputConfig(param)
		values = append(values, newValue)

		if newValue == "" {
			break
		}

		if !c.askYesNo("  " + c.localizedString("Config_AddAnotherValue", nil)) {
			break
		}
	}

	return values
}

// handle user input for a simple one value input
func (c *CmdConfigure) processInputConfig(param templates.Param) string {
	label := param.Name
	langLabel := param.Description.String(c.lang)
	if langLabel != "" {
		label = langLabel
	}

	return c.askValue(question{
		label:        label,
		defaultValue: param.Default,
		exampleValue: param.Example,
		help:         param.Help.String(c.lang),
		valueType:    param.ValueType,
		mask:         param.Mask,
		required:     param.Required})
}

// handle user input for a device modbus configuration
func (c *CmdConfigure) processModbusConfig(templateItem *templates.Template, deviceCategory DeviceCategory) {
	var choices []string
	var choiceTypes []string

	modbusIndex, modbusParam := templateItem.ParamByName(templates.ParamModbus)
	if modbusIndex == -1 {
		return
	}

	config := templateItem.ConfigDefaults.Config.Modbus

	for _, choice := range modbusParam.Choice {
		if config.Interfaces[choice] == nil {
			continue
		}

		for _, itype := range config.Interfaces[choice] {
			title := config.Types[itype].Description
			choices = append(choices, title.String(c.lang))
			choiceTypes = append(choiceTypes, itype)
		}
	}

	if len(choices) == 0 {
		return
	}

	// ask for modbus interface type
	var index int
	if len(choices) > 1 {
		index, _ = c.askChoice(c.localizedString("Config_ModbusInterface", nil), choices)
	}

	values := make(map[string]interface{})
	templateItem.Params[modbusIndex].Value = choiceTypes[index]
	// add the interface type specific modbus params
	templateItem.ModbusParams(choiceTypes[index], values)
	// Update the modbus default values
	templateItem.ModbusValues(templates.TemplateRenderModeInstance, values)
}
