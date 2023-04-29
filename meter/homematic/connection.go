package homematic

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/provider"
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
	Cache                                                    time.Duration
}

// Connection is the Homematic CCU connection
type Connection struct {
	log *util.Logger
	*request.Helper
	*Settings
	meterCache    provider.Cacheable[MethodResponse]
	meterUpdated  time.Time
	switchCache   provider.Cacheable[MethodResponse]
	switchUpdated time.Time
}

// NewConnection creates a new Homematic device connection.
func NewConnection(uri, device, meterchannel, switchchannel, user, password string, cache time.Duration) (*Connection, error) {
	log := util.NewLogger("homematic")

	settings := &Settings{
		URI:           util.DefaultScheme(uri, "http"),
		Device:        device,
		MeterChannel:  meterchannel,
		SwitchChannel: switchchannel,
		Cache:         cache,
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

	conn.switchCache = provider.ResettableCached(func() (MethodResponse, error) {
		var res MethodResponse
		var err error
		res, err = conn.XmlCmd("getParamset", conn.SwitchChannel, Param{CCUString: "VALUES"})
		return res, err
	}, conn.Cache)

	conn.meterCache = provider.ResettableCached(func() (MethodResponse, error) {
		var res MethodResponse
		var err error
		res, err = conn.XmlCmd("getParamset", conn.MeterChannel, Param{CCUString: "VALUES"})
		return res, err
	}, conn.Cache)

	return conn, nil
}

// reset caches
func (c *Connection) reset() {
	c.switchCache.Reset()
	c.meterCache.Reset()
}

// Enable sets the homematic HMIP-PSM switchchannel state to true=on/false=off
func (c *Connection) Enable(enable bool) error {
	onoff := map[bool]string{true: "1", false: "0"}
	_, err := c.XmlCmd("setValue", c.SwitchChannel, Param{CCUString: "STATE"}, Param{CCUBool: onoff[enable]})
	return err
}

// Enabled reads the homematic HMIP-PSM switchchannel state true=on/false=off
func (c *Connection) Enabled() (bool, error) {
	_, ccuBool, err := c.getParamsetValue("STATE")
	return ccuBool, err
}

// CurrentPower reads the homematic HMIP-PSM meterchannel power in W
func (c *Connection) CurrentPower() (float64, error) {
	ccuFloat, _, err := c.getParamsetValue("POWER")
	return ccuFloat, err
}

// TotalEnergy reads the homematic HMIP-PSM meterchannel energy in Wh
func (c *Connection) TotalEnergy() (float64, error) {
	ccuFloat, _, err := c.getParamsetValue("ENERGY_COUNTER")
	return ccuFloat / 1000, err
}

// Currents reads the homematic HMIP-PSM meterchannel L1 current in A
func (c *Connection) Currents() (float64, float64, float64, error) {
	ccuFloat, _, err := c.getParamsetValue("CURRENT")
	return ccuFloat / 1000, 0, 0, err
}

// GridCurrentPower reads the homematic HM-ES-TX-WM grid meterchannel power in W
func (c *Connection) GridCurrentPower() (float64, error) {
	ccuFloat, _, err := c.getParamsetValue("IEC_POWER")
	return ccuFloat, err
}

// GridTotalEnergy reads the homematic HM-ES-TX-WM grid meterchannel energy in Wh
func (c *Connection) GridTotalEnergy() (float64, error) {
	ccuFloat, _, err := c.getParamsetValue("IEC_ENERGY_COUNTER")
	return ccuFloat, err
}

func (c *Connection) XmlCmd(method, channel string, values ...Param) (MethodResponse, error) {
	target := fmt.Sprintf("%s:%s", c.Device, channel)
	hmc := MethodCall{
		XMLName:    xml.Name{},
		MethodName: method,
		Params:     append([]Param{{CCUString: target}}, values...),
	}

	var hmr MethodResponse
	body, err := xml.Marshal(hmc)
	if err != nil {
		return hmr, err
	}

	req, err := request.New(http.MethodPost, c.URI, strings.NewReader(xml.Header+string(body)), map[string]string{
		"Content-Type": "text/xml",
	})
	if err != nil {
		return hmr, err
	}

	res, err := c.DoBody(req)
	if err != nil {
		return hmr, err
	}

	// correct Homematic IP Legacy API (CCU port 2010) method response encoding value
	res = []byte(strings.Replace(string(res), "ISO-8859-1", "UTF-8", 1))

	// correct XML-RPC-Schnittstelle (CCU port 2001) method response encoding value
	res = []byte(strings.Replace(string(res), "iso-8859-1", "UTF-8", 1))

	if err := xml.Unmarshal(res, &hmr); err != nil {
		return hmr, err
	}

	return hmr, parseError(hmr)
}

// getParamsetValue reads all parameter values of a device channel
func (c *Connection) getParamsetValue(valueName string) (float64, bool, error) {
	var res MethodResponse
	var err error

	if valueName == "STATE" {
		res, err = c.switchCache.Get()
	} else {
		res, err = c.meterCache.Get()
	}

	return getFloatValue(res, valueName), getBoolValue(res, valueName), err
}

// getCCUFloat selects a float value of a CCU API response member
func getFloatValue(res MethodResponse, valueName string) float64 {
	var ccuFloat float64

	for _, m := range res.Member {
		if m.Name == valueName {
			ccuFloat = m.Value.CCUFloat
		}
	}

	return ccuFloat
}

// getCCUBool selects a float value of a CCU API response member
func getBoolValue(res MethodResponse, valueName string) bool {
	var ccuBool bool

	for _, m := range res.Member {
		if m.Name == valueName {
			ccuBool = m.Value.CCUBool
		}
	}

	return ccuBool
}

// parseError checks on Homematic CCU error codes
// Refer to page 30 of https://homematic-ip.com/sites/default/files/downloads/HM_XmlRpc_API.pdf
func parseError(res MethodResponse) error {
	var faultCode int64
	var faultString string

	faultCode = 0
	for _, f := range res.Fault {
		if f.Name == "faultCode" {
			faultCode = f.Value.CCUInt
		}
		if f.Name == "faultString" {
			faultString = f.Value.CCUString
		}
	}

	if faultString == "" {
		faultString = "Unknown Homematic API Error"
	}

	if faultCode != 0 {
		return fmt.Errorf("%s (%v)", faultString, faultCode)
	}

	return nil
}
