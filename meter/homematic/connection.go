package homematic

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Homematic CCU settings
type Settings struct {
	URI, DeviceId, MeterId, SwitchId, User, Password string
}

// Connection is the Homematic CCU connection
type Connection struct {
	log *util.Logger
	*request.Helper
	*Settings
}

// NewConnection creates a new Homematic device connection.
// User is encoded by using MessageDigest of SHA1 which is afterwards B64 encoded.
// Password is directly B64 encoded.
func NewConnection(uri, deviceid, meterid, switchid, user, password string) *Connection {
	log := util.NewLogger("homematic")

	settings := &Settings{
		URI:      util.DefaultScheme(uri, "http"),
		DeviceId: deviceid,
		MeterId:  meterid,
		SwitchId: switchid,
	}

	conn := &Connection{
		log:      log,
		Helper:   request.NewHelper(log),
		Settings: settings,
	}

	return conn
}

func (c *Connection) XmlCmd(method, param1, param2, param3 string) (MethodResponse, error) {
	var body []byte
	var err error
	hmr := MethodResponse{}

	if method == "setValue" {
		hmc := MethodSetCall{
			XMLName:    xml.Name{},
			MethodName: method,
			Params:     []ParamValue{{CCUString: param1}, {CCUString: param2}, {CCUBool: param3}},
		}
		body, err = xml.MarshalIndent(hmc, "", "  ")
		if err != nil {
			return hmr, err
		}
	} else {
		hmc := MethodGetCall{
			XMLName:    xml.Name{},
			MethodName: method,
			Params:     []MethodGetParam{{param1}, {param2}, {param3}},
		}
		body, err = xml.MarshalIndent(hmc, "", "  ")
		if err != nil {
			return hmr, err
		}
	}

	headers := map[string]string{
		"Content-Type": "text/xml",
	}

	c.log.TRACE.Printf("request: %s\n", xml.Header+string(body))

	fmt.Printf("request: %s\n", xml.Header+string(body))

	if req, err := request.New(http.MethodPost, c.URI, strings.NewReader(xml.Header+string(body)), headers); err == nil {
		if res, err := c.DoBody(req); err == nil {
			c.log.TRACE.Printf("response: %s\n", res)
			xml.Unmarshal([]byte(strings.Replace(string(res), "ISO-8859-1", "UTF-8", 1)), &hmr)
		}
	}

	return hmr, err
}

//Enabled reads the homematic switch state true=on/false=off
func (c *Connection) Enabled() (bool, error) {
	sr, err := c.XmlCmd("getValue", fmt.Sprintf("%s:%s", c.DeviceId, c.SwitchId), "STATE", "")
	return sr.Value.CCUBool == 1, err
}

//SetSwitchState sets the homematic switch state true=on/false=off
func (c *Connection) Enable(enable bool) (bool, error) {
	onoff := map[bool]string{true: "1", false: "0"}
	sr, err := c.XmlCmd("setValue", fmt.Sprintf("%s:%s", c.DeviceId, c.SwitchId), "STATE", onoff[enable])
	return sr.Value.CCUBool == 1, err
}

//GetMeterPower reads the homematic meter power in W
func (c *Connection) CurrentPower() (float64, error) {
	sr, err := c.XmlCmd("getValue", fmt.Sprintf("%s:%s", c.DeviceId, c.MeterId), "POWER", "")
	return sr.Value.CCUFloat, err
}
