package configure

import (
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/util/templates"
)

// configureDeviceGuidedSetup lets the user choose a device that is set to support guided setup
// these are typically devices that
// - contain multiple usages but have the same parameters like host, port, etc.
// - devices that typically are installed with additional specific devices (e.g. SMA Home Manager with SMA Inverters)
func (c *CmdConfigure) configureDeviceGuidedSetup() {
	var err error

	var values map[string]interface{}
	var deviceCategory DeviceCategory
	var supportedDeviceCategories []DeviceCategory
	var templateItem templates.Template

	deviceItem := device{}

	for ok := true; ok; {
		fmt.Println()

		templateItem, err = c.processDeviceSelection(DeviceCategoryGuidedSetup)
		if err != nil {
			return
		}

		usageChoices := c.paramChoiceValues(templateItem.Params, templates.ParamUsage)
		if len(usageChoices) == 0 {
			panic("ERROR: Device template is missing valid usages!")
		}
		if len(usageChoices) == 0 {
			usageChoices = []string{string(DeviceCategoryGridMeter), string(DeviceCategoryPVMeter), string(DeviceCategoryBatteryMeter)}
		}

		supportedDeviceCategories = []DeviceCategory{}

		for _, usage := range usageChoices {
			switch usage {
			case string(DeviceCategoryGridMeter):
				supportedDeviceCategories = append(supportedDeviceCategories, DeviceCategoryGridMeter)
			case string(DeviceCategoryPVMeter):
				supportedDeviceCategories = append(supportedDeviceCategories, DeviceCategoryPVMeter)
			case string(DeviceCategoryBatteryMeter):
				supportedDeviceCategories = append(supportedDeviceCategories, DeviceCategoryBatteryMeter)
			}
		}

		// we only ask for the configuration for the first usage
		deviceCategory = supportedDeviceCategories[0]

		values = c.processConfig(templateItem.Params, deviceCategory)

		deviceItem, err = c.processDeviceValues(values, templateItem, deviceItem, deviceCategory)
		if err != nil {
			if err != c.errDeviceNotValid {
				fmt.Println()
				fmt.Println(err)
			}
			fmt.Println()
			if !c.askConfigFailureNextStep() {
				return
			}
			continue
		}

		break
	}

	c.configuration.AddDevice(deviceItem, deviceCategory)

	if len(supportedDeviceCategories) > 1 {
		for _, additionalCategory := range supportedDeviceCategories[1:] {
			values[templates.ParamUsage] = additionalCategory.String()
			deviceItem, err := c.processDeviceValues(values, templateItem, deviceItem, additionalCategory)
			if err != nil {
				continue
			}

			c.configuration.AddDevice(deviceItem, additionalCategory)
		}
	}

	fmt.Println()
	fmt.Println(templateItem.Description + " " + c.localizedString("Device_Added", nil))

	c.configureLinkedTypes(templateItem)
}

// configureLinkedTypes lets the user configure devices that are marked as being linked to a guided device
// e.g. SMA Inverters, Energy Meter with SMA Home Manager
func (c *CmdConfigure) configureLinkedTypes(templateItem templates.Template) {
	linkedTemplates := templateItem.GuidedSetup.Linked

	if linkedTemplates == nil {
		return
	}

	for _, linkedTemplate := range linkedTemplates {
		linkedTemplateItem, err := templates.ByTemplate(linkedTemplate.Template, string(DeviceClassMeter))
		if err != nil {
			fmt.Println("Error: " + err.Error())
			return
		}
		if len(linkedTemplateItem.Params) == 0 || linkedTemplate.Usage == "" {
			break
		}

		category := DeviceCategory(linkedTemplate.Usage)

		localizeMap := localizeMap{
			"Linked":     linkedTemplateItem.Description,
			"Article":    DeviceCategories[category].article,
			"Additional": DeviceCategories[category].additional,
			"Category":   DeviceCategories[category].title,
		}

		fmt.Println()
		if !c.askYesNo(c.localizedString("AddLinkedDeviceInCategory", localizeMap)) {
			continue
		}

		for ok := true; ok; {
			c.configureLinkedTemplate(linkedTemplateItem, category)

			if !linkedTemplate.Multiple {
				break
			}

			fmt.Println()
			if !c.askYesNo(c.localizedString("AddAnotherLinkedDeviceInCategory", localizeMap)) {
				break
			}
		}
	}
}

// configureLinkedTemplate lets the user configure a device that is marked as being linked to a guided device
func (c *CmdConfigure) configureLinkedTemplate(template templates.Template, category DeviceCategory) {
	for ok := true; ok; {
		deviceItem := device{}

		values := c.processConfig(template.Params, category)
		deviceItem, err := c.processDeviceValues(values, template, deviceItem, category)
		if err != nil {
			if !errors.Is(err, c.errDeviceNotValid) {
				fmt.Println()
				fmt.Println(err)
			}
			fmt.Println()
			if c.askConfigFailureNextStep() {
				continue
			}

		} else {
			c.configuration.AddDevice(deviceItem, category)

			fmt.Println()
			fmt.Println(template.Description + " " + c.localizedString("Device_Added", nil))
		}
		break
	}
}

// configureDeviceCategory lets the user select and configure a device from a specific category
func (c *CmdConfigure) configureDeviceCategory(deviceCategory DeviceCategory) (device, error) {
	fmt.Println()
	fmt.Printf("- %s %s\n", c.localizedString("Device_Configure", nil), DeviceCategories[deviceCategory].title)

	device := device{
		Name: DeviceCategories[deviceCategory].defaultName,
	}

	var deviceDescription string

	// repeat until the device is added or the user chooses to continue without adding a device
	for ok := true; ok; {
		fmt.Println()

		templateItem, err := c.processDeviceSelection(deviceCategory)
		if err != nil {
			return device, c.errItemNotPresent
		}

		deviceDescription = templateItem.Description
		values := c.processConfig(templateItem.Params, deviceCategory)

		device, err = c.processDeviceValues(values, templateItem, device, deviceCategory)
		if err != nil {
			if err != c.errDeviceNotValid {
				fmt.Println()
				fmt.Println(err)
			}
			// ask if the user wants to add the
			fmt.Println()
			if !c.askConfigFailureNextStep() {
				return device, err
			}
			continue
		}

		break
	}

	c.configuration.AddDevice(device, deviceCategory)

	var deviceTitle string
	if device.Title != "" {
		deviceTitle = " " + device.Title
	}

	fmt.Println()
	fmt.Println(deviceDescription + deviceTitle + " " + c.localizedString("Device_Added", nil))

	return device, nil
}
