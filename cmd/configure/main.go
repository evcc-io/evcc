package configure

import (
	_ "embed"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/viper"
)

type CmdConfigure struct {
	configuration config
	log           *util.Logger
}

// start the interactive configuration
func (c *CmdConfigure) Run(log *util.Logger) {
	c.log = log
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	c.log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	fmt.Println()
	fmt.Println("The next steps will guide throught the creation of a EVCC configuration file.")
	fmt.Println("Please be aware that this process does not cover all possible scenarios.")
	fmt.Println("You can stop the process by pressing ctrl-c.")
	fmt.Println()
	fmt.Println("Let's start:")

	c.configureDevices()
	c.configureLoadpoints()
	c.configureSite()

	yaml, err := c.renderConfiguration()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Your configuration:")
	fmt.Println()
	fmt.Println(string(yaml))
}

// ask devuce specfic questions
func (c *CmdConfigure) configureDevices() {
	categoriesOrder := []string{DeviceCategoryCharger, DeviceCategoryGridMeter, DeviceCategoryPVMeter, DeviceCategoryBatteryMeter, DeviceCategoryVehicle}
	for _, category := range categoriesOrder {
		fmt.Println()
		if !c.askYesNo("Do you want to add a " + DeviceCategories[category].title + "?") {
			continue
		}

		var deviceInCategoryIndex int = 0

		for ok := true; ok; {
			deviceInCategoryIndex++

			err := c.configureDeviceCategory(category, deviceInCategoryIndex)
			if err != nil {
				break
			}

			if category == DeviceCategoryGridMeter {
				break
			}
			if !c.askYesNo("Do you want to add another " + DeviceCategories[category].title + "?") {
				break
			}
		}
	}
}

// ask loadpoint specific questions
func (c *CmdConfigure) configureLoadpoints() {
	if len(c.configuration.Chargers) == 0 {
		return
	}

	chargeMeterIndex := 0

	fmt.Println()
	fmt.Println("- Configure your loadpoint(s)")

	for index, charger := range c.configuration.Chargers {
		fmt.Println()
		fmt.Printf("%#v\n", charger)
		fmt.Printf("- Configure a loadpoint for the %d. wallbox %s (%s)\n", (index + 1), charger.Title, charger.Name)

		loadpointTitle := c.askValue("Loadpoint title", defaultTitleLoadpoint, "", nil, false, true)
		loadpoint := loadpoint{
			Title:      loadpointTitle,
			Charger:    charger.Name,
			Phases:     3,
			MinCurrent: 6,
		}

		if !charger.ChargerHasMeter {
			if c.askYesNo("The charger does not provide charging power. Do you have a meter installed that can be used instead?") {
				chargeMeterIndex++
				err := c.configureDeviceCategory(DeviceCategoryChargeMeter, chargeMeterIndex)
				if err != nil {
					break
				}

				for _, device := range c.configuration.Meters {
					if device.Name == fmt.Sprintf("%s%d", defaultNameChargeMeter, chargeMeterIndex) {
						loadpoint.ChargeMeter = device.Name
						break
					}
				}
			}
		}

		vehicleAmount := len(c.configuration.Vehicles)
		if vehicleAmount == 1 {
			loadpoint.Vehicles = append(loadpoint.Vehicles, c.configuration.Vehicles[0].Name)
		} else if vehicleAmount > 1 {
			for _, vehicle := range c.configuration.Vehicles {
				fmt.Printf("%#v\n", vehicle)
				if c.askYesNo("Will the vehicle " + vehicle.Title + " charge here?") {
					loadpoint.Vehicles = append(loadpoint.Vehicles, vehicle.Name)
				}
			}
		}

		powerChoices := []string{"3,6kW", "11kW", "22kW"}
		powerIndex, _ := c.askChoice("What is the maximum power the wallbox can provide?", powerChoices)
		switch powerIndex {
		case 0:
			loadpoint.MaxCurrent = 16
			loadpoint.Phases = 1
		case 1:
			loadpoint.MaxCurrent = 16
		case 2:
			loadpoint.MaxCurrent = 32
		}

		chargingModes := []string{string(api.ModeOff), string(api.ModeNow), string(api.ModeMinPV), string(api.ModePV)}
		_, modeChoice := c.askChoice("What should be the default charging mode when an EV is connected?", chargingModes)
		loadpoint.Mode = modeChoice

		c.configuration.Loadpoints = append(c.configuration.Loadpoints, loadpoint)
	}
}

// ask site specific questions
func (c *CmdConfigure) configureSite() {
	fmt.Println()
	fmt.Println("- Configure your site")

	c.configuration.Site.Title = c.askValue("Site title", defaultTitleSite, "", nil, false, true)
}
