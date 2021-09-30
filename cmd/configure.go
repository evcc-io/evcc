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

// var configuration Config
type CmdConfigure struct {
	configuration Config
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
	impl := &CmdConfigure{}
	impl.Run()
}

func (c *CmdConfigure) Run() {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	fmt.Println()
	fmt.Println("The next steps will guide throught the creation of a EVCC configuration file.")
	fmt.Println("Please be aware that this process does not cover all possible scenarios.")
	fmt.Println("You can stop the process by pressing ctrl-c.")
	fmt.Println()
	fmt.Println("Let's start:")

	var chargerItem, gridItem, pvItem, batteryItem test.ConfigTemplate
	var err error

	// chargerItem = c.configureClass("wallbox", "charger", "", defaultChargerName)
	// gridItem = c.configureClass("grid meter", "meter", registry.UsageChoiceGrid, defaultGridMeterName)
	pvItem = c.configureClass("pv meter", "meter", registry.UsageChoicePV, defaultPVInverterMeter)

	fmt.Println()
	if c.askYesNo("Do you have a home battery") {
		batteryItem = c.configureClass("battery meter", "meter", registry.UsageChoiceBattery, defaultHomeBatteryMeter)
	}

	vehicleItem := c.configureClass("vehicle", "vehicle", "", defaultEVTitle)

	fmt.Println()
	fmt.Println("- Configure your loadpoints")

	loadpointTitle := c.askValue("Loadpoint title", defaultLoadpointTitle, "", false)
	loadpoint := Loadpoint{
		Title: loadpointTitle,
	}
	if chargerItem.Config["name"] != nil {
		loadpoint.Charger = chargerItem.Config["name"].(string)
	}
	if vehicleItem.Config["name"] != nil {
		loadpoint.Vehicle = vehicleItem.Config["name"].(string)
	}
	c.configuration.Loadpoints = append(c.configuration.Loadpoints, loadpoint)

	fmt.Println()
	fmt.Println("- Configure your site")

	siteTitle := c.askValue("Site title", defaultSiteTitle, "", false)
	c.configuration.Site.Title = siteTitle
	if gridItem.Config["name"] != nil {
		c.configuration.Site.Meters.Grid = gridItem.Config["name"].(string)
	}
	if pvItem.Config["name"] != nil {
		c.configuration.Site.Meters.Pv = pvItem.Config["name"].(string)
	}
	if batteryItem.Config["name"] != nil {
		c.configuration.Site.Meters.Battery = batteryItem.Config["name"].(string)
	}

	yaml, err := yaml.Marshal(c.configuration)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Your configuration:")
	fmt.Println()
	fmt.Println(string(yaml[:]))
}

func (c *CmdConfigure) configureClass(title, class, usageFilter, defaultName string) test.ConfigTemplate {
	fmt.Println()
	fmt.Printf("- Configure your %s\n", title)

	classItem, err := c.processClass(title, class, usageFilter, defaultName)
	if err != nil && err != ErrItemNotPresent {
		log.FATAL.Fatal(err)
	}
	if err != ErrItemNotPresent {
		switch class {
		case "charger":
			c.configuration.Chargers = append(c.configuration.Chargers, classItem.Config)
		case "meter":
			// shorten the configuration if this is a device with a public entry
			if classItem.Public != "" && len(classItem.Params) > 0 {
				for _, param := range classItem.Params {
					if param.Name == registry.ParamNameValueModbus {
						continue
					}
					classItem.Config[param.Name] = param.Value
				}
				classItem.Config["public"] = classItem.Public
				delete(classItem.Config, "power")
			}
			c.configuration.Meters = append(c.configuration.Meters, classItem.Config)
		case "vehicle":
			c.configuration.Vehicles = append(c.configuration.Vehicles, classItem.Config)
		}
	}

	return classItem
}

