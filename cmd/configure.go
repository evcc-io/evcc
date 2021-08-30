package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/test"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const defaultChargerName string = "wallbox"
const defaultGridMeterName string = "grid"
const defaultPVInverterMeter string = "PV"
const defaultHomeBatteryMeter string = "Battery"
const defaultEVTitle string = "EV"
const defaultLoadpointTitle = "Garage"
const defaultSiteTitle = "My Home"

const itemNotPresent string = "My item is not in this list"

var ErrItemNotPresent = errors.New("item not present")

var uriMapping = map[string]string{
	"192.0.2.2": "",
	"192.0.2.3": "",
	"192.0.2.4": "",
	"192.0.2.5": "",
}

type Loadpoint struct {
	Title   string `yaml:"title,omitempty"`
	Charger string `yaml:"charger,omitempty"`
	Vehicle string `yaml:"vehicle,omitempty"`
}

type Config struct {
	Chargers   []map[string]interface{} `yaml:"chargers,omitempty"`
	Meters     []map[string]interface{} `yaml:"meters,omitempty"`
	Vehicles   []map[string]interface{} `yaml:"vehicles,omitempty"`
	Loadpoints []Loadpoint              `yaml:"loadpoints,omitempty"`
	Site       struct {
		Title  string `yaml:"title,omitempty"`
		Meters struct {
			Grid    string `yaml:"grid,omitempty"`
			Pv      string `yaml:"pv,omitempty"`
			Battery string `yaml:"battery,omitempty"`
		} `yaml:"meters,omitempty"`
	} `yaml:"site,omitempty"`
}

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Create an EVCC configuration",
	Run:   runConfigure,
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

func runConfigure(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	fmt.Println()
	fmt.Println("The next steps will guide throught the creation of a EVCC configuration file.")
	fmt.Println("Please be aware that this process does not cover all possible scenarios.")
	fmt.Println("You can stop the process by pressing ctrl-c.")
	fmt.Println()
	fmt.Println("Let's start:")

	var configuration Config
	var err error

	fmt.Println()
	fmt.Println("- Configure your wallbox")
	chargerItem, err := processClass("wallbox", "charger", "", defaultChargerName)
	if err != nil && err != ErrItemNotPresent {
		log.FATAL.Fatal(err)
	}
	if err != ErrItemNotPresent {
		configuration.Chargers = append(configuration.Chargers, chargerItem.Config)
	}

	fmt.Println()
	fmt.Println("- Configure your grid meter")

	gridItem, err := processClass("grid meter", "meter", "Grid Meter", defaultGridMeterName)
	if err != nil && err != ErrItemNotPresent {
		log.FATAL.Fatal(err)
	}
	if err != ErrItemNotPresent {
		configuration.Meters = append(configuration.Meters, gridItem.Config)
	}

	fmt.Println()
	fmt.Println("- Configure your PV inverter or PV meter")

	pvItem, err := processClass("pv meter", "meter", "PV Meter", defaultPVInverterMeter)
	if err != nil && err != ErrItemNotPresent {
		log.FATAL.Fatal(err)
	}
	if err != ErrItemNotPresent {
		configuration.Meters = append(configuration.Meters, pvItem.Config)
	}

	var batteryItem test.ConfigTemplate
	fmt.Println()
	if askYesNo("Do you have a home battery system?") {
		fmt.Println("- Configure your Battery inverter or Battery meter")

		batteryItem, err = processClass("battery meter", "meter", "Battery Meter", defaultHomeBatteryMeter)
		if err != nil && err != ErrItemNotPresent {
			log.FATAL.Fatal(err)
		}
		if err != ErrItemNotPresent {
			configuration.Meters = append(configuration.Meters, batteryItem.Config)
		}
	}

	fmt.Println()
	fmt.Println("- Configure your vehicle")

	vehicleItem, err := processClass("vehicle", "vehicle", "", defaultEVTitle)
	if err != nil && err != ErrItemNotPresent {
		log.FATAL.Fatal(err)
	}
	if err != ErrItemNotPresent {
		configuration.Vehicles = append(configuration.Vehicles, vehicleItem.Config)
	}

	fmt.Println()
	fmt.Println("- Configure your loadpoints")

	loadpointTitle := askValue("Loadpoint title", defaultLoadpointTitle)
	loadpoint := Loadpoint{
		Title: loadpointTitle.(string),
	}
	if chargerItem.Config["name"] != nil {
		loadpoint.Charger = chargerItem.Config["name"].(string)
	}
	if vehicleItem.Config["name"] != nil {
		loadpoint.Vehicle = vehicleItem.Config["name"].(string)
	}
	configuration.Loadpoints = append(configuration.Loadpoints, loadpoint)

	fmt.Println()
	fmt.Println("- Configure your site")

	siteTitle := askValue("Site title", defaultSiteTitle)
	configuration.Site.Title = siteTitle.(string)
	if gridItem.Config["name"] != nil {
		configuration.Site.Meters.Grid = gridItem.Config["name"].(string)
	}
	if pvItem.Config["name"] != nil {
		configuration.Site.Meters.Pv = pvItem.Config["name"].(string)
	}
	if batteryItem.Config["name"] != nil {
		configuration.Site.Meters.Battery = batteryItem.Config["name"].(string)
	}

	yaml, err := yaml.Marshal(configuration)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Your configuration:")
	fmt.Println()
	fmt.Println(string(yaml[:]))
}

