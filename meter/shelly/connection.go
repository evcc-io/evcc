package shelly

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// Connection is the Shelly connection
type Connection struct {
	*request.Helper
	uri        string
	channel    int
	gen        int    // Shelly api generation
	devicetype string // Shelly device type
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

	conn := &Connection{
		Helper:     client,
		channel:    channel,
		gen:        resp.Gen,
		devicetype: resp.Type,
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return conn, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	switch conn.gen {
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
		return conn, fmt.Errorf("%s (%s) unknown api generation (%d)", resp.Type, resp.Model, conn.gen)
	}

	return conn, nil
}

// execGen2Cmd executes a shelly api gen1/gen2 command and provides the response
func (d *Connection) execGen2Cmd(method string, enable bool, res interface{}) error {
	// Shelly gen 2 rfc7616 authentication
	// https://shelly-api-docs.shelly.cloud/gen2/Overview/CommonDeviceTraits#authentication
	// https://datatracker.ietf.org/doc/html/rfc7616

	data := &Gen2RpcPost{
		Id:     d.channel,
		On:     enable,
		Src:    "evcc",
		Method: method,
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/%s", d.uri, method), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	return d.DoJSON(req, &res)
}