// let the user select a device item from a list defined by class and filter
func (c *CmdConfigure) processClass(title, class, usageFilter, defaultName string) (test.ConfigTemplate, error) {
	var repeat bool = true
	var deviceConfiguration test.ConfigTemplate

	for ok := true; ok; ok = repeat {
		var localConfiguration Config

		fmt.Println()
		configItem := c.selectItem(title, class, usageFilter)
		if configItem.Name == itemNotPresent {
			return deviceConfiguration, ErrItemNotPresent
		}

		configItem.PlainSample = strings.TrimRight(configItem.Sample, "\r\n")

		params, deviceName, additionalConfig := c.processConfig(configItem.Params, defaultName, usageFilter)
		configItem.Params = params

		// patch the configuration sample text with modbus configuration data
		if len(additionalConfig) > 0 {
			if c.paramsHasTypeModbus(configItem.Params) {
				// remove all modbus key/value pairs from Sample
				substrings := []string{
					registry.ModbusParamNameId + ":",
					registry.ModbusParamNameDevice + ":",
					registry.ModbusParamNameBaudrate + ":",
					registry.ModbusParamNameComset + ":",
					registry.ModbusParamNameURI + ":",
					registry.ModbusParamNameRTU + ":",
				}
				configItem.Sample = c.removeLineWithSubstring(configItem.Sample, substrings)
			}

			// add additional config to Sample
			for key, value := range additionalConfig {
				configItem.Sample += key + ": " + value + "\r\n"
			}
		}
		configItem = c.renderTemplateSample(configItem, usageFilter)

		// create the configuration data structure
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
			err = c.setupEEBUSConfig()

			if err != nil {
				return deviceConfiguration, fmt.Errorf("error creating EEBUS cert: %s", err)
			}

			localConfiguration.EEBUS = map[string]interface{}{
				"certificate": c.configuration.EEBUS["certificate"],
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

		err = c.testDeviceConfig(class, yaml)
		if err == nil {
			// Do we see proper values?
			fmt.Println()
			if c.askYesNo("Does the test data above show proper values?") {
				repeat = false
			}
		}

		if err != nil || repeat {
			fmt.Println("Error: ", err)
			fmt.Println()
			if !c.askYesNo("This device configuration does not work and can not be selected. Do you want to restart the device selection?") {
				fmt.Println()
				return deviceConfiguration, ErrItemNotPresent
			}
		}
	}

	return deviceConfiguration, nil
}

func (c *CmdConfigure) removeLineWithSubstring(src string, substr []string) string {
	for _, s := range substr {
		re := regexp.MustCompile(".*" + s + ".*[\r\n]*")
		src = re.ReplaceAllString(src, "")
	}
	return src
}

func (c *CmdConfigure) paramsHasTypeModbus(params []registry.TemplateParam) bool {
	for _, param := range params {
		if param.Name == "modbus" {
			return true
		}
	}
	return false
}

func (c *CmdConfigure) renderTemplateSample(tmpl registry.Template, usageFilter string) registry.Template {
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
		if item.Value == "" && !item.Optional && item.Choice == nil {
			panic("params value or choice is required")
		}

		if item.Name == registry.ParamNameValueUsage {
			if len(item.Choice) == 0 {
				panic("params choice is required with usage")
			}

			if usageFilter != "" {
				for _, usage := range item.Choice {
					if usage == usageFilter {
						paramItem[usage] = "true"
					}
				}
			}
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
func (c *CmdConfigure) setupEEBUSConfig() error {
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
	c.configuration.EEBUS = map[string]interface{}{
		"certificate": certificate,
	}

	return nil
}

// return EVCC configuration items of a given class
func (c *CmdConfigure) fetchElements(class, usageFilter string) []registry.Template {
	var items []registry.Template

	for _, tmpl := range registry.TemplatesByClass(class) {
		if len(tmpl.Params) == 0 {
			continue
		}

		if len(usageFilter) == 0 ||
			c.paramChoiceContains(tmpl.Params, registry.ParamNameValueUsage, usageFilter, true) {
			items = append(items, tmpl)
		}
	}

	sort.Slice(items[:], func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items
}

func (c *CmdConfigure) paramChoiceContains(params []registry.TemplateParam, name, filter string, considerEmptyAsTrue bool) bool {
	filterFound := false
	for _, item := range params {
		if item.Name != name {
			continue
		}

		filterFound = true
		if item.Choice == nil || len(item.Choice) == 0 {
			return true
		}

		for _, choice := range item.Choice {
			if choice == filter {
				return true
			}
		}
	}

	if !filterFound && considerEmptyAsTrue {
		return true
	}

	return false
}

// PromptUI: select item from list
func (c *CmdConfigure) selectItem(title, class, usageFilter string) registry.Template {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ .Name }}",
		Inactive: "   {{ .Name }}",
		Selected: fmt.Sprintf("%s: {{ .Name }}", class),
	}

	var emptyItem registry.Template
	emptyItem.Name = itemNotPresent

	items := c.fetchElements(class, usageFilter)
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
func (c *CmdConfigure) askChoice(label string, choices []string) (int, string) {
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
func (c *CmdConfigure) askYesNo(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()

	return !errors.Is(err, promptui.ErrAbort)
}

// PromputUI: ask for input
func (c *CmdConfigure) askValue(label, defaultValue, hint string, optional bool) string {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	validate := func(input string) error {
		return nil
	}

	if hint != "" {
		fmt.Println(hint)
	}
	if optional {
		fmt.Println("(optional, can be ignored)")
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
}

// Process an EVCC configuration item
// Returns
//   processed params and their user values
//   the user entered name of the device
//   a list of additional key/value pairs the need to be added to the configuration
func (c *CmdConfigure) processConfig(paramItems []registry.TemplateParam, defaultName, usageFilter string) ([]registry.TemplateParam, string, map[string]string) {
	additionalConfig := make(map[string]string)

	fmt.Println("Enter the configuration values:")

	for index, param := range paramItems {
		if param.Name == "modbus" {
			choices := []string{}
			choiceKeys := []string{}
			for _, choice := range param.Choice {
				switch choice {
				case registry.ModbusChoiceRS485:
					choices = append(choices, "Serial (USB-RS485 Adapter)")
					choiceKeys = append(choiceKeys, registry.ModbusKeyRS485Serial)
					choices = append(choices, "Serial (Ethernet-RS485 Adapter)")
					choiceKeys = append(choiceKeys, registry.ModbusKeyRS485TCPIP)
				case registry.ModbusChoiceTCPIP:
					choices = append(choices, "TCP/IP")
					choiceKeys = append(choiceKeys, registry.ModbusKeyTCPIP)
				}
			}

			if len(choices) > 0 {
				// ask for modbus address
				id := c.askValue("ID", "1", "Modbus ID", false)
				additionalConfig[registry.ModbusParamNameId] = id

				// ask for modbus interface type
				index, _ := c.askChoice("Select the Modbus interface", choices)
				selectedType := choiceKeys[index]
				switch selectedType {
				case registry.ModbusKeyRS485Serial:
					device := c.askValue("Device", registry.ModbusParamValueDevice, "USB-RS485 Adapter address", false)
					additionalConfig[registry.ModbusParamNameDevice] = device
					baudrate := c.askValue("Baudrate", registry.ModbusParamValueBaudrate, "", false)
					additionalConfig[registry.ModbusParamNameBaudrate] = baudrate
					comset := c.askValue("ComSet", registry.ModbusParamValueComset, "", false)
					additionalConfig[registry.ModbusParamNameComset] = comset
				case registry.ModbusKeyRS485TCPIP, registry.ModbusKeyTCPIP:
					if selectedType == registry.ModbusKeyRS485TCPIP {
						additionalConfig[registry.ModbusParamNameRTU] = "true"
					}
					uri := c.askValue("Host", registry.ModbusParamValueURI, "IP address or hostname", false)
					port := c.askValue("Port", registry.ModbusParamValuePort, "Port address", false)
					additionalConfig[registry.ModbusParamNameURI] = uri + ":" + port
				}
			}
		} else if param.Name != registry.ParamNameValueUsage {
			value := c.askValue(param.Name, param.Value, param.Hint, param.Optional)
			// if value is optional and the user retunred the default value, skip this parameter
			if !param.Optional || value != param.Value {
				paramItems[index].Value = value
			}
		} else if param.Name == registry.ParamNameValueUsage {
			if usageFilter != "" {
				paramItems[index].Value = usageFilter
			}
		}
	}

	fmt.Println()
	deviceName := c.askValue("Name", defaultName, "Give the device a name", false)

	return paramItems, deviceName, additionalConfig
}

// return a usable EVCC configuration
func (c *CmdConfigure) readConfiguration(configuration []byte) (conf config, err error) {
	if err := viper.ReadConfig(bytes.NewBuffer(configuration)); err != nil {
		return conf, err
	}

	if err := viper.UnmarshalExact(&conf); err != nil {
		return conf, err
	}

	return conf, nil
}

// test a device configuration
func (c *CmdConfigure) testDeviceConfig(class string, configuration []byte) error {
	conf, err := c.readConfiguration(configuration)
	if err != nil {
		return err
	}

	switch class {
	case "charger":
		return c.testChargerConfig(conf)
	case "meter":
		return c.testMeterConfig(conf)
	case "vehicle":
		return c.testVehicleConfig(conf)
	}

	return fmt.Errorf("invalid class %s provided", class)
}

// test a charger configuration
// almost identical to charger.go implementation
func (c *CmdConfigure) testChargerConfig(conf config) error {
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
func (c *CmdConfigure) testMeterConfig(conf config) error {
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
func (c *CmdConfigure) testVehicleConfig(conf config) error {
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
