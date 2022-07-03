package homematic

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Homematic plugable switchchannel and meterchannel charger based on CCU XML-RPC interface
// https://homematic-ip.com/sites/default/files/downloads/HM_XmlRpc_API.pdf
// https://homematic-ip.com/sites/default/files/downloads/HMIP_XmlRpc_API_Addendum.pdf

// Homematic CCU settings
type Settings struct {
	URI, Device, MeterChannel, SwitchChannel, User, Password string
}

// Connection is the Homematic CCU connection
type Connection struct {
	log *util.Logger
	*request.Helper
	*Settings
}

// NewConnection creates a new Homematic device connection.
func NewConnection(uri, device, meterchannel, switchchannel, user, password string) (*Connection, error) {
	log := util.NewLogger("homematic")

	settings := &Settings{
		URI:           util.DefaultScheme(uri, "http"),
		Device:        device,
		MeterChannel:  meterchannel,
		SwitchChannel: switchchannel,
	}

	conn := &Connection{
		log:      log,
		Helper:   request.NewHelper(log),
		Settings: settings,
	}

	conn.Client.Transport = request.NewTripper(log, transport.Insecure())

	if user != "" {
		log.Redact(transport.BasicAuthHeader(user, password))
		conn.Client.Transport = transport.BasicAuth(user, password, conn.Client.Transport)
	}

	//lint:ignore nothing to return as error, but needed to fullfill api.meter requirements
	return conn, nil
}

func (c *Connection) XmlCmd(method string, param1, param2, param3 ParamValue) (MethodResponse, error) {
	var body []byte
	var err error
	hmr := MethodResponse{}

	hmc := MethodCall{
		XMLName:    xml.Name{},
		MethodName: method,
		Params:     []ParamValue{param1, param2, param3},
	}
	body, err = xml.MarshalIndent(hmc, "", "  ")
	if err != nil {
		return hmr, err
	}

	headers := map[string]string{
		"Content-Type": "text/xml",
	}

	c.log.TRACE.Printf("request: %s\n", xml.Header+string(body))

	if req, err := request.New(http.MethodPost, c.URI, strings.NewReader(xml.Header+string(body)), headers); err == nil {
		if res, err := c.DoBody(req); err == nil {
			c.log.TRACE.Printf("response: %s\n", res)
			err = xml.Unmarshal([]byte(strings.Replace(string(res), "ISO-8859-1", "UTF-8", 1)), &hmr)
			if err != nil {
				return hmr, err
			}
		}
	}

	return hmr, err
}

//Enabled reads the homematic switchchannel state true=on/false=off
func (c *Connection) Enabled() (bool, error) {
	//fmt.Sprintf("%s:%s", c.Device, c.SwitchChannel)
	p1 := ParamValue{CCUString: fmt.Sprintf("%s:%s", c.Device, c.SwitchChannel)}
	p2 := ParamValue{CCUString: "STATE"}
	p3 := ParamValue{CCUString: ""}
	sr, err := c.XmlCmd("getValue", p1, p2, p3)
	return sr.Value.CCUBool == "1", err
}

//Enable sets the homematic switchchannel state true=on/false=off
func (c *Connection) Enable(enable bool) error {
	onoff := map[bool]string{true: "1", false: "0"}
	p1 := ParamValue{CCUString: fmt.Sprintf("%s:%s", c.Device, c.SwitchChannel)}
	p2 := ParamValue{CCUString: "STATE"}
	p3 := ParamValue{CCUBool: onoff[enable]}
	_, err := c.XmlCmd("setValue", p1, p2, p3)
	return err
}

//CurrentPower reads the homematic meterchannel power in W
func (c *Connection) CurrentPower() (float64, error) {
	p1 := ParamValue{CCUString: fmt.Sprintf("%s:%s", c.Device, c.MeterChannel)}
	p2 := ParamValue{CCUString: "POWER"}
	p3 := ParamValue{CCUString: ""}
	sr, err := c.XmlCmd("getValue", p1, p2, p3)
	return sr.Value.CCUFloat, err
}

//TotalEnergy reads the homematic meterchannel power in W
func (c *Connection) TotalEnergy() (float64, error) {
	p1 := ParamValue{CCUString: fmt.Sprintf("%s:%s", c.Device, c.MeterChannel)}
	p2 := ParamValue{CCUString: "ENERGY_COUNTER"}
	p3 := ParamValue{CCUString: ""}
	sr, err := c.XmlCmd("getValue", p1, p2, p3)
	return sr.Value.CCUFloat / 1000, err
}

// Currents implements the api.MeterChannelCurrent interface
func (c *Connection) Currents() (float64, float64, float64, error) {
	p1 := ParamValue{CCUString: fmt.Sprintf("%s:%s", c.Device, c.MeterChannel)}
	p2 := ParamValue{CCUString: "CURRENT"}
	p3 := ParamValue{CCUString: ""}
	sr, err := c.XmlCmd("getValue", p1, p2, p3)
	return sr.Value.CCUFloat / 1000, 0, 0, err
}
