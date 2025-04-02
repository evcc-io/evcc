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

// DeviceInfo is the common /shelly endpoint response
// https://shelly-api-docs.shelly.cloud/#common-http-api
type DeviceInfo struct {
	Gen       int    `json:"gen"`
	Id        string `json:"id"`
	Model     string `json:"model"`
	Type      string `json:"type"`
	Mac       string `json:"mac"`
	App       string `json:"app"`
	Auth      bool   `json:"auth"`
	AuthEn    bool   `json:"auth_en"`
	NumMeters int    `json:"num_meters"`
	Profile   string `json:"profile"`
}

// Connection is the Shelly connection
type Connection struct {
	*request.Helper
	log              util.Logger
	uri              string
	channel          int
	Gen              int    // Shelly api generation
	model            string // Shelly device model
	app              string // Shelly device app code
	profile          string // Shelly device profile
	Cache            time.Duration
	gen1Status       util.Cacheable[Gen1Status]
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

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return nil, fmt.Errorf("missing user/password (%s)", resp.Mac)
	}

	// Set default profile to "monophase" if not provided
	if resp.Profile == "" {
		resp.Profile = "monophase"
	}

	c := &Connection{
		Helper:  client,
		log:     *log,
		channel: channel,
		Gen:     resp.Gen,
		model:   strings.Split(resp.Type+resp.Model, "-")[0],
		app:     resp.App,
		profile: resp.Profile,
		Cache:   cache,
	}

	c.Client.Transport = request.NewTripper(&c.log, transport.Insecure())

	// Initialize the connection to the Shelly API
	switch c.Gen {
	case 0, 1:
		c.gen1InitApi(uri, user, password)
	case 2, 3, 4:
		c.gen2InitApi(uri, user, password)

	default:
		return c, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, c.Gen)
	}

	return c, nil
}

func (c *Connection) Enabled() (bool, error) {
	switch c.Gen {
	case 0, 1:
		return c.Gen1Enabled()
	default:
		return c.Gen2Enabled()
	}
}

func (c *Connection) Enable(enable bool) error {
	switch c.Gen {
	case 0, 1:
		return c.Gen1Enable(enable)
	default:
		return c.Gen2Enable(enable)
	}
}

func (c *Connection) CurrentPower() (float64, error) {
	var power float64
	var err error
	switch c.Gen {
	case 0, 1:
		power, err = c.Gen1CurrentPower()
		if err != nil {
			return 0, err
		}
	default:
		power, err = c.Gen2CurrentPower()
		if err != nil {
			return 0, err
		}
	}
	return power, nil
}

func (c *Connection) TotalEnergy() (float64, float64, error) {
	var energyConsumed float64
	var energyFeedIn float64
	var err error
	switch c.Gen {
	case 0, 1:
		energyConsumed, err = c.Gen1TotalEnergy()
		if err != nil {
			return 0, 0, err
		}
	default:
		energyConsumed, energyFeedIn, err = c.Gen2TotalEnergy()
		if err != nil {
			return 0, 0, err
		}
	}
	return energyConsumed, energyFeedIn, nil
}

func (c *Connection) Currents() (float64, float64, float64, error) {
	switch c.Gen {
	case 0, 1:
		return c.Gen1Currents()
	default:
		return c.Gen2Currents()
	}
}

func (c *Connection) Voltages() (float64, float64, float64, error) {
	switch c.Gen {
	case 0, 1:
		return c.Gen1Voltages()
	default:
		return c.Gen2Voltages()
	}
}

func (c *Connection) Powers() (float64, float64, float64, error) {
	switch c.Gen {
	case 0, 1:
		return c.Gen1Powers()
	default:
		return c.Gen2Powers()
	}
}
