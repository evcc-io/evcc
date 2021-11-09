package configure

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/util"
)

type CmdConfigure struct {
	configuration config
	log           *util.Logger
}

// start the interactive configuration
func (c *CmdConfigure) Run(log *util.Logger, logLevel string) {
	c.log = log

	defaultLevel := "error"
	if logLevel != "" {
		defaultLevel = logLevel
	}
	util.LogLevel(defaultLevel, map[string]string{})
	c.log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	fmt.Println()
	fmt.Println("Die nächsten Schritte führen durch die Einrichtung einer Konfigurationsdatei für evcc.")
	fmt.Println("Beachte dass dieser Prozess nicht alle möglichen Szenarien berücksichtigen kann.")
	fmt.Println("Durch Drücken von CTRL-C kann der Prozess abgebrochen werden.")
	fmt.Println()
	fmt.Println("Auf geht`s:")

	c.configureDevices(DeviceCategoryGridMeter, false)
	c.configureDevices(DeviceCategoryPVMeter, true)
	c.configureDevices(DeviceCategoryBatteryMeter, true)
	c.configureDevices(DeviceCategoryVehicle, true)
	c.configureLoadpoints()
	c.configureSite()

	yaml, err := c.renderConfiguration()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	fmt.Println()

	filename := "evcc.yaml"

	for ok := true; ok; {
		_, err := os.Open(filename)
		if errors.Is(err, os.ErrNotExist) {
			break
		}

		fmt.Printf("Die Datei %s existiert bereits.\n", filename)
		if c.askYesNo("Soll die Datei überschrieben werden") {
			break
		}

		filename = c.askValue("Gib einen neuen Dateinamen an", "", "", nil, "string", false, true)
	}

	err = os.WriteFile(filename, yaml, 0755)
	if err != nil {
		fmt.Printf("Die Konfiguration konnte nicht in die Datei %s gespeicher werden", filename)
		c.log.FATAL.Fatal(err)
	}
	fmt.Printf("Deine Konfiguration wurde erfolgreich in die Datei %s gespeichert.\n", filename)
}

// ask devuce specfic questions
func (c *CmdConfigure) configureDevices(deviceCategory string, askMultiple bool) []device {
	fmt.Println()
	if !c.askYesNo("Möchtest du " + DeviceCategories[deviceCategory].article + " " + DeviceCategories[deviceCategory].title + " hinzufügen") {
		return nil
	}

	var devices []device
	var deviceInCategoryIndex int = 0

	for ok := true; ok; {
		deviceInCategoryIndex++

		device, err := c.configureDeviceCategory(deviceCategory, deviceInCategoryIndex)
		if err != nil {
			break
		}
		devices = append(devices, device)

		if !askMultiple {
			break
		}

		if !c.askYesNo("Möchstest du noch " + DeviceCategories[deviceCategory].article + " " + DeviceCategories[deviceCategory].title + "hinzufügen") {
			break
		}
	}

	return devices
}

// ask loadpoint specific questions
func (c *CmdConfigure) configureLoadpoints() {
	fmt.Println()
	fmt.Println("- Ladepunkt(e) einrichten")

	chargerIndex := 0
	chargeMeterIndex := 0

	for ok := true; ok; {

		loadpointTitle := c.askValue("Titel des Ladepunktes", defaultTitleLoadpoint, "", nil, templates.ParamValueTypeString, false, true)
		loadpoint := loadpoint{
			Title:      loadpointTitle,
			Phases:     3,
			MinCurrent: 6,
		}

		chargerIndex++
		charger, err := c.configureDeviceCategory(DeviceCategoryCharger, chargerIndex)
		if err != nil {
			break
		}

		loadpoint.Charger = charger.Name

		if !charger.ChargerHasMeter {
			if c.askYesNo("Die Wallbox hat keinen Ladestromzähler. Hast du einen externen Zähler dafür installiert der verwendet werden kann") {
				chargeMeterIndex++
				chargeMeter, err := c.configureDeviceCategory(DeviceCategoryChargeMeter, chargeMeterIndex)
				if err != nil {
					break
				}

				loadpoint.ChargeMeter = chargeMeter.Name
			}
		}

		vehicleAmount := len(c.configuration.Vehicles)
		if vehicleAmount == 1 {
			loadpoint.Vehicles = append(loadpoint.Vehicles, c.configuration.Vehicles[0].Name)
		} else if vehicleAmount > 1 {
			for _, vehicle := range c.configuration.Vehicles {
				if c.askYesNo("Wird das Fahrzeug " + vehicle.Title + " hier laden?") {
					loadpoint.Vehicles = append(loadpoint.Vehicles, vehicle.Name)
				}
			}
		}

		powerChoices := []string{"3,6kW", "11kW", "22kW", "Other"}
		powerIndex, _ := c.askChoice("Was ist die maximale Leistung, welche die Wallbox zur Verfügung stellen kann?", powerChoices)
		switch powerIndex {
		case 0:
			loadpoint.MaxCurrent = 16
			loadpoint.Phases = 1
		case 1:
			loadpoint.MaxCurrent = 16
		case 2:
			loadpoint.MaxCurrent = 32
		case 3:
			amperage := c.askValue("Was ist die maximale Stromstärke welche die Wallbox auf einer Phase zur Verfügung stellen kann?", "", "", nil, templates.ParamValueTypeInt, false, true)
			loadpoint.MaxCurrent, _ = strconv.Atoi(amperage)

			phaseChoices := []string{"1", "2", "3"}
			phaseIndex, _ := c.askChoice("Mit wievielen Phasen ist die Wallbox angeschlossen?", phaseChoices)
			loadpoint.Phases = phaseIndex + 1
		}

		chargingModes := []string{string(api.ModeOff), string(api.ModeNow), string(api.ModeMinPV), string(api.ModePV)}
		_, modeChoice := c.askChoice("Was sollte der Standard-Lademodus sein, wenn ein Fahrzeug angeschlossen wird?", chargingModes)
		loadpoint.Mode = modeChoice

		c.configuration.Loadpoints = append(c.configuration.Loadpoints, loadpoint)

		if !c.askYesNo("Möchtest du einen weiteren Ladepunkt hinzufügen") {
			break
		}
	}
}

// ask site specific questions
func (c *CmdConfigure) configureSite() {
	fmt.Println()
	fmt.Println("- Richte deinen Standort ein")

	c.configuration.Site.Title = c.askValue("Titel des Standortes", defaultTitleSite, "", nil, templates.ParamValueTypeString, false, true)
}
