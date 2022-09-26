package configure

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/util/templates"
)

type device struct {
	Name            string
	Title           string
	Yaml            string
	ChargerHasMeter bool // only used with chargers to detect if we need to ask for a charge meter
}

type loadpoint struct {
	Title             string // TODO Perspektivisch können wir was aus core wiederverwenden, für später
	Charger           string
	ChargeMeter       string
	Vehicles          []string
	Mode              string
	MinCurrent        int
	MaxCurrent        int
	Phases            int
	ResetOnDisconnect string
}

type config struct {
	Meters     []device
	Chargers   []device
	Vehicles   []device
	Loadpoints []loadpoint
	Site       struct { // TODO Perspektivisch können wir was aus core wiederverwenden, für später
		Title     string
		Grid      string
		PVs       []string
		Batteries []string
	}
	Hems         string
	EEBUS        string
	MQTT         string
	SponsorToken string
}

type Configure struct {
	config config
}

// AddDevice adds a device reference of a specific category to the configuration
// e.g. a PV meter to site.PVs
func (c *Configure) AddDevice(d device, category DeviceCategory) {
	switch DeviceCategories[category].class {
	case templates.Charger:
		c.config.Chargers = append(c.config.Chargers, d)

	case templates.Meter:
		c.config.Meters = append(c.config.Meters, d)
		switch DeviceCategories[category].categoryFilter {
		case DeviceCategoryGridMeter:
			c.config.Site.Grid = d.Name
		case DeviceCategoryPVMeter:
			c.config.Site.PVs = append(c.config.Site.PVs, d.Name)
		case DeviceCategoryBatteryMeter:
			c.config.Site.Batteries = append(c.config.Site.Batteries, d.Name)
		}

	case templates.Vehicle:
		c.config.Vehicles = append(c.config.Vehicles, d)

	default:
		panic("invalid class for category: " + category)
	}
}

// DevicesOfClass returns all configured devices of a given DeviceClass
func (c *Configure) DevicesOfClass(class templates.Class) []device {
	switch class {
	case templates.Charger:
		return c.config.Chargers
	case templates.Meter:
		return c.config.Meters
	case templates.Vehicle:
		return c.config.Vehicles
	default:
		panic("invalid class: " + class)
	}
}

// AddLoadpoint adds a loadpoint to the configuration
func (c *Configure) AddLoadpoint(l loadpoint) {
	c.config.Loadpoints = append(c.config.Loadpoints, l)
}

// MetersOfCategory returns the number of configured meters of a given DeviceCategory
func (c *Configure) MetersOfCategory(category DeviceCategory) int {
	switch category {
	case DeviceCategoryGridMeter:
		if c.config.Site.Grid != "" {
			return 1
		}
	case DeviceCategoryPVMeter:
		return len(c.config.Site.PVs)
	case DeviceCategoryBatteryMeter:
		return len(c.config.Site.Batteries)
	}

	return 0
}

//go:embed configure.tpl
var configTmpl string

// RenderConfiguration creates a yaml configuration
func (c *Configure) RenderConfiguration() ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(configTmpl)
	if err != nil {
		panic(err)
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, c.config)

	return bytes.TrimSpace(out.Bytes()), err
}