// let the user select a device item from a list defined by class and filter
func processClass(title, class, filter, defaultName string) (test.ConfigTemplate, error) {
	var repeat bool = true
	var classConfiguration test.ConfigTemplate

	for ok := true; ok; ok = repeat {
		var configuration Config

		fmt.Println()
		configItem := selectItem(title, class, filter)
		if configItem.Name == itemNotPresent {
			return classConfiguration, ErrItemNotPresent
		}

		classConfiguration = processConfig(configItem, defaultName)

		switch class {
		case "charger":
			configuration.Chargers = append(configuration.Chargers, classConfiguration.Config)
		case "meter":
			configuration.Meters = append(configuration.Meters, classConfiguration.Config)
		case "vehicle":
			configuration.Vehicles = append(configuration.Vehicles, classConfiguration.Config)
		default:
			return classConfiguration, fmt.Errorf("unknown class: %s", class)
		}

		yaml, err := yaml.Marshal(configuration)
		if err != nil {
			return classConfiguration, err
		}

		// check if we need to setup an EEBUS hems
		// if class == "charger" {

		// }

		fmt.Println()
		fmt.Println("Testing configuration...")
		fmt.Println()

		err = testDeviceConfig(class, yaml)
		if err == nil {
			// Do we see proper values?
			fmt.Println()
			if askYesNo("Does the test data above show proper values?") {
				repeat = false
			}
		}

		if err != nil || repeat == true {
			fmt.Println()
			if !askYesNo("This device configuration does not work and can not be selected. Do you want to restart the device selection?") {
				fmt.Println()
				return classConfiguration, ErrItemNotPresent
			}
		}
	}

	return classConfiguration, nil
}

// return EVCC configuration items of a given class
func fetchElements(class, filter string) []test.ConfigTemplate {
	var items []test.ConfigTemplate

	for _, tmpl := range test.ConfigTemplates(class) {
		if len(filter) == 0 || strings.Contains(tmpl.Name, filter) {
			items = append(items, tmpl)
		}
	}

	sort.Slice(items[:], func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items
}

// PromptUI: select item from list
func selectItem(title, class, filter string) test.ConfigTemplate {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ .Name }}",
		Inactive: "   {{ .Name }}",
		Selected: fmt.Sprintf("%s: {{ .Name }}", class),
	}

	var emptyItem test.ConfigTemplate
	emptyItem.Name = itemNotPresent

	items := fetchElements(class, filter)
	items = append(items, emptyItem)

	prompt := promptui.Select{
		Label:     fmt.Sprintf("Select your %s", title),
		Items:     items,
		Templates: templates,
		Size:      10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	return items[index]
}

// PromptUI: ask yes/no question, return true if yes is selected
func askYesNo(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()

	return !errors.Is(err, promptui.ErrAbort)
}

// PromputUI: ask for input
func askValue(label string, defaultValue interface{}) interface{} {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	validate := func(input string) error {
		return nil
	}
	var defValue string
	switch v := defaultValue.(type) {
	case nil:
		defValue = ""
	case string:
		defValue = v
	case int:
		defValue = strconv.Itoa(v)
		validate = func(input string) error {
			_, err := strconv.ParseInt(input, 10, 64)
			return err
		}
	default:
		log.FATAL.Fatalf("unsupported type: %s", defaultValue)
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Default:   defValue,
		Validate:  validate,
		AllowEdit: true,
	}

	result, err := prompt.Run()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	var returnValue interface{}
	switch defaultValue.(type) {
	case nil:
		returnValue = result
	case string:
		returnValue = result
	case int:
		returnValue, err = strconv.Atoi(result)
		if err != nil {
			log.FATAL.Fatal("entered invalid int value")
		}
	default:
		log.FATAL.Fatalf("unsupported type: %s", defaultValue)
	}
	return returnValue
}

