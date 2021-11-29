package charger

// LICENSE

// Copyright (c) 2019-2021 andig

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/philippseith/signalr"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
)

// Easee charger implementation
type Easee struct {
	*request.Helper
	charger               string
	updated               time.Time
	chargeStatus          api.ChargeStatus
	log                   *util.Logger
	mux                   sync.Mutex
	dynamicChargerCurrent int64
	current               int64
	chargerEnabled        bool
	enabledStatus         bool
	currentPower, sessionEnergy,
	currentL1, currentL2, currentL3 float64
}

func init() {
	registry.Add("easee", NewEaseeFromConfig)
}

// NewEaseeFromConfig creates a go-e charger from generic config
func NewEaseeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User     string
		Password string
		Charger  string
		Circuit  int
		Cache    time.Duration // deprecated
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewEasee(cc.User, cc.Password, cc.Charger, cc.Circuit, cc.Cache)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string, circuit int, cache time.Duration) (*Easee, error) {
	log := util.NewLogger("easee").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Easee{
		Helper:  request.NewHelper(log),
		charger: charger,
		log:     log,
		current: 6, // default current
	}

	ts, err := easee.TokenSource(log, user, password)
	if err != nil {
		return c, err
	}

	// replace client transport with authenticated transport
	c.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   c.Client.Transport,
	}

	// find charger
	if charger == "" {
		chargers, err := c.chargers()
		if err != nil {
			return c, err
		}

		if len(chargers) != 1 {
			return c, fmt.Errorf("cannot determine charger id, found: %v", funk.Map(chargers, func(c easee.Charger) string { return c.ID }))
		}

		c.charger = chargers[0].ID
	}

	client, err := signalr.NewClient(context.Background(),
		signalr.WithAutoReconnect(c.connect(ts)),
		signalr.WithReceiver(c),
		signalr.Logger(easee.SignalrLogger(c.log.TRACE), false),
	)

	if err == nil {
		client.Start()

		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()
		err = <-client.WaitForState(ctx, signalr.ClientConnected)
	}

	if err == nil {
		err = <-client.Send("SubscribeWithCurrentState", c.charger, true)
	}

	return c, err
}

// subscribe connects to the signalR hub
func (c *Easee) connect(ts oauth2.TokenSource) func() (signalr.Connection, error) {
	return func() (signalr.Connection, error) {
		tok, err := ts.Token()
		if err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()

		return signalr.NewHTTPConnection(ctx, "https://api.easee.cloud/hubs/chargers",
			signalr.WithHTTPClient(c.Client),
			signalr.WithHTTPHeaders(func() (res http.Header) {
				return http.Header{
					"Authorization": []string{fmt.Sprintf("Bearer %s", tok.AccessToken)},
				}
			}),
		)
	}
}

func (c *Easee) observe(typ string, i json.RawMessage) {
	var res easee.Observation
	err := json.Unmarshal(i, &res)
	if err != nil {
		c.log.ERROR.Printf("invalid message: %s %s %v", i, typ, err)
		return
	}

	var value interface{}

	switch res.DataType {
	case easee.Boolean:
		value = res.Value == "1"
	case easee.Double:
		value, err = strconv.ParseFloat(res.Value, 64)
		if err != nil {
			c.log.ERROR.Println(err)
			return
		}
	case easee.Integer:
		value, err = strconv.Atoi(res.Value)
		if err != nil {
			c.log.ERROR.Println(err)
			return
		}
	}

	c.mux.Lock()
	defer c.mux.Unlock()

	switch res.ID {
	case easee.IS_ENABLED:
		c.chargerEnabled = value.(bool)
	case easee.TOTAL_POWER:
		c.currentPower = 1e3 * value.(float64)
	case easee.SESSION_ENERGY:
		c.sessionEnergy = value.(float64)
	case easee.IN_CURRENT_T3:
		c.currentL1 = value.(float64)
	case easee.IN_CURRENT_T4:
		c.currentL2 = value.(float64)
	case easee.IN_CURRENT_T5:
		c.currentL3 = value.(float64)
	case easee.DYNAMIC_CHARGER_CURRENT:
		c.dynamicChargerCurrent = int64(value.(float64))
		// ensure that charger current matches evcc's expectation
		if c.dynamicChargerCurrent > 0 && c.dynamicChargerCurrent != c.current {
			if err = c.MaxCurrent(c.current); err != nil {
				c.log.ERROR.Println(err)
			}
		}
	case easee.CHARGER_OP_MODE:
		switch value.(int) {
		case easee.ModeDisconnected:
			c.chargeStatus = api.StatusA
		case easee.ModeAwaitingStart, easee.ModeCompleted, easee.ModeReadyToCharge:
			c.chargeStatus = api.StatusB
		case easee.ModeCharging:
			c.chargeStatus = api.StatusC
		case easee.ModeError:
			c.chargeStatus = api.StatusF
		default:
			c.chargeStatus = api.StatusNone
			c.log.ERROR.Printf("unknown opmode: %d", value.(int))
		}
		c.enabledStatus = value.(int) == easee.ModeCharging ||
			value.(int) == easee.ModeAwaitingStart ||
			value.(int) == easee.ModeCompleted ||
			value.(int) == easee.ModeReadyToCharge
	}

	c.log.TRACE.Printf("%s %s: %s %.4v", typ, res.Mid, res.ID, value)
	c.updated = time.Now()
}

