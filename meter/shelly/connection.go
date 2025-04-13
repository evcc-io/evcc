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
	log                util.Logger
	uri                string
	channel            int
	gen                int    // Shelly api generation
	model              string // Shelly device type
	profile            string // Shelly device profile
	Cache              time.Duration
	gen1Status         util.Cacheable[Gen1Status]
	gen2StatusResponse util.Cacheable[Gen2StatusResponse]
	gen2EMStatus       util.Cacheable[Gen2EMStatus]
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
		return nil, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	conn := &Connection{
		Helper:  client,
		log:     *log,
		uri:     util.DefaultScheme(uri, "http"),
		channel: channel,
		gen:     resp.Gen,
		model:   strings.Split(resp.Type+resp.Model, "-")[0],
		profile: resp.Profile,
		Cache:   cache,
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	// Set default profile to "monophase" if not provided
	if resp.Profile == "" {
		resp.Profile = "monophase"
	}

	switch conn.gen {
	case 0, 1:
		conn.gen1InitApi(uri, user, password)
	case 2, 3:
		conn.gen2InitApi(uri, user, password)
	default:
		return conn, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, conn.gen)
	}

	return conn, nil
}
