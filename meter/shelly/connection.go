package shelly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type Generation interface {
	CurrentPower() (float64, error)
	Enabled() (bool, error)
	Enable(bool) error
	TotalEnergy() (float64, error)
}

type Phases interface {
	Currents() (float64, float64, float64, error)
	Voltages() (float64, float64, float64, error)
	Powers() (float64, float64, float64, error)
}

// Connection is the Shelly connection
type Connection struct {
	Generation
}

// NewConnection creates a new Shelly device connection.
func NewConnection(uri, user, password string, channel int, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	for _, suffix := range []string{"/", "/rcp", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}
	uri = util.DefaultScheme(uri, "http")

	log := util.NewLogger("shelly")
	client := request.NewHelper(log)

	// Shelly Gen1 and Gen2 families expose the /shelly endpoint
	var resp DeviceInfo
	if err := client.GetJSON(fmt.Sprintf("%s/shelly", uri), &resp); err != nil {
		return nil, err
	}

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return nil, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	model := strings.Split(resp.Type+resp.Model, "-")[0]

	client.Transport = request.NewTripper(log, transport.Insecure())

	var gen Generation
	if resp.Gen < 2 {
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		if user != "" {
			log.Redact(transport.BasicAuthHeader(user, password))
		}
		gen = newGen1(client, uri, model, channel, user, password, cache)
	} else {
		// Shelly GEN 2+ API
		// https://shelly-api-docs.shelly.cloud/gen2/

		var err error
		gen, err = newGen2(client, uri, model, channel, user, password, cache)
		if err != nil {
			return nil, err
		}
	}

	conn := &Connection{gen}

	return conn, nil
}
