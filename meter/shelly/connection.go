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
	uri     string
	channel int
	gen     int // Shelly api generation
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
		Helper:  client,
		channel: channel,
		gen:     resp.Gen,
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

	case 2:
		// Shelly GEN 2 API
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

// CurrentPower implements the api.Meter interface
func (d *Connection) CurrentPower() (float64, error) {
	var power float64
	switch d.gen {
	case 0, 1:
		var res Gen1StatusResponse
		uri := fmt.Sprintf("%s/status", d.uri)
		if err := d.GetJSON(uri, &res); err != nil {
			return 0, err
		}

		switch {
		case d.channel < len(res.Meters):
			power = res.Meters[d.channel].Power
		case d.channel < len(res.EMeters):
			power = res.EMeters[d.channel].Power
		default:
			return 0, errors.New("invalid channel, missing power meter")
		}

	default:
		var res Gen2StatusResponse
		if err := d.execGen2Cmd("Shelly.GetStatus", false, &res); err != nil {
			return 0, err
		}

		switch d.channel {
		case 1:
			power = res.Switch1.Apower
		case 2:
			power = res.Switch2.Apower
		default:
			power = res.Switch0.Apower
		}
	}

	return power, nil
}

// Enabled implements the api.Charger interface
func (d *Connection) Enabled() (bool, error) {
	switch d.gen {
	case 0, 1:
		var res Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d", d.uri, d.channel)
		err := d.GetJSON(uri, &res)
		return res.Ison, err

	default:
		var res Gen2SwitchResponse
		err := d.execGen2Cmd("Switch.GetStatus", false, &res)
		return res.Output, err
	}
}

// Enable implements the api.Charger interface
func (d *Connection) Enable(enable bool) error {
	var err error
	onoff := map[bool]string{true: "on", false: "off"}

	switch d.gen {
	case 0, 1:
		var res Gen1SwitchResponse
		uri := fmt.Sprintf("%s/relay/%d?turn=%s", d.uri, d.channel, onoff[enable])
		err = d.GetJSON(uri, &res)

	default:
		var res Gen2SwitchResponse
		err = d.execGen2Cmd("Switch.Set", enable, &res)
	}

	return err
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

// TotalEnergy implements the api.Meter interface
func (d *Connection) TotalEnergy() (float64, error) {
	var energy float64
	switch d.gen {
	case 0, 1:
		var res Gen1StatusResponse
		uri := fmt.Sprintf("%s/status", d.uri)
		if err := d.GetJSON(uri, &res); err != nil {
			return 0, err
		}

		switch {
		case d.channel < len(res.Meters):
			energy = res.Meters[d.channel].Total / 60
		case d.channel < len(res.EMeters):
			energy = res.EMeters[d.channel].Total / 60
		default:
			return 0, errors.New("invalid channel, missing power meter")
		}

	default:
		var res Gen2StatusResponse
		if err := d.execGen2Cmd("Shelly.GetStatus", false, &res); err != nil {
			return 0, err
		}

		switch d.channel {
		case 1:
			energy = res.Switch1.Aenergy.Total
		case 2:
			energy = res.Switch2.Aenergy.Total
		default:
			energy = res.Switch0.Aenergy.Total
		}
	}

	return energy / 1000, nil
}
