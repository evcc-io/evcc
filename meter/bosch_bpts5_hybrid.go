package meter

// LICENSE

// Bosch is the Bosch BPT-S 5 Hybrid meter

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

// Example config:
// meters:
// - name: bosch_grid
//   type: bosch-bpts5-hybrid
//   uri: http://192.168.178.22
//   usage: grid
// - name: bosch_pv
//   type: bosch-bpts5-hybrid
//   uri: http://192.168.178.22
//   usage: pv
// - name: bosch_battery
//   type: bosch-bpts5-hybrid
//   uri: http://192.168.178.22
//   usage: battery

type BoschBpts5Hybrid struct {
	*request.Helper
	usage                 string
	wuSid                 string
	uri                   string
	updated               time.Time
	cache                 time.Duration
	logger                *util.Logger
	CurrentBatterySoc     float64
	SellToGrid            float64
	BuyFromGrid           float64
	PvPower               float64
	BatteryChargePower    float64
	BatteryDischargePower float64
}

func init() {
	registry.Add("bosch-bpts5-hybrid", NewBoschBpts5HybridFromConfig)
}

//go:generate go run ../cmd/tools/decorate.go -f decorateBoschBpts5Hybrid -b api.Meter -t "api.Battery,SoC,func() (float64, error)"

// NewBoschBpts5HybridFromConfig creates a Bosch BPT-S 5 Hybrid Meter from generic config
func NewBoschBpts5HybridFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI   string
		Usage string
		Cache time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewBoschBpts5Hybrid(cc.URI, cc.Usage, cc.Cache)
}

// NewBoschBpts5Hybrid creates a Bosch BPT-S 5 Hybrid Meter
func NewBoschBpts5Hybrid(uri, usage string, cache time.Duration) (api.Meter, error) {
	log := util.NewLogger("bosch-bpts5-hybrid")

	m := &BoschBpts5Hybrid{
		Helper: request.NewHelper(log),
		usage:  strings.ToLower(usage),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		cache:  cache,
		logger: log,
	}

	// ignore the self signed certificate
	m.Client.Transport = request.NewTripper(log, transport.Insecure())
	// create cookie jar to save login tokens
	m.Client.Jar, _ = cookiejar.New(nil)

	if err := m.login(); err != nil {
		return nil, err
	}

	// decorate api.BatterySoC
	var batterySoC func() (float64, error)
	if usage == "battery" {
		batterySoC = m.batterySoC
	}

	return decorateBoschBpts5Hybrid(m, batterySoC), nil
}

func (c *BoschBpts5Hybrid) login() (err error) {
	req, err := request.New(http.MethodGet, c.uri, nil, nil)
	if err != nil {
		return err
	}

	body, err := c.DoBody(req)
	if err != nil {
		return err
	}

	return extractWuiSidFromBody(c, string(body))
}

// CurrentPower implements the api.Meter interface
func (m *BoschBpts5Hybrid) CurrentPower() (float64, error) {
	err := m.status()

	switch m.usage {
	case "grid":
		return m.BuyFromGrid - m.SellToGrid, err
	case "pv":
		return m.PvPower, err
	case "battery":
		return m.BatteryDischargePower - m.BatteryChargePower, err
	default:
		return 0, err
	}
}

// batterySoC implements the api.Battery interface
func (m *BoschBpts5Hybrid) batterySoC() (float64, error) {
	err := m.status()
	return m.CurrentBatterySoc, err
}

//////////// value retrieval ////////////////

func (c *BoschBpts5Hybrid) status() error {
	var err error
	if time.Since(c.updated) > c.cache {
		if err = c.updateValues(); err == nil {
			c.updated = time.Now()
		}
	}
	return err
}

//////////// helpers ////////////////

func extractWuiSidFromBody(c *BoschBpts5Hybrid, body string) error {
	index := strings.Index(body, "WUI_SID=")

	if index < 0 {
		c.wuSid = ""
		return fmt.Errorf("error while extracting wui sid. body was= %s", body)
	}

	c.wuSid = body[index+9 : index+9+15]

	c.logger.DEBUG.Println("extractWuiSidFromBody: result=", c.wuSid)

	return nil
}

func (c *BoschBpts5Hybrid) updateValues() error {
	data := "action=get.hyb.overview&flow=1"

	uri := c.uri + "/cgi-bin/ipcclient.fcgi?" + c.wuSid
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

func extractValues(c *BoschBpts5Hybrid, body string) error {
	if strings.Contains(body, "session invalid") {
		c.logger.DEBUG.Println("extractValues: Session invalid. Performing Re-login")
		return c.login()
	}

	values := strings.Split(body, "|")

	if len(values) < 14 {
		return fmt.Errorf("extractValues: response has not enough values")
	}

	soc, err := strconv.Atoi(values[3])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 1: %s", err)
	}

	c.CurrentBatterySoc = float64(soc)
	c.SellToGrid, err = parseWattValue(values[11])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 2: %s", err)
	}

	c.BuyFromGrid, err = parseWattValue(values[14])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 3: %s", err)
	}

	c.PvPower, err = parseWattValue(values[2])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 4: %s", err)
	}

	c.BatteryChargePower, err = parseWattValue(values[10])

	if err != nil {
		return fmt.Errorf("extractValues: error during value parsing 5: %s", err)
	}

	c.BatteryDischargePower, err = parseWattValue(values[13])

	c.logger.DEBUG.Println("extractValues: batterieLadeStrom=", c.BatteryChargePower, ";currentBatterySocValue=", c.CurrentBatterySoc, ";einspeisung=", c.SellToGrid, ";pvLeistungWatt=", c.PvPower, ";strombezugAusNetz=", c.BuyFromGrid, ";verbrauchVonBatterie=", c.BatteryDischargePower)

	return err
}
