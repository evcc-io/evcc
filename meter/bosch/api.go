package bosch

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

type API struct {
	*request.Helper
	uri     string
	status  StatusResponse
	login   LoginResponse
	updated time.Time
	cache   time.Duration
}

var Instances = new(sync.Map)

func NewLocal(log *util.Logger, uri string, cache time.Duration) *API {
	api := &API{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		cache:  cache,
	}

	// ignore the self signed certificate
	api.Client.Transport = request.NewTripper(log, transport.Insecure())
	// create cookie jar to save login tokens
	api.Client.Jar, _ = cookiejar.New(nil)

	return api
}

func (c *API) Login() (err error) {
	req, err := request.New(http.MethodGet, c.uri, nil, nil)
	if err != nil {
		return err
	}

	body, err := c.DoBody(req)
	if err != nil {
		return err
	}

	return c.extractWuiSidFromBody(string(body))
}

func (c *API) Status() (StatusResponse, error) {
	var err error
	if time.Since(c.updated) > c.cache {
		if err = c.updateValues(); err == nil {
			c.updated = time.Now()
		}
	}
	return c.status, err
}

func (c *API) extractWuiSidFromBody(body string) error {
	index := strings.Index(body, "WUI_SID=")

	if index < 0 || len(body) < index+9+15 {
		c.login.wuSid = ""
		return fmt.Errorf("error while extracting wui sid. body was= %s", body)
	}

	c.login.wuSid = body[index+9 : index+9+15]

	return nil
}

func (c *API) updateValues() error {
	data := "action=get.hyb.overview&flow=1"

	uri := c.uri + "/cgi-bin/ipcclient.fcgi?" + c.login.wuSid
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), map[string]string{
		"Content-Type": "text/plain",
	})
	if err != nil {
		return err
	}

	body, err := c.DoBody(req)
	if err != nil {
		return err
	}

	return c.extractValues(string(body))
}

func (c *API) extractValues(body string) error {
	if strings.Contains(body, "session invalid") {
		return c.Login()
	}

	values := strings.Split(body, "|")

	if len(values) < 14 {
		return fmt.Errorf("extractValues: response has not enough values")
	}

	soc, err := strconv.Atoi(values[3])
	if err == nil {
		c.status.CurrentBatterySoc = float64(soc)
		c.status.SellToGrid, err = parseWattValue(values[11])
	}

	if err == nil {
		c.status.BuyFromGrid, err = parseWattValue(values[14])
	}

	if err == nil {
		c.status.PvPower, err = parseWattValue(values[2])
	}

	if err == nil {
		c.status.BatteryChargePower, err = parseWattValue(values[10])
	}

	if err == nil {
		c.status.BatteryDischargePower, err = parseWattValue(values[13])
	}

	return err
}

func parseWattValue(inputString string) (float64, error) {
	if len(strings.TrimSpace(inputString)) == 0 || strings.Contains(inputString, "nbsp;") {
		return 0.0, nil
	}

	num := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(inputString, "kW", " "), "von", " "))
	res, err := strconv.ParseFloat(num, 64)

	return res * 1000.0, err
}
