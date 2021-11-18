package configure

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/jibber_jabber"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed localization/de.toml
var lang_de string

type CmdConfigure struct {
	configuration Configure
	log           *util.Logger
}

// start the interactive configuration
func (c *CmdConfigure) Run(log *util.Logger, logLevel, flagLang string) {
	c.log = log

	defaultLevel := "error"
	if logLevel != "" {
		defaultLevel = logLevel
	}
	util.LogLevel(defaultLevel, map[string]string{})
	c.log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	bundle := i18n.NewBundle(language.German)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	_, err := bundle.ParseMessageFileBytes([]byte(lang_de), "localization/de.toml")
	if err != nil {
		panic(err)
	}

	lang := "de"
	systemLanguage, err := jibber_jabber.DetectLanguage()
	if err == nil {
		lang = systemLanguage
	}
	if flagLang != "" {
		lang = flagLang
	}

	localizer = i18n.NewLocalizer(bundle, lang)

	c.setDefaultTexts()

	fmt.Println()
	fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "Intro"}))
	fmt.Println()
	fmt.Println(localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    "Meters_Guide_Select",
		TemplateData: map[string]interface{}{"ItemNotPresent": localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: "ItemNotPresent"})},
	}))
	c.configureDeviceGuidedSetup()

	c.configureDevices(DeviceCategoryGridMeter, false)
	c.configureDevices(DeviceCategoryPVMeter, true)
	c.configureDevices(DeviceCategoryBatteryMeter, true)
	c.configureDevices(DeviceCategoryVehicle, true)
	c.configureLoadpoints()
	c.configureSite()

	yaml, err := c.configuration.RenderConfiguration()
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

		filename = c.askValue(question{
			label:        "Gib einen neuen Dateinamen an",
			exampleValue: "evcc_neu.yaml",
			required:     true})
	}

	err = os.WriteFile(filename, yaml, 0755)
	if err != nil {
		fmt.Printf("Die Konfiguration konnte nicht in die Datei %s gespeicher werden", filename)
		c.log.FATAL.Fatal(err)
	}
	fmt.Printf("Deine Konfiguration wurde erfolgreich in die Datei %s gespeichert.\n", filename)
}

// ask device specfic questions
func (c *CmdConfigure) configureDevices(deviceCategory DeviceCategory, askMultiple bool) []device {
	var devices []device

	if deviceCategory == DeviceCategoryGridMeter && c.configuration.MetersOfCategory(deviceCategory) > 0 {
		return nil
	}

	additionalMeter := ""
	if c.configuration.MetersOfCategory(deviceCategory) > 0 {
		additionalMeter = "noch "
	}

	fmt.Println()
	if !c.askYesNo("Möchtest du " + additionalMeter + DeviceCategories[deviceCategory].article + " " + DeviceCategories[deviceCategory].title + " hinzufügen") {
		return nil
	}

	for ok := true; ok; {
		device, err := c.configureDeviceCategory(deviceCategory)
		if err != nil {
			break
		}
		devices = append(devices, device)

		if !askMultiple {
			break
		}

		fmt.Println()
		if !c.askYesNo("Möchtest du noch " + DeviceCategories[deviceCategory].article + " " + DeviceCategories[deviceCategory].title + " hinzufügen") {
			break
		}
	}

	return devices
}

// ask loadpoint specific questions
func (c *CmdConfigure) configureLoadpoints() {
	fmt.Println()
	fmt.Println("- Ladepunkt(e) einrichten")

	for ok := true; ok; {

		loadpointTitle := c.askValue(question{
			label:        "Titel des Ladepunktes",
			defaultValue: defaultTitleLoadpoint,
			required:     true})
		loadpoint := loadpoint{
			Title:      loadpointTitle,
			Phases:     3,
			MinCurrent: 6,
		}

		charger, err := c.configureDeviceCategory(DeviceCategoryCharger)
		if err != nil {
			break
		}

		loadpoint.Charger = charger.Name

		if !charger.ChargerHasMeter {
			if c.askYesNo("Die Wallbox hat keinen Ladestromzähler. Hast du einen externen Zähler dafür installiert der verwendet werden kann") {
				chargeMeter, err := c.configureDeviceCategory(DeviceCategoryChargeMeter)
				if err != nil {
					break
				}

				loadpoint.ChargeMeter = chargeMeter.Name
			}
		}

		vehicles := c.configuration.DevicesOfClass(DeviceClassVehicle)
		if len(vehicles) == 1 {
			loadpoint.Vehicles = append(loadpoint.Vehicles, vehicles[0].Name)
		} else if len(vehicles) > 1 {
			for _, vehicle := range vehicles {
				if c.askYesNo("Wird das Fahrzeug " + vehicle.Title + " hier laden?") {
					loadpoint.Vehicles = append(loadpoint.Vehicles, vehicle.Name)
				}
			}
		}

		powerChoices := []string{"3,6kW", "11kW", "22kW", "Other"}
		fmt.Println()
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
			amperage := c.askValue(question{
				label:     "Was ist die maximale Stromstärke welche die Wallbox auf einer Phase zur Verfügung stellen kann?",
				valueType: templates.ParamValueTypeNumber,
				required:  true})
			loadpoint.MaxCurrent, _ = strconv.Atoi(amperage)

			phaseChoices := []string{"1", "2", "3"}
			fmt.Println()
			phaseIndex, _ := c.askChoice("Mit wievielen Phasen ist die Wallbox angeschlossen?", phaseChoices)
			loadpoint.Phases = phaseIndex + 1
		}

		chargingModes := []string{string(api.ModeOff), string(api.ModeNow), string(api.ModeMinPV), string(api.ModePV)}
		ladeModi := []string{"Aus", "Sofort (mit größtmöglicher Leistung)", "Min+PV (mit der kleinstmöglichen Leistung, schneller wenn genügend PV Überschuss vorhanden ist)", "PV (Nur mit PV Überschuß)"}
		fmt.Println()
		modeChoice, _ := c.askChoice("Was sollte der Standard-Lademodus sein, wenn ein Fahrzeug angeschlossen wird?", ladeModi)
		loadpoint.Mode = chargingModes[modeChoice]

		c.configuration.AddLoadpoint(loadpoint)

		fmt.Println()
		if !c.askYesNo("Möchtest du einen weiteren Ladepunkt hinzufügen") {
			break
		}
	}
}

// ask site specific questions
func (c *CmdConfigure) configureSite() {
	fmt.Println()
	fmt.Println("- Richte deinen Standort ein")

	siteTitle := c.askValue(question{
		label:        "Titel des Standortes",
		defaultValue: defaultTitleSite,
		required:     true})
	c.configuration.SetSiteTitle(siteTitle)
}
