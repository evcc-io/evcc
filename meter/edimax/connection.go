package edimax

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Edimax SP-2101W smart plug local API (port 10000)

const (
	getStatus = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="get"><Device.System.Power.State></Device.System.Power.State><NOW_POWER><Device.System.Power.NowPower></Device.System.Power.NowPower></NOW_POWER></CMD></SMARTPLUG>`
	setState  = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="setup"><Device.System.Power.State>%s</Device.System.Power.State></CMD></SMARTPLUG>`
)

type status struct {
	State string  `xml:"CMD>Device.System.Power.State"`
	Power float64 `xml:"CMD>NOW_POWER>Device.System.Power.NowPower"`
}

type setupResponse struct {
	Result string `xml:"CMD"`
}

// Connection is the Edimax smart plug connection
type Connection struct {
	*request.Helper
	ctx     context.Context
	uri     string
	statusG util.Cacheable[status]
}

// NewConnection creates an Edimax smart plug connection
func NewConnection(ctx context.Context, host, user, password string, cache time.Duration) *Connection {
	log := util.NewLogger("edimax").Redact(transport.BasicAuthHeader(user, password))

	c := &Connection{
		Helper: request.NewHelper(log),
		ctx:    ctx,
		uri:    fmt.Sprintf("http://%s/smartplug.cgi", util.DefaultPort(host, 10000)),
	}

	// the plug never redirects, don't forward credentials elsewhere
	c.Client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	if user != "" {
		c.Client.Transport = newAuthTransport(user, password, c.Client.Transport)
	}

	c.statusG = util.ResettableCached(c.status, cache)

	return c
}

func (c *Connection) post(body string, res any) error {
	req, err := request.New(http.MethodPost, c.uri, strings.NewReader(body), request.XMLEncoding)
	if err != nil {
		return err
	}
	req = req.WithContext(c.ctx)

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := request.ResponseError(resp); err != nil {
		return err
	}

	// the plug declares the non-standard encoding "UTF8"
	dec := xml.NewDecoder(resp.Body)
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	return dec.Decode(res)
}

func (c *Connection) status() (status, error) {
	var res status
	if err := c.post(getStatus, &res); err != nil {
		return status{}, err
	}

	res.State = strings.ToUpper(strings.TrimSpace(res.State))
	if res.State != "ON" && res.State != "OFF" {
		return status{}, fmt.Errorf("invalid state: %q", res.State)
	}

	return res, nil
}

// Enabled returns the relay state
func (c *Connection) Enabled() (bool, error) {
	res, err := c.statusG.Get()
	return res.State == "ON", err
}

// Enable switches the relay
func (c *Connection) Enable(enable bool) error {
	defer c.statusG.Reset()

	onoff := map[bool]string{true: "ON", false: "OFF"}

	var res setupResponse
	if err := c.post(fmt.Sprintf(setState, onoff[enable]), &res); err != nil {
		return err
	}

	if r := strings.TrimSpace(res.Result); !strings.EqualFold(r, "OK") {
		return fmt.Errorf("switch %s failed: %s", strings.ToLower(onoff[enable]), r)
	}

	return nil
}

var _ api.Meter = (*Connection)(nil)

// CurrentPower implements the api.Meter interface
func (c *Connection) CurrentPower() (float64, error) {
	res, err := c.statusG.Get()
	return res.Power, err
}
