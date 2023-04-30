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
	meterCache  provider.Cacheable[MethodResponse]
	switchCache provider.Cacheable[MethodResponse]
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
		return conn.XmlCmd("getParamset", conn.SwitchChannel, Param{CCUString: "VALUES"})
	}, conn.Cache)

	conn.meterCache = provider.ResettableCached(func() (MethodResponse, error) {
		return conn.XmlCmd("getParamset", conn.MeterChannel, Param{CCUString: "VALUES"})
	}, conn.Cache)

	return conn, nil
}

// Enable sets the homematic HMIP-PSM switchchannel state to true=on/false=off
func (c *Connection) Enable(enable bool) error {
	onoff := map[bool]string{true: "1", false: "0"}
	_, err := c.XmlCmd("setValue", c.SwitchChannel, Param{CCUString: "STATE"}, Param{CCUBool: onoff[enable]})
	if err == nil {
		c.switchCache.Reset()
		c.meterCache.Reset()
	}
	return err
}

// Enabled reads the homematic HMIP-PSM switchchannel state true=on/false=off
func (c *Connection) Enabled() (bool, error) {
	res, err := c.switchCache.Get()
	if err != nil {
		return false, err
	}
	return getBoolValue(res, "STATE"), nil
}

// CurrentPower reads the homematic HMIP-PSM meterchannel power in W
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.meterCache.Get()
	if err != nil {
		return 0, err
	}
	return getFloatValue(res, "POWER"), nil
}

// TotalEnergy reads the homematic HMIP-PSM meterchannel energy in Wh
func (c *Connection) TotalEnergy() (float64, error) {
	res, err := c.meterCache.Get()
	if err != nil {
		return 0, err
	}
	return getFloatValue(res, "ENERGY_COUNTER") / 1e3, nil
}

// Currents reads the homematic HMIP-PSM meterchannel L1 current in A
func (c *Connection) Currents() (float64, float64, float64, error) {
	res, err := c.meterCache.Get()
	if err != nil {
		return 0, 0, 0, err
	}
	return getFloatValue(res, "CURRENT") / 1e3, 0, 0, nil
}

// GridCurrentPower reads the homematic HM-ES-TX-WM grid meterchannel power in W
func (c *Connection) GridCurrentPower() (float64, error) {
	res, err := c.meterCache.Get()
	if err != nil {
		return 0, err
	}
	return getFloatValue(res, "IEC_POWER"), nil
}

// GridTotalEnergy reads the homematic HM-ES-TX-WM grid meterchannel energy in kWh
func (c *Connection) GridTotalEnergy() (float64, error) {
	res, err := c.meterCache.Get()
	if err != nil {
		return 0, err
	}
	return getFloatValue(res, "IEC_ENERGY_COUNTER"), nil
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

// getCCUFloat selects a float value of a CCU API response member
func getFloatValue(res MethodResponse, valueName string) float64 {
	for _, m := range res.Member {
		if m.Name == valueName {
			return m.Value.CCUFloat
		}
	}

	return 0
}

// getCCUBool selects a float value of a CCU API response member
func getBoolValue(res MethodResponse, valueName string) bool {
	for _, m := range res.Member {
		if m.Name == valueName {
			return m.Value.CCUBool
		}
	}

	return false
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
