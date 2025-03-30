package shelly

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Connection is the Shelly connection
type Connection struct {
	*request.Helper
	uri     string
	channel int
	Gen     int    // Shelly api generation
	model   string // Shelly device model
	app     string // Shelly device app code
	profile string // Shelly device profile
}

// NewConnection creates a new Shelly device connection.
func NewConnection(uri, user, password string, channel int) (*Connection, error) {
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

	conn := &Connection{
		Helper:  client,
		channel: channel,
		Gen:     resp.Gen,
		model:   modelgroup,
		app:     resp.App,
		profile: resp.Profile,
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return conn, fmt.Errorf("%s (%s) missing user/password", conn.model, resp.Mac)
	}

	switch conn.Gen {
	case 0, 1:
		// Shelly GEN 1 API
		// https://shelly-api-docs.shelly.cloud/gen1/#shelly-family-overview
		conn.uri = util.DefaultScheme(uri, "http")
		if user != "" {
			log.Redact(transport.BasicAuthHeader(user, password))
			conn.Client.Transport = transport.BasicAuth(user, password, conn.Client.Transport)
		}

	case 2, 3:
		// Shelly GEN 2+ API
		// https://shelly-api-docs.shelly.cloud/gen2/
		conn.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http"))
		if user != "" {
			conn.Client.Transport = digest.NewTransport(user, password, conn.Client.Transport)
		}

	default:
		return conn, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, conn.Gen)
	}

	return conn, nil
	// return conn, fmt.Errorf("%s (%s) unknown api generation (%d)", conn.model, resp.Model, conn.gen)
}
