package shelly_pro_3em

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
}

// NewConnection creates a new Shelly device connection.
func NewConnection(uri, user, password string, channel int) (*Connection, error) {
	if uri == "" {
		return nil, errors.New("missing uri")
	}

	for _, suffix := range []string{"/", "/rcp", "/shelly"} {
		uri = strings.TrimSuffix(uri, suffix)
	}

	log := util.NewLogger("shelly-pro-3em")
	client := request.NewHelper(log)

	// Shelly Gen1 and Gen2 families expose the /shelly endpoint
	var resp DeviceInfo
	if err := client.GetJSON(fmt.Sprintf("%s/shelly", util.DefaultScheme(uri, "http")), &resp); err != nil {
		return nil, err
	}

	log.INFO.Println("resp: ", resp.Id)

	conn := &Connection{
		Helper:     client,
		channel:    channel,
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	if (resp.Auth || resp.AuthEn) && (user == "" || password == "") {
		return conn, fmt.Errorf("%s (%s) missing user/password", resp.Model, resp.Mac)
	}

	// Shelly GEN 2 API
	// https://shelly-api-docs.shelly.cloud/gen2/
	conn.uri = fmt.Sprintf("%s/rpc", util.DefaultScheme(uri, "http"))
	if user != "" {
		conn.Client.Transport = digest.NewTransport(user, password, conn.Client.Transport)
	}

	return conn, nil
}

// CurrentPower implements the api.Meter interface (Meter provides total active power in W)
func (d *Connection) CurrentPower() (float64, error) {
	var power float64
	var res Gen2EmStatusResponse
	if err := d.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, err
	}

	power = res.TotalPower

	return power, nil
}

// TotalEnergy implements the api.Meter interface (MeterEnergy provides total energy in kWh)
func (d *Connection) TotalEnergy() (float64, error) {
	var energy float64
	var res Gen2EmDataStatusResponce
	if err := d.execGen2Cmd("EMData.GetStatus", false, &res); err != nil {
		return 0, err
	}

	energy = res.TotalEnergy

	return energy, nil
}

// Currents implements the api.Meter interface (PhaseCurrents provides per-phase current A)
func (d *Connection) Currents() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := d.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.CurrentA, res.CurrentB, res.CurrentC, nil
}

// Voltages implements the api.Meter interface (PhaseVoltages provides per-phase voltage V)
func (d *Connection) Voltages() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := d.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.VoltageA, res.VoltageB, res.VoltageC, nil
}

// Powers implements the api.Meter interface (PhasePowers provides signed per-phase power W)
func (d *Connection) Powers() (float64, float64, float64, error) {
	var res Gen2EmStatusResponse
	if err := d.execGen2Cmd("EM.GetStatus", false, &res); err != nil {
		return 0, 0, 0, err
	}

	return res.PowerA, res.PowerB, res.PowerC, nil
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
