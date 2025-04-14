package shelly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Gen1API endpoint reference: https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview

type Gen1SwitchResponse struct {
	Ison bool
}

type Gen1Status struct {
	Meters []struct {
		Power          float64
		Current        float64
		Voltage        float64
		Total          float64
		Total_Returned float64
	}
	// Shelly EM meter JSON response
	EMeters []struct {
		Power          float64
		Current        float64
		Voltage        float64
		Total          float64
		Total_Returned float64
	}
}

type gen1 struct {
	*request.Helper
	uri     string
	channel int
	model   string
	status  util.Cacheable[Gen1Status]
}

// gen1InitApi initializes the connection to the shelly gen1 api and sets up the cached gen1Status
func (c *gen1) InitApi(uri, user, password string, cache time.Duration) {
	// Cached gen1Status
	c.status = util.ResettableCached(func() (Gen1Status, error) {
		var gen1StatusResponse Gen1Status
		err := c.GetJSON(fmt.Sprintf("%s/status", uri), &gen1StatusResponse)
		if err != nil {
			return Gen1Status{}, err
		}
		return gen1StatusResponse, nil
	}, cache)
}

func (c *gen1) CurrentPower() (float64, error) {
	var power float64
	res, err := c.status.Get()
	if err != nil {
		return 0, err
	}
	switch {
	case c.channel < len(res.Meters):
		power = res.Meters[c.channel].Power
	case c.channel < len(res.EMeters):
		power = res.EMeters[c.channel].Power
	default:
		return 0, errors.New("invalid channel, missing power meter")
	}
	return power, nil
}

func (c *gen1) Enabled() (bool, error) {
	var res Gen1SwitchResponse
	uri := fmt.Sprintf("%s/relay/%d", c.uri, c.channel)
	err := c.GetJSON(uri, &res)
	return res.Ison, err
}

func (c *gen1) Enable(enable bool) error {
	var err error
	onoff := map[bool]string{true: "on", false: "off"}

	var res Gen1SwitchResponse
	uri := fmt.Sprintf("%s/relay/%d?turn=%s", c.uri, c.channel, onoff[enable])
	err = c.GetJSON(uri, &res)

	return err
}

func (c *gen1) TotalEnergy() (float64, error) {
	var energy float64
	res, err := c.status.Get()
	if err != nil {
		return 0, err
	}
	switch {
	case c.channel < len(res.Meters):
		energy = res.Meters[c.channel].Total - res.Meters[c.channel].Total_Returned
	case c.channel < len(res.EMeters):
		energy = res.EMeters[c.channel].Total - res.EMeters[c.channel].Total_Returned
	default:
		return 0, errors.New("invalid channel, missing power meter")
	}

	energy = gen1Energy(c.model, energy)

	return energy / 1000, nil
}

// gen1Energy in kWh
func gen1Energy(devicetype string, energy float64) float64 {
	// Gen 1 Shelly EM devices are providing Watt hours, Gen 1 Shelly PM devices are providing Watt minutes
	if !strings.Contains(devicetype, "EM") {
		energy /= 60
	}
	return energy
}
