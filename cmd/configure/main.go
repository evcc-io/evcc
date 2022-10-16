package configure

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/evcc-io/evcc/hems/semp"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
)

//go:embed localization/de.toml
var lang_de string

//go:embed localization/en.toml
var lang_en string

type CmdConfigure struct {
	configuration Configure
	localizer     *i18n.Localizer
	log           *util.Logger

	lang                                 string
	advancedMode, expandedMode           bool
	addedDeviceIndex                     int
	errItemNotPresent, errDeviceNotValid error

	capabilitySMAHems bool
}

// Run starts the interactive configuration
func (c *CmdConfigure) Run(log *util.Logger, flagLang string, advancedMode, expandedMode bool, category string) {
	c.log = log
	c.advancedMode = advancedMode
	c.expandedMode = expandedMode

	c.log.INFO.Printf("evcc %s", server.FormattedVersion())

	bundle := i18n.NewBundle(language.German)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	if _, err := bundle.ParseMessageFileBytes([]byte(lang_de), "localization/de.toml"); err != nil {
		panic(err)
	}
	if _, err := bundle.ParseMessageFileBytes([]byte(lang_en), "localization/en.toml"); err != nil {
		panic(err)
	}

	c.lang = "de"
	systemLanguage, err := jibber_jabber.DetectLanguage()
	if err == nil {
		c.lang = systemLanguage
	}
	if flagLang != "" {
		c.lang = flagLang
	}

	c.localizer = i18n.NewLocalizer(bundle, c.lang)

	c.setDefaultTexts()

	fmt.Println()
	fmt.Println(c.localizedString("Intro", nil))

	if !c.advancedMode && category == "" {
		// ask the user for his knowledge, so advanced mode can also be turned on this way
		fmt.Println()
		flowIndex, _ := c.askChoice(c.localizedString("Flow_Mode", nil), []string{
			c.localizedString("Flow_Mode_Standard", nil),
			c.localizedString("Flow_Mode_Advanced", nil),
		})
		if flowIndex == 1 {
			c.advancedMode = true
		}
	}

	if !c.advancedMode && category == "" {
		c.flowNewConfigFile()
		return
	}

	if category != "" {
		for cat := range DeviceCategories {
			if cat == DeviceCategory(category) {
				c.flowSingleDevice(DeviceCategory(category))
				return
			}
		}

		panic("invalid category: " + category)
	}

	fmt.Println()
	flowIndex, _ := c.askChoice(c.localizedString("Flow_Type", nil), []string{
		c.localizedString("Flow_Type_NewConfiguration", nil),
		c.localizedString("Flow_Type_SingleDevice", nil),
	})
	switch flowIndex {
	case 0:
		c.flowNewConfigFile()
	case 1:
		c.flowSingleDevice("")
	}
}

// configureSingleDevice implements the flow for getting a single device configuration
func (c *CmdConfigure) flowSingleDevice(category DeviceCategory) {
	fmt.Println()
	fmt.Println(c.localizedString("Flow_SingleDevice_Setup", nil))
	fmt.Println()
	fmt.Println(c.localizedString("Flow_SingleDevice_Select", nil))

	// only consider the device categories that are marked for this flow
	categoryChoices := []string{
		DeviceCategories[DeviceCategoryGridMeter].title,
		DeviceCategories[DeviceCategoryPVMeter].title,
		DeviceCategories[DeviceCategoryBatteryMeter].title,
		DeviceCategories[DeviceCategoryChargeMeter].title,
		DeviceCategories[DeviceCategoryCharger].title,
		DeviceCategories[DeviceCategoryVehicle].title,
	}

	if category == "" {
		fmt.Println()
		_, categoryTitle := c.askChoice(c.localizedString("Flow_SingleDevice_Select", nil), categoryChoices)

		for item, data := range DeviceCategories {
			if data.title == categoryTitle {
				category = item
				break
			}
		}
	}

	devices := c.configureDevices(category, false, false)
	for _, item := range devices {
		fmt.Println()
		fmt.Println(c.localizedString("Flow_SingleDevice_Config", localizeMap{}))
		fmt.Println()

		scanner := bufio.NewScanner(strings.NewReader(item.Yaml))
		for scanner.Scan() {
			fmt.Println("  " + scanner.Text())
		}
	}
	fmt.Println()
}

