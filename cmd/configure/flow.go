package configure

import (
	"fmt"

	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/vehicle"
	"gopkg.in/yaml.v3"
)

func (c *CmdConfigure) configureDeviceGuidedSetup() {
	var repeat bool = true
	var err error

	var values map[string]interface{}
	var deviceCategory DeviceCategory
	var supportedDeviceCategories []DeviceCategory
	var templateItem templates.Template

	deviceItem := device{}

	for ok := true; ok; ok = repeat {
		fmt.Println()

		templateItem, err = c.processDeviceSelection(DeviceCategoryGuidedSetup)
		if err != nil {
			return
		}

		usageFound, usageChoices := c.paramChoiceValues(templateItem.Params, templates.ParamUsage)
		if !usageFound {
			fmt.Println("error")
			return
		}
		if len(usageChoices) == 0 {
			usageChoices = []string{DeviceCategoryGridMeter, DeviceCategoryPVMeter, DeviceCategoryBatteryMeter}
		}

		supportedDeviceCategories = []DeviceCategory{}

		for _, usage := range usageChoices {
			switch usage {
			case DeviceCategoryGridMeter:
				supportedDeviceCategories = append(supportedDeviceCategories, DeviceCategoryGridMeter)
			case DeviceCategoryPVMeter:
				supportedDeviceCategories = append(supportedDeviceCategories, DeviceCategoryPVMeter)
			case DeviceCategoryBatteryMeter:
				supportedDeviceCategories = append(supportedDeviceCategories, DeviceCategoryBatteryMeter)
			}
		}

		// we only ask for the configuration for the first usage
		deviceCategory = supportedDeviceCategories[0]

		values := c.processConfig(templateItem.Params, deviceCategory, false)

		deviceItem, err = c.processDeviceValues(values, templateItem, deviceItem, deviceCategory)
		if err != nil {
			if err != ErrDeviceNotValid {
				fmt.Println()
				fmt.Println("Fehler: ", err)
			}
			fmt.Println()
			if !c.askConfigFailureNextStep() {
				return
			}
			continue
		}

		repeat = false
	}

	c.configuration.AddDevice(deviceItem, deviceCategory)

	for _, deviceCategory = range supportedDeviceCategories[1:] {
		deviceItem, err := c.processDeviceValues(values, templateItem, deviceItem, deviceCategory)
		if err != nil {
			continue
		}

		c.configuration.AddDevice(deviceItem, deviceCategory)
	}

	fmt.Println()
	fmt.Println(templateItem.Description + " wurde erfolgreich hinzugefügt.")

	c.configureLinkedTypes(templateItem)
}

func (c *CmdConfigure) configureLinkedTypes(templateItem templates.Template) {
	var repeat bool = true

	linkedTemplates := templateItem.GuidedSetup.Linked

	if linkedTemplates == nil {
		return
	}

	for _, linkedTemplate := range linkedTemplates {
		for ok := true; ok; ok = repeat {
			deviceItem := device{}

			linkedTemplateItem := templates.ByType(linkedTemplate.Type, DeviceClassMeter)
			if len(linkedTemplateItem.Params) == 0 || linkedTemplate.Usage == "" {
				return
			}

			category := DeviceCategory(linkedTemplate.Usage)

			fmt.Println()
			if !c.askYesNo("Möchtest du " + DeviceCategories[category].article + " " + linkedTemplateItem.Description + " als " + DeviceCategories[category].title + " hinzufügen") {
				repeat = false
				continue
			}

			values := c.processConfig(linkedTemplateItem.Params, category, false)
			deviceItem, err := c.processDeviceValues(values, linkedTemplateItem, deviceItem, category)
			if err != nil {
				if err != ErrDeviceNotValid {
					fmt.Println()
					fmt.Println("Fehler: ", err)
				}
				fmt.Println()
				if c.askConfigFailureNextStep() {
					continue
				}

			} else {
				c.configuration.AddDevice(deviceItem, category)

				fmt.Println(linkedTemplateItem.Description + " wurde erfolgreich hinzugefügt.")
			}
			repeat = false
		}
		repeat = true
	}
}

func (c *CmdConfigure) configureDeviceCategory(deviceCategory DeviceCategory) (device, error) {
	fmt.Println()
	fmt.Printf("- %s konfigurieren\n", DeviceCategories[deviceCategory].title)

	var repeat bool = true

	device := device{
		Name:  DeviceCategories[deviceCategory].defaultName,
		Title: "",
		Yaml:  "",
	}

	deviceDescription := ""

	for ok := true; ok; ok = repeat {
		fmt.Println()

		templateItem, err := c.processDeviceSelection(deviceCategory)
		if err != nil {
			return device, ErrItemNotPresent
		}

		deviceDescription = templateItem.Description
		values := c.processConfig(templateItem.Params, deviceCategory, false)

		device, err = c.processDeviceValues(values, templateItem, device, deviceCategory)
		if err != nil {
			if err != ErrDeviceNotValid {
				fmt.Println()
				fmt.Println("Fehler: ", err)
			}
			fmt.Println()
			if !c.askConfigFailureNextStep() {
				return device, err
			}
			continue
		}

		repeat = false
	}

	c.configuration.AddDevice(device, deviceCategory)

	deviceTitle := ""
	if device.Title != "" {
		deviceTitle = " " + device.Title
	}

	fmt.Println(deviceDescription + deviceTitle + " wurde erfolgreich hinzugefügt.")

	return device, nil
}

// create a configured device from a template so we can test it
func (c *CmdConfigure) configureDevice(deviceCategory DeviceCategory, device templates.Template, values map[string]interface{}) (interface{}, error) {
	b, err := device.RenderResult(false, values)
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
