package shelly

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type Generation interface {
	Enabled() (bool, error)
	Enable(bool) error
	api.Meter
	api.MeterEnergy
	api.MeterReturnEnergy
	IsThreePhase() bool
	IsReversed() bool
	HasReturnEnergy() bool
	HasPhases() bool
}

type Phases interface {
	api.PhaseCurrents
	api.PhaseVoltages
	api.PhasePowers
}

// Connection is the Shelly connection
type Connection struct {
	Generation
	gen int
}

// SignedPower reports whether the device returns directional (signed) power.
func (c *Connection) SignedPower() bool {
	return c.gen >= 3
}

// NewConnection creates a new Shelly device connection.
func NewConnection(uri, user, password string, channel int, cache time.Duration) (*Connection, error) {
	log := util.NewLogger("shelly")

	client, uri, resp, err := newClient(log, uri, user, password)
	if err != nil {
		return nil, err
	}

	model, _, _ := strings.Cut(resp.Type+resp.Model, "-")

	var gen Generation
	if resp.Gen < 2 {
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		gen = newGen1(client, uri, model, channel, cache)
	} else {
		// Shelly GEN 2+ API
		// https://shelly-api-docs.shelly.cloud/gen2/
		gen, err = newGen2(client, uri, model, channel, cache)
		if err != nil {
			return nil, err
		}
	}

	conn := &Connection{Generation: gen, gen: resp.Gen}

	return conn, nil
}

// newClient probes the /shelly endpoint and returns an authenticated client, the normalized uri and the device info
func newClient(log *util.Logger, uri, user, password string) (*request.Helper, string, DeviceInfo, error) {
	if uri == "" {
		return nil, "", DeviceInfo{}, errors.New("missing uri")
	}

	for _, suffix := range []string{"/", "/rcp", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}
	uri = util.DefaultScheme(uri, "http")

	client := request.NewHelper(log)

	// Shelly Gen1 and Gen2 families expose the /shelly endpoint
	var resp DeviceInfo
	if err := client.GetJSON(fmt.Sprintf("%s/shelly", uri), &resp); err != nil {
		return nil, "", DeviceInfo{}, err
	}

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return nil, "", DeviceInfo{}, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	client.Transport = request.NewTripper(log, transport.Insecure())

	if user != "" {
		if resp.Gen < 2 {
			log.Redact(transport.BasicAuthHeader(user, password))
			client.Transport = transport.BasicAuth(user, password, client.Transport)
		} else {
			// Shelly gen 2 rfc7616 authentication
			// https://shelly-api-docs.shelly.cloud/gen2/General/Authentication
			client.Transport = transport.Digest(user, password, client.Transport)
		}
	}

	return client, uri, resp, nil
}
