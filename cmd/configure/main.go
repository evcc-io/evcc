package configure

import (
	_ "embed"
	"fmt"

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

	fmt.Println()
	fmt.Println("- Configure your loadpoint(s)")

	for _, charger := range c.configuration.Chargers {
		fmt.Println()
		fmt.Printf("- Configure a loadpoint for the wallbox named %s\n", charger.Name)

		loadpointTitle := c.askValue("Loadpoint title", defaultTitleLoadpoint, "", nil, false, true)
		loadpoint := loadpoint{
			Title:   loadpointTitle,
			Charger: charger.Name,
		}

		vehicleAmount := len(c.configuration.Vehicles)
		if vehicleAmount == 1 {
			loadpoint.Vehicles = append(loadpoint.Vehicles, c.configuration.Vehicles[0].Name)
		} else if vehicleAmount > 1 {
			for _, vehicle := range c.configuration.Vehicles {
				if c.askYesNo("Will the vehicle named " + vehicle.Name + " charge here?") {
					loadpoint.Vehicles = append(loadpoint.Vehicles, vehicle.Name)
				}
			}
		}
		c.configuration.Loadpoints = append(c.configuration.Loadpoints, loadpoint)
	}
}

// ask site specific questions
func (c *CmdConfigure) configureSite() {
	fmt.Println()
	fmt.Println("- Configure your site")

	c.configuration.Site.Title = c.askValue("Site title", defaultTitleSite, "", nil, false, true)
}
