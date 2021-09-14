package cmd

import (
	"bytes"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/evcc-io/config/registry"
	certhelper "github.com/evcc-io/eebus/cert"
	"github.com/evcc-io/eebus/communication"
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

type Loadpoint struct {
	Title   string `yaml:"title,omitempty"`
	Charger string `yaml:"charger,omitempty"`
	Vehicle string `yaml:"vehicle,omitempty"`
}

type Config struct {
	EEBUS      map[string]interface{}   `yaml:"eebus,omitempty"`
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

var configuration Config

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

	loadpointTitle := askValue("Loadpoint title", defaultLoadpointTitle, "")
	loadpoint := Loadpoint{
		Title: loadpointTitle,
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

	siteTitle := askValue("Site title", defaultSiteTitle, "")
	configuration.Site.Title = siteTitle
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

func removeLineWithSubstring(src string, substr []string) string {
	for _, s := range substr {
		re := regexp.MustCompile(".*" + s + ".*[\r\n]*")
		src = re.ReplaceAllString(src, "")
	}
	return src
}

func hasTypeModbus(params []registry.TemplateParam) bool {
	for _, param := range params {
		if param.Type == "modbus" {
			return true
		}
	}
	return false
}

// let the user select a device item from a list defined by class and filter
func processClass(title, class, filter, defaultName string) (test.ConfigTemplate, error) {
	var repeat bool = true
	var deviceConfiguration test.ConfigTemplate

	for ok := true; ok; ok = repeat {
		var localConfiguration Config

		fmt.Println()
		configItem := selectItem(title, class, filter)
		if configItem.Name == itemNotPresent {
			return deviceConfiguration, ErrItemNotPresent
		}

		configItem.PlainSample = strings.TrimRight(configItem.Sample, "\r\n")

		params, deviceName, additionalConfig := processConfig(configItem.Params, defaultName)
		configItem.Params = params

		if len(additionalConfig) > 0 {
			if hasTypeModbus(configItem.Params) {
				// remove all modbus key/value pairs from Sample
				substrings := []string{"id:", "device:", "baudrate:", "comset:", "uri:", "rtu:"}
				configItem.Sample = removeLineWithSubstring(configItem.Sample, substrings)
			}

			// add additional config to Sample
			for key, value := range additionalConfig {
				configItem.Sample += key + ": " + value + "\r\n"
			}
		}

		configItem = renderTemplateSample(configItem)
		var conf map[string]interface{}
		if err := yaml.Unmarshal([]byte(configItem.Sample), &conf); err != nil {
			// silently ignore errors here
			panic("unable to parse sample: %s" + err.Error())
		}
		deviceConfiguration = test.ConfigTemplate{
			Template: configItem,
			Config:   conf,
		}
		deviceConfiguration.Config["name"] = deviceName
		deviceConfiguration.Config["type"] = configItem.Type

		switch class {
		case "charger":
			localConfiguration.Chargers = append(localConfiguration.Chargers, deviceConfiguration.Config)
		case "meter":
			localConfiguration.Meters = append(localConfiguration.Meters, deviceConfiguration.Config)
		case "vehicle":
			localConfiguration.Vehicles = append(localConfiguration.Vehicles, deviceConfiguration.Config)
		default:
			return deviceConfiguration, fmt.Errorf("unknown class: %s", class)
		}

		// check if we need to setup an EEBUS hems
		if class == "charger" && configItem.Type == "eebus" {
			var err error
			err = setupEEBUSConfig()

			if err != nil {
				return deviceConfiguration, fmt.Errorf("error creating EEBUS cert: %s", err)
			}

			localConfiguration.EEBUS = map[string]interface{}{
				"certificate": configuration.EEBUS["certificate"],
			}

			err = configureEEBus(localConfiguration.EEBUS)
			if err != nil {
				return deviceConfiguration, err
			}

			fmt.Println()
			fmt.Println("You have selected an EEBUS wallbox.")
			fmt.Println("Please pair your wallbox with EVCC in the wallbox web interface")
			fmt.Println("When done, press enter to continue.")
			fmt.Scanln()
		}

		yaml, err := yaml.Marshal(localConfiguration)
		if err != nil {
			return deviceConfiguration, err
		}

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

		if err != nil || repeat {
			fmt.Println("Error: ", err)
			fmt.Println()
			if !askYesNo("This device configuration does not work and can not be selected. Do you want to restart the device selection?") {
				fmt.Println()
				return deviceConfiguration, ErrItemNotPresent
			}
		}
	}

	return deviceConfiguration, nil
}

func renderTemplateSample(tmpl registry.Template) registry.Template {
	if len(tmpl.Params) == 0 {
		return tmpl
	}

	sampleTmpl, err := template.New("sample").Parse(tmpl.Sample)
	if err != nil {
		panic(err)
	}

	paramItems := make(map[string]interface{})

	for _, item := range tmpl.Params {
		paramItem := make(map[string]string)

		if item.Name == "" {
			panic("params name is required")
		}
		if item.Value == "" && item.Type == "" {
			panic("params value or type is required")
		}
		if item.Type != "" && len(item.Choice) == 0 {
			panic("params choice is required with type")
		}

		if item.Value != "" {
			paramItem["value"] = item.Value
		}
		if item.Hint != "" {
			paramItem["hint"] = item.Hint
		}
		paramItems[item.Name] = paramItem
	}

	var tpl bytes.Buffer
	if err = sampleTmpl.Execute(&tpl, paramItems); err != nil {
		panic(err)
	}

	tmpl.Sample = tpl.String()

	return tmpl
}

// setup EEBUS certificate
// this id nearly identical to eebus.go
func setupEEBUSConfig() error {
	details := communication.ManufacturerDetails{
		DeviceName:    "EVCC",
		DeviceCode:    "EVCC_HEMS_01",
		DeviceAddress: "EVCC_HEMS",
		BrandName:     "EVCC",
	}

	subject := pkix.Name{
		CommonName:   details.DeviceCode,
		Country:      []string{"DE"},
		Organization: []string{details.BrandName},
	}

	cert, err := certhelper.CreateCertificate(true, subject)
	if err != nil {
		return fmt.Errorf("could not create certificate")
	}

	pubKey, privKey, err := certhelper.GetX509KeyPair(cert)
	if err != nil {
		return fmt.Errorf("could not process generated certificate")
	}

	var certificate = map[string]interface{}{
		"public":  pubKey,
		"private": privKey,
	}
	configuration.EEBUS = map[string]interface{}{
		"certificate": certificate,
	}

	return nil
}

// return EVCC configuration items of a given class
func fetchElements(class, filter string) []registry.Template {
	var items []registry.Template

	for _, tmpl := range registry.TemplatesByClass(class) {
		if len(tmpl.Params) == 0 {
			continue
		}

		if len(filter) == 0 || strings.Contains(tmpl.Name, filter) ||
			(class == "meter" && !strings.Contains(tmpl.Name, "(") && !strings.Contains(tmpl.Name, ")")) {
			items = append(items, tmpl)
		}
	}

	sort.Slice(items[:], func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items
}

// PromptUI: select item from list
func selectItem(title, class, filter string) registry.Template {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ .Name }}",
		Inactive: "   {{ .Name }}",
		Selected: fmt.Sprintf("%s: {{ .Name }}", class),
	}

	var emptyItem registry.Template
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

// PromptUI: select item from list
func askChoice(label string, choices []string) (int, string) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ . }}",
		Inactive: "   {{ . }}",
		Selected: "   {{ . }}",
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     choices,
		Templates: templates,
		Size:      10,
	}

	index, result, err := prompt.Run()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	return index, result
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
func askValue(label, defaultValue, hint string) string {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	validate := func(input string) error {
		return nil
	}
	// var defValue string
	// switch v := defaultValue.(type) {
	// case nil:
	// 	defValue = ""
	// case string:
	// 	defValue = v
	// case int:
	// 	defValue = strconv.Itoa(v)
	// 	validate = func(input string) error {
	// 		_, err := strconv.ParseInt(input, 10, 64)
	// 		return err
	// 	}
	// default:
	// 	log.FATAL.Fatalf("unsupported type: %s", defaultValue)
	// }

	if hint != "" {
		fmt.Println(hint)
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Default:   defaultValue,
		Validate:  validate,
		AllowEdit: true,
	}

	result, err := prompt.Run()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	return result
	// var returnValue interface{}
	// switch defaultValue.(type) {
	// case nil:
	// 	returnValue = result
	// case string:
	// 	returnValue = result
	// case int:
	// 	returnValue, err = strconv.Atoi(result)
	// 	if err != nil {
	// 		log.FATAL.Fatal("entered invalid int value")
	// 	}
	// default:
	// 	log.FATAL.Fatalf("unsupported type: %s", defaultValue)
	// }
	// return returnValue
}