// configureNewConfigFile implements the flow for creating a new configuration file
func (c *CmdConfigure) flowNewConfigFile() {
	fmt.Println()
	fmt.Println(c.localizedString("Flow_NewConfiguration_Setup", nil))
	fmt.Println()
	fmt.Println(c.localizedString("Flow_NewConfiguration_Select", localizeMap{"ItemNotPresent": c.localizedString("ItemNotPresent", nil)}))
	c.configureDeviceGuidedSetup()

	_ = c.configureDevices(DeviceCategoryGridMeter, true, false)
	_ = c.configureDevices(DeviceCategoryPVMeter, true, true)
	_ = c.configureDevices(DeviceCategoryBatteryMeter, true, true)
	_ = c.configureDevices(DeviceCategoryVehicle, true, true)

	c.configureLoadpoints()
	c.configureSite()

	// check if SMA HEMS is available and ask the user if it should be added
	if c.capabilitySMAHems {
		c.configureSMAHems()
	}

	if c.advancedMode && c.configuration.config.SponsorToken == "" {
		_ = c.askSponsortoken(false, false)
	}

	yaml, err := c.configuration.RenderConfiguration()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	fmt.Println()

	filename := DefaultConfigFilename

	for {
		file, err := os.OpenFile(filename, os.O_WRONLY, 0666)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		file.Close()
		// in case of permission error, we can't write to the file anyway
		if os.IsPermission(err) {
			fmt.Println(c.localizedString("File_Permissions", localizeMap{"FileName": filename}))
		} else {
			if c.askYesNo(c.localizedString("File_Exists", localizeMap{"FileName": filename})) {
				break
			}
		}

		filename = c.askValue(question{
			label:        c.localizedString("File_NewFilename", nil),
			exampleValue: "evcc_neu.yaml",
			required:     true,
		})
	}

	err = os.WriteFile(filename, yaml, 0o755)
	if err != nil {
		fmt.Printf("%s: ", c.localizedString("File_Error_SaveFailed", localizeMap{"FileName": filename}))
		c.log.FATAL.Fatal(err)
	}
	fmt.Println(c.localizedString("File_SaveSuccess", localizeMap{"FileName": filename}))
}

// configureDevices asks device specific questions
func (c *CmdConfigure) configureDevices(deviceCategory DeviceCategory, askAdding, askMultiple bool) []device {
	var devices []device

	if deviceCategory == DeviceCategoryGridMeter && c.configuration.MetersOfCategory(deviceCategory) > 0 {
		return nil
	}

	localizeMap := localizeMap{
		"Article":    DeviceCategories[deviceCategory].article,
		"Additional": DeviceCategories[deviceCategory].additional,
		"Category":   DeviceCategories[deviceCategory].title,
	}
	if askAdding {
		addDeviceText := c.localizedString("AddDeviceInCategory", localizeMap)
		if c.configuration.MetersOfCategory(deviceCategory) > 0 {
			addDeviceText = c.localizedString("AddAnotherDeviceInCategory", localizeMap)
		}

		fmt.Println()
		if !c.askYesNo(addDeviceText) {
			return nil
		}
	}

	for {
		device, capabilities, err := c.configureDeviceCategory(deviceCategory)
		if err != nil {
			break
		}
		devices = append(devices, device)

		c.processDeviceCapabilities(capabilities)

		if !askMultiple {
			break
		}

		fmt.Println()
		if !c.askYesNo(c.localizedString("AddAnotherDeviceInCategory", localizeMap)) {
			break
		}
	}

	return devices
}

// configureSMAHems asks the user if he wants to add the SMA HEMS
func (c *CmdConfigure) configureSMAHems() {
	// check if the system provides a machine-id
	if _, err := semp.UniqueDeviceID(); err != nil {
		return
	}

	fmt.Println()
	fmt.Println(c.localizedString("Flow_SMAHems_Setup", nil))

	fmt.Println()
	if !c.askYesNo(c.localizedString("Flow_SMAHems_Add", nil)) {
		return
	}

	// check if we need to setup a HEMS
	c.configuration.config.Hems = "type: sma\nAllowControl: false\n"
}

