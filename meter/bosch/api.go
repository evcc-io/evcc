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
	logger  *util.Logger
}

var Instances = new(sync.Map)

func NewLocal(log *util.Logger, uri string, cache time.Duration) *API {
	api := &API{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		cache:  cache,
		logger: log,
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

//////////// helpers ////////////////

func (c *API) extractWuiSidFromBody(body string) error {
	index := strings.Index(body, "WUI_SID=")

	if index < 0 {
		c.login.wuSid = ""
		return fmt.Errorf("error while extracting wui sid. body was= %s", body)
	}

	c.login.wuSid = body[index+9 : index+9+15]

	c.logger.DEBUG.Println("extractWuiSidFromBody: result=", c.login.wuSid)

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

	return extractValues(c, string(body))
}

func parseWattValue(inputString string) (float64, error) {
	if len(strings.TrimSpace(inputString)) == 0 || strings.Contains(inputString, "nbsp;") {
		return 0.0, nil
	}

	numberString := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(inputString, "kW", " "), "von", " "))

	res, err := strconv.ParseFloat(numberString, 64)

	return res * 1000.0, err
}

func extractValues(c *API, body string) error {
	if strings.Contains(body, "session invalid") {
		c.logger.DEBUG.Println("extractValues: Session invalid. Performing Re-login")
		return c.Login()
	}

	values := strings.Split(body, "|")

	if len(values) < 14 {
		return fmt.Errorf("extractValues: response has not enough values")
	}

	soc, err := strconv.Atoi(values[3])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 1: %s", err)
	}

	c.status.CurrentBatterySoc = float64(soc)
	c.status.SellToGrid, err = parseWattValue(values[11])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 2: %s", err)
	}

	c.status.BuyFromGrid, err = parseWattValue(values[14])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 3: %s", err)
	}

	c.status.PvPower, err = parseWattValue(values[2])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 4: %s", err)
	}

	c.status.BatteryChargePower, err = parseWattValue(values[10])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 5: %s", err)
	}

	c.status.BatteryDischargePower, err = parseWattValue(values[13])

	c.logger.DEBUG.Println("extractValues: batterieLadeStrom=", c.status.BatteryChargePower, ";currentBatterySocValue=", c.status.CurrentBatterySoc, ";einspeisung=", c.status.SellToGrid, ";pvLeistungWatt=", c.status.PvPower, ";strombezugAusNetz=", c.status.BuyFromGrid, ";verbrauchVonBatterie=", c.status.BatteryDischargePower)

	return err
}
