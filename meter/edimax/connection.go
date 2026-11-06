package edimax

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Edimax SP-2101W smart plug local API (port 10000)

const (
	getStatus = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="get"><Device.System.Power.State></Device.System.Power.State><NOW_POWER><Device.System.Power.NowCurrent></Device.System.Power.NowCurrent><Device.System.Power.NowPower></Device.System.Power.NowPower></NOW_POWER></CMD></SMARTPLUG>`
	setState  = `<?xml version="1.0" encoding="UTF8"?><SMARTPLUG id="edimax"><CMD id="setup"><Device.System.Power.State>%s</Device.System.Power.State></CMD></SMARTPLUG>`
)

// Status is the plug's relay state and live measurement
type Status struct {
	On      bool
	Power   float64
	Current float64
}

type statusResponse struct {
	State   string `xml:"CMD>Device.System.Power.State"`
	Power   string `xml:"CMD>NOW_POWER>Device.System.Power.NowPower"`
	Current string `xml:"CMD>NOW_POWER>Device.System.Power.NowCurrent"`
}

type setupResponse struct {
	Result string `xml:"CMD"`
}

// Connection is the Edimax smart plug connection
type Connection struct {
	*request.Helper
	uri     string
	statusG util.Cacheable[Status]
}

// NewConnection creates an Edimax smart plug connection
func NewConnection(host, user, password string, cache time.Duration) *Connection {
	log := util.NewLogger("edimax").Redact(transport.BasicAuthHeader(user, password))

	c := &Connection{
		Helper: request.NewHelper(log),
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

func parseFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid value: %s", s)
	}
	return f, nil
}

func (c *Connection) status() (Status, error) {
	var res statusResponse
	if err := c.post(getStatus, &res); err != nil {
		return Status{}, err
	}

	var (
		status Status
		err    error
	)

	switch strings.ToUpper(strings.TrimSpace(res.State)) {
	case "ON":
		status.On = true
	case "OFF":
	default:
		return Status{}, fmt.Errorf("invalid state: %q", res.State)
	}

	if status.Power, err = parseFloat(res.Power); err != nil {
		return Status{}, fmt.Errorf("power: %w", err)
	}

	if status.Current, err = parseFloat(res.Current); err != nil {
		return Status{}, fmt.Errorf("current: %w", err)
	}

	return status, nil
}

// Status returns the cached plug status
func (c *Connection) Status() (Status, error) {
	return c.statusG.Get()
}

// Enabled returns the relay state
func (c *Connection) Enabled() (bool, error) {
	res, err := c.Status()
	return res.On, err
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
	res, err := c.Status()
	return res.Power, err
}
