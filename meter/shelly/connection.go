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

// Connection is the Shelly connection
type Connection struct {
	*request.Helper
	uri              string
	channel          int
	Gen              int    // Shelly api generation
	model            string // Shelly device model
	app              string // Shelly device app code
	profile          string // Shelly device profile
	Cache            time.Duration
	gen2SwitchStatus util.Cacheable[Gen2SwitchStatus]
	gen2EM1Status    util.Cacheable[Gen2EM1Status]
	gen2EMStatus     util.Cacheable[Gen2EMStatus]
}

// NewConnection creates a new Shelly device connection.
func NewConnection(uri, user, password string, channel int, cache time.Duration) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	for _, suffix := range []string{"/", "/rcp", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}

	log := util.NewLogger("shelly")
	client := request.NewHelper(log)

	// Shelly Gen1 and Gen2 families expose the /shelly endpoint
	var resp DeviceInfo
	if err := client.GetJSON(fmt.Sprintf("%s/shelly", util.DefaultScheme(uri, "http")), &resp); err != nil {
		return nil, err
	}
	// Determine device model/type
	model := strings.Split(resp.Type+resp.Model, "-")
	modelgroup := model[0]
	// Set default profile to "monophase" if not provided
	if resp.Profile == "" {
		resp.Profile = "monophase"
	}

	c := &Connection{
		Helper:  client,
		channel: channel,
		Gen:     resp.Gen,
		model:   modelgroup,
		app:     resp.App,
		profile: resp.Profile,
		Cache:   cache,
	}

	c.Client.Transport = request.NewTripper(log, transport.Insecure())

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return c, fmt.Errorf("%s (%s) missing user/password", c.model, resp.Mac)
	}

	switch c.Gen {
	case 0, 1:
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		c.uri = util.DefaultScheme(uri, "http")
		if user != "" {
			log.Redact(transport.BasicAuthHeader(user, password))
			c.Client.Transport = transport.BasicAuth(user, password, c.Client.Transport)
		}

	case 2, 3:
		c.gen2InitApi(uri, user, password)

	default:
		return c, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, c.Gen)
	}

	return c, nil
}