// Process an EVCC configuration item
func processConfig(configItem test.ConfigTemplate, defaultName string) test.ConfigTemplate {
	// check for parameters the user has to provide
	var conf map[string]interface{}
	if err := yaml.Unmarshal([]byte(configItem.Sample), &conf); err != nil {
		// silently ignore errors here
		log.WARN.Printf("unable to parse sample: %s", err)
	}

	parsed := test.ConfigTemplate{
		Template: configItem.Template,
		Config:   conf,
	}

	if len(conf) > 0 {
		fmt.Println()
		fmt.Println("Enter the configuration values:")
		parsed.Config = processConfigLevel(parsed.Config)
	}

	parsed.Config["type"] = configItem.Template.Type

	fmt.Println()
	fmt.Println("Provide a name for this device:")
	parsed.Config["name"] = askValue("Name", defaultName).(string)

	return parsed
}

func processConfigLevel(config map[string]interface{}) map[string]interface{} {
	if len(config) > 0 {
		for param, value := range config {
			var prompt string
			valueType := reflect.ValueOf(value)

			switch param {
			case "user":
				prompt = "Username"
			case "password":
				prompt = "Password"
			case "meter":
				// e.g. Discovery meter
				prompt = "Identifier"
			case "device":
				// Serial modbus devices
				prompt = "Serial port"
			case "baudrate":
				// Serial modbus devices
				prompt = "Serial baudrate"
			case "ain":
				// Fritzbox Dect devices
				prompt = "AIN (printed on the device)"
			case "token":
				// Tokens, e.g. go-e Charger
				prompt = "Token"
			case "mac":
				// MAC Address, e.g. NRGKick Charger
				prompt = "MAC address"
			case "pin":
				// PIN Number, e.g. NRGKick Charger
				prompt = "PIN number"
			case "charger":
				// Charger Identifier, e.g. Easee
				prompt = "Charger identifier"
			case "title":
				// device title, e.g. vehicle title
				prompt = "Title"
			case "capacity":
				// device capacitiy, e.g. vehicle battery
				prompt = "Battery capacity"
			case "vin":
				// Vehicle VIN
				prompt = "Vehicle VIN"
			default:
				if valueType.Kind() == reflect.Map {
					value = processConfigLevel(value.(map[string]interface{}))
				} else if valueType.Kind() == reflect.Slice {
					sliceValue := value.([]interface{})
					for i := range sliceValue {
						value = processConfigLevel(sliceValue[i].(map[string]interface{}))
					}
				} else if valueType.Kind() == reflect.String {
					for k := range uriMapping {
						if strings.Contains(value.(string), k) {
							if len(uriMapping[k]) == 0 {
								uriMapping[k] = askValue("Address:", k).(string)
							}
							value = strings.Replace(value.(string), k, uriMapping[k], -1)
						}
					}
				}
			}

			if len(prompt) > 0 {
				value = askValue(prompt, value)
			}

			config[param] = value
		}
	}

	return config
}

// return a usable EVCC configuration
func readConfiguration(configuration []byte) (conf config, err error) {
	if err := viper.ReadConfig(bytes.NewBuffer(configuration)); err != nil {
		return conf, err
	}

	if err := viper.UnmarshalExact(&conf); err != nil {
		return conf, err
	}

	return conf, nil
}

// test a device configuration
func testDeviceConfig(class string, configuration []byte) error {
	conf, err := readConfiguration(configuration)
	if err != nil {
		return err
	}

	switch class {
	case "charger":
		return testChargerConfig(conf)
	case "meter":
		return testMeterConfig(conf)
	case "vehicle":
		return testVehicleConfig(conf)
	}

	return fmt.Errorf("invalid class %s provided", class)
}

// test a charger configuration
// almost identical to charger.go implementation
func testChargerConfig(conf config) error {
	if err := cp.configureChargers(conf); err != nil {
		return err
	}

	d := dumper{len: 1}
	for name, v := range cp.chargers {
		d.DumpWithHeader(name, v)
	}

	return nil
}

// test a meter configuration
// almost identical to meter.go implementation
func testMeterConfig(conf config) error {
	if err := cp.configureMeters(conf); err != nil {
		return err
	}

	d := dumper{len: 1}
	for name, v := range cp.meters {
		d.DumpWithHeader(name, v)
	}

	return nil
}

// test a meter configuration
// almost identical to vehicle.go implementation
func testVehicleConfig(conf config) error {
	if err := cp.configureVehicles(conf); err != nil {
		return err
	}

	d := dumper{len: 1}
NEXT:
	for name, v := range cp.vehicles {
		start := time.Now()

	WAIT:
		// wait up to 1m for the vehicle to wakeup
		for {
			if time.Since(start) > time.Minute {
				log.ERROR.Println(api.ErrTimeout)
				continue NEXT
			}

			if _, err := v.SoC(); err != nil {
				if errors.Is(err, api.ErrMustRetry) {
					time.Sleep(5 * time.Second)
					fmt.Print(".")
					continue WAIT
				}

				log.ERROR.Println(err)
				continue NEXT
			}

			break
		}

		d.DumpWithHeader(name, v)
	}

	return nil
}