// ProductUpdate implements the signalr receiver
func (c *Easee) ProductUpdate(i json.RawMessage) {
	c.observe("ProductUpdate", i)
}

// ChargerUpdate implements the signalr receiver
func (c *Easee) ChargerUpdate(i json.RawMessage) {
	// c.observe("ChargerUpdate", i)
}

// CommandResponse implements the signalr receiver
func (c *Easee) CommandResponse(i json.RawMessage) {
	// c.observe("CommandResponse", i)
}

func (c *Easee) chargers() (res []easee.Charger, err error) {
	uri := fmt.Sprintf("%s/chargers", easee.API)

	req, err := request.New(http.MethodGet, uri, nil, request.JSONEncoding)
	if err == nil {
		err = c.DoJSON(req, &res)
	}

	return res, err
}

// Status implements the api.Charger interface
func (c *Easee) Status() (api.ChargeStatus, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.chargeStatus, nil
}

// Enabled implements the api.Charger interface
func (c *Easee) Enabled() (bool, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.enabledStatus && c.dynamicChargerCurrent > 0, nil
}

// Enable implements the api.Charger interface
func (c *Easee) Enable(enable bool) error {
	c.mux.Lock()
	enablingRequired := enable && !c.chargerEnabled
	c.mux.Unlock()

	// enable charger once if it's switched off
	if enablingRequired {
		data := easee.ChargerSettings{
			Enabled: &enable,
		}

		uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
		resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	// resume/stop charger
	action := easee.ChargePause
	if enable {
		action = easee.ChargeResume
	}

	uri := fmt.Sprintf("%s/chargers/%s/commands/%s", easee.API, c.charger, action)
	_, err := c.Post(uri, request.JSONContent, nil)

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Easee) MaxCurrent(current int64) error {
	return c.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Easee)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (c *Easee) MaxCurrentMillis(current float64) error {
	data := easee.ChargerSettings{
		DynamicChargerCurrent: &current,
	}

	uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
	resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err == nil {
		c.current = int64(current)
		resp.Body.Close()
	}

	return err
}

var _ api.ChargePhases = (*Easee)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (c *Easee) Phases1p3p(phases int) error {
	if phases == 3 {
		phases = 2
	}

	data := easee.ChargerSettings{
		PhaseMode: &phases,
	}

	uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
	resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err == nil {
		resp.Body.Close()
	}

	return err
}

var _ api.Meter = (*Easee)(nil)

// CurrentPower implements the api.Meter interface
func (c *Easee) CurrentPower() (float64, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.currentPower, nil
}

var _ api.ChargeRater = (*Easee)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Easee) ChargedEnergy() (float64, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.sessionEnergy, nil
}

var _ api.MeterCurrent = (*Easee)(nil)

// Currents implements the api.MeterCurrent interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.currentL1, c.currentL2, c.currentL3, nil
}
