package shelly

import (
	"fmt"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/transport"
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

// gen1InitApi initializes the connection to the shelly gen1 api and sets up the cached gen1Status
func (c *Connection) gen1InitApi(uri, user, password string) {
	// Shelly GEN 1 API
	// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
	c.uri = util.DefaultScheme(uri, "http")
	if user != "" {
		c.log.Redact(transport.BasicAuthHeader(user, password))
		c.Client.Transport = transport.BasicAuth(user, password, c.Client.Transport)
	}
	// Cached gen1Status
	c.gen1Status = util.ResettableCached(func() (Gen1Status, error) {
		var gen1StatusResponse Gen1Status
		err := c.GetJSON(fmt.Sprintf("%s/status", c.uri), &gen1StatusResponse)
		if err != nil {
			return Gen1Status{}, err
		}
		return gen1StatusResponse, nil
	}, c.Cache)
}