// Process an EVCC configuration item
// Returns
//   processed params and their user values
//   the user entered name of the device
//   a list of additional key/value pairs the need to be added to the configuration
func processConfig(paramItems []registry.TemplateParam, defaultName string) ([]registry.TemplateParam, string, map[string]string) {
	additionalConfig := make(map[string]string)

	fmt.Println("Enter the configuration values:")

	for index, param := range paramItems {
		if param.Type == "modbus" {
			choices := []string{}
			for _, choice := range param.Choice {
				switch choice {
				case "serial":
					choices = append(choices, "Serial (USB-RS485 Adapter)")
				case "tcprtu":
					choices = append(choices, "Serial (Ethernet-RS485 Adapter)")
				case "tcp":
					choices = append(choices, "TCP/IP")
				}
			}

			if len(choices) > 0 {
				// ask for modbus address
				id := askValue("ID", "1", "Modbus ID")
				additionalConfig["id"] = id

				// ask for modbus interface type
				index, _ := askChoice("Select the Modbus interface", choices)
				selectedType := param.Choice[index]
				fmt.Println("Selected Type:", selectedType)
				switch selectedType {
				case "serial":
					device := askValue("Device", "/dev/ttyUSB0", "USB-RS485 Adapter address")
					additionalConfig["device"] = device
					baudrate := askValue("Baudrate", "9600", "")
					additionalConfig["baudrate"] = baudrate
					comset := askValue("ComSet", "8N1", "")
					additionalConfig["comset"] = comset
				case "tcprtu", "tcp":
					if selectedType == "tcprtu" {
						additionalConfig["rtu"] = "true"
					}
					uri := askValue("Host", "192.0.2.2", "IP address or hostname")
					port := askValue("Port", "502", "Port address")
					additionalConfig["uri"] = uri + ":" + port
				}
			}
		} else {
			paramItems[index].Value = askValue(param.Name, param.Value, param.Hint)
		}
	}

	fmt.Println()
	deviceName := askValue("Name", defaultName, "Give the device a name")

	return paramItems, deviceName, additionalConfig
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
