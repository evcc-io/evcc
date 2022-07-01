package homematic

// Credits to
// https://github.com/dhcgn/hm2mqtt/blob/d05ef217ec5aa667e43401587e41c60aa4168d31/hmclient/hmclient.go
// https://github.com/Bug405/Homematic-XML-RPC/blob/master/src/main/java/xmlrpc/HomematicClient.java
// https://github.com/homematic-community/awesome-homematic

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

type MethodParam struct {
	ParamString string `xml:"value>string,omitempty"`
}

type MethodCall struct {
	XMLName    xml.Name      `xml:"methodCall"`
	MethodName string        `xml:"methodName"`
	Params     []MethodParam `xml:"params>param,omitempty"`
}

type MethodResponseValue struct {
	XMLName     xml.Name `xml:"value"`
	BoolValue   int64    `xml:"boolean"`
	FloatValue  float64  `xml:"double"`
	IntValue    int64    `xml:"i4"`
	StringValue string   `xml:"string"`
}

type MethodResponse struct {
	XMLName xml.Name            `xml:"methodResponse"`
	Value   MethodResponseValue `xml:"params>param>value,omitempty"`
}

// NewConnection creates a new Homematic device connection.
// User is encoded by using MessageDigest of SHA1 which is afterwards B64 encoded.
// Password is directly B64 encoded.
func NewConnection(uri, user, password, deviceid, meterid, switchid string) *Connection {
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

	hmc := MethodCall{
		XMLName:    xml.Name{},
		MethodName: method,
		Params:     []MethodParam{{param1}, {param2}, {param3}},
	}

	hmr := MethodResponse{}

	body, err := xml.MarshalIndent(hmc, "", "  ")
	if err != nil {
		return hmr, err
	}

	headers := map[string]string{
		"Content-Type": "text/xml",
	}

	c.log.TRACE.Printf("request: %s\n", xml.Header+string(body))

	fmt.Printf("request: %s\n", xml.Header+string(body))

	if req, err := request.New(http.MethodPost, c.URI, strings.NewReader(xml.Header+string(body)), headers); err == nil {
		if res, err := c.DoBody(req); err == nil {
			fmt.Printf("response: %s\n", strings.Replace(string(res), "ISO-8859-1", "UTF-8", 1))

			xml.Unmarshal([]byte(strings.Replace(string(res), "ISO-8859-1", "UTF-8", 1)), &hmr)
		}
	}
	return hmr, err
}

//GetSwitchState reads the homematic switch state true=on/false=off
func (c *Connection) GetSwitchState() (bool, error) {
	sr, err := c.XmlCmd("getValue", fmt.Sprintf("%s:%s", c.DeviceId, c.SwitchId), "STATE", "")
	return sr.Value.BoolValue == 1, err
}

//SetSwitchState sets the homematic switch state true=on/false=off
func (c *Connection) SetSwitchState(bool) (bool, error) {
	sr, err := c.XmlCmd("setValue", fmt.Sprintf("%s:%s", c.DeviceId, c.SwitchId), "STATE", "true")
	return sr.Value.BoolValue == 1, err
}

//GetMeterPower reads the homematic meter power in W
func (c *Connection) GetMeterPower() (float64, error) {
	sr, err := c.XmlCmd("getValue", fmt.Sprintf("%s:%s", c.DeviceId, c.MeterId), "POWER", "")
	return sr.Value.FloatValue, err
}
