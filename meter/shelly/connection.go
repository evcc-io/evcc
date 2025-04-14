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

type Gen interface {
	InitApi(string, string, string, time.Duration)
	CurrentPower() (float64, error)
	Enabled() (bool, error)
	Enable(bool) error
	TotalEnergy() (float64, error)
}

// Connection is the Shelly connection
type Connection struct {
	log     util.Logger
	model   string // Shelly device type
	profile string // Shelly device profile
	Cache   time.Duration
	Gen
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

	// Set default profile to "monophase" if not provided
	if resp.Profile == "" {
		resp.Profile = "monophase"
	}

	client.Transport = request.NewTripper(log, transport.Insecure())

	var gen Gen
	if resp.Gen < 2 {
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		if user != "" {
			log.Redact(transport.BasicAuthHeader(user, password))
			client.Transport = transport.BasicAuth(user, password, client.Transport)
		}
		gen = &gen1{
			Helper:  client,
			uri:     uri,
			channel: channel,
			model:   resp.Model,
			status:  util.NewCacheable[Gen1Status](),
		}
	}
	if resp.Gen > 1 {
		// Shelly GEN 2+ API
		// https://shelly-api-docs.shelly.cloud/gen2/
		gen = &gen2{
			Helper:   client,
			uri:      uri,
			channel:  channel,
			model:    resp.Model,
			profile:  resp.Profile,
			status:   util.NewCacheable[Gen2StatusResponse](),
			emstatus: util.NewCacheable[Gen2EMStatus](),
			emdata:   util.NewCacheable[Gen2EMData](),
		}
	}
	conn := &Connection{
		model:   strings.Split(resp.Type+resp.Model, "-")[0],
		profile: resp.Profile,
		Cache:   cache,
		Gen:     gen,
	}
	conn.Gen.InitApi(uri, user, password, cache)
	return conn, nil
}