// configureLoadpoints asks loadpoint specific questions
func (c *CmdConfigure) configureLoadpoints() {
	fmt.Println()
	fmt.Println(c.localizedString("Loadpoint_Setup", nil))

	for {

		loadpointTitle := c.askValue(question{
			label:        c.localizedString("Loadpoint_Title", nil),
			defaultValue: c.localizedString("Loadpoint_DefaultTitle", nil),
			required:     true,
		})
		loadpoint := loadpoint{
			Title:      loadpointTitle,
			Phases:     3,
			MinCurrent: 6,
		}

		charger, capabilities, err := c.configureDeviceCategory(DeviceCategoryCharger)
		if err != nil {
			break
		}
		chargerHasMeter := charger.ChargerHasMeter

		loadpoint.Charger = charger.Name

		if !chargerHasMeter {
			if c.askYesNo(c.localizedString("Loadpoint_WallboxWOMeter", nil)) {
				chargeMeter, _, err := c.configureDeviceCategory(DeviceCategoryChargeMeter)
				if err == nil {
					loadpoint.ChargeMeter = chargeMeter.Name
					chargerHasMeter = true
				}
			}
		}

		vehicles := c.configuration.DevicesOfClass(templates.Vehicle)
		if len(vehicles) > 0 {
			fmt.Println()
			if c.askYesNo(c.localizedString("Loadpoint_VehicleDisableAutoDetection", nil)) {
				if len(vehicles) == 1 {
					loadpoint.Vehicle = vehicles[0].Name
				} else {
					fmt.Println()

					var vehicleTitles []string
					for _, vehicle := range vehicles {
						vehicleTitles = append(vehicleTitles, vehicle.Title)
					}

					vehicleIndex, _ := c.askChoice(c.localizedString("Loadpoint_VehicleSelection", nil), vehicleTitles)
					loadpoint.Vehicle = vehicles[vehicleIndex].Name
				}
			}
		}

		var minValue int = 6
		if slices.Contains(capabilities, templates.CapabilityISO151182) {
			minValue = 2
		}

		if c.advancedMode {
			fmt.Println()
			minAmperage := c.askValue(question{
				label:          c.localizedString("Loadpoint_WallboxMinAmperage", nil),
				valueType:      templates.ParamValueTypeNumber,
				minNumberValue: int64(minValue),
				maxNumberValue: 32,
				required:       true,
			})
			loadpoint.MinCurrent, _ = strconv.Atoi(minAmperage)
			maxAmperage := c.askValue(question{
				label:          c.localizedString("Loadpoint_WallboxMaxAmperage", nil),
				valueType:      templates.ParamValueTypeNumber,
				minNumberValue: 6,
				maxNumberValue: 32,
				required:       true,
			})
			loadpoint.MaxCurrent, _ = strconv.Atoi(maxAmperage)

			if !chargerHasMeter {
				phaseChoices := []string{"1", "2", "3"}
				fmt.Println()
				phaseIndex, _ := c.askChoice(c.localizedString("Loadpoint_WallboxPhases", nil), phaseChoices)
				loadpoint.Phases = phaseIndex + 1
			}
		} else {
			powerChoices := []string{
				c.localizedString("Loadpoint_WallboxPower36kW", nil),
				c.localizedString("Loadpoint_WallboxPower11kW", nil),
				c.localizedString("Loadpoint_WallboxPower22kW", nil),
				c.localizedString("Loadpoint_WallboxPowerOther", nil),
			}
			fmt.Println()
			powerIndex, _ := c.askChoice(c.localizedString("Loadpoint_WallboxMaxPower", nil), powerChoices)
			loadpoint.MinCurrent = minValue
			switch powerIndex {
			case 0:
				loadpoint.MaxCurrent = 16
				if !chargerHasMeter {
					loadpoint.Phases = 1
				}
			case 1:
				loadpoint.MaxCurrent = 16
				if !chargerHasMeter {
					loadpoint.Phases = 3
				}
			case 2:
				loadpoint.MaxCurrent = 32
				if !chargerHasMeter {
					loadpoint.Phases = 3
				}
			case 3:
				amperage := c.askValue(question{
					label:          c.localizedString("Loadpoint_WallboxMaxAmperage", nil),
					valueType:      templates.ParamValueTypeNumber,
					minNumberValue: int64(minValue),
					maxNumberValue: 32,
					required:       true,
				})
				loadpoint.MaxCurrent, _ = strconv.Atoi(amperage)

				if !chargerHasMeter {
					phaseChoices := []string{"1", "2", "3"}
					fmt.Println()
					phaseIndex, _ := c.askChoice(c.localizedString("Loadpoint_WallboxPhases", nil), phaseChoices)
					loadpoint.Phases = phaseIndex + 1
				}
			}
		}

		fmt.Println()
		loadpoint.Mode = c.askValue(question{valueType: templates.ParamValueTypeChargeModes, excludeNone: true})

		fmt.Println()
		loadpoint.ResetOnDisconnect = c.askValue(question{
			label:     c.localizedString("Loadpoint_ResetOnDisconnect", nil),
			valueType: templates.ParamValueTypeBool,
		})
		c.configuration.AddLoadpoint(loadpoint)

		fmt.Println()
		if !c.askYesNo(c.localizedString("Loadpoint_AddAnother", nil)) {
			break
		}
	}
}

// configureSite asks site specific questions
func (c *CmdConfigure) configureSite() {
	fmt.Println()
	fmt.Println(c.localizedString("Site_Setup", nil))

	siteTitle := c.askValue(question{
		label:        c.localizedString("Site_Title", nil),
		defaultValue: c.localizedString("Site_DefaultTitle", nil),
		required:     true,
	})
	c.configuration.config.Site.Title = siteTitle
}
