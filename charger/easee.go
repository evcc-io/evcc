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

	"github.com/avast/retry-go/v3"
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
	charger        string
	site, circuit  int
	updated        time.Time
	chargeStatus   api.ChargeStatus
	cache          time.Duration
	log            *util.Logger
	mux            sync.Mutex
	phases         int
	chargerEnabled bool
	enabledStatus  bool
	current, currentPower, sessionEnergy,
	circuitTotalPhaseConductorCurrentL1,
	circuitTotalPhaseConductorCurrentL2,
	circuitTotalPhaseConductorCurrentL3 float64
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
	log := util.NewLogger("easee")

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Easee{
		Helper:  request.NewHelper(log),
		charger: charger,
		circuit: circuit,
		cache:   cache,
		log:     log,
		phases:  3,
	}

	if cache > 0 {
		c.log.WARN.Println("cache is deprecated and will be removed in a future release")
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

	// find site
	site, err := c.chargerDetails(c.charger)
	if err != nil {
		return c, err
	}

	c.site = site.ID

	// find circuit
	if circuit == 0 {
		if len(site.Circuits) != 1 {
			return c, fmt.Errorf("cannot determine circuit id, found: %v", funk.Map(site.Circuits, func(c easee.Circuit) int { return c.ID }))
		}

		c.circuit = site.Circuits[0].ID
	}

	err = c.subscribe(ts)
	if err != nil {
		return c, err
	}

	// verify charger config
	config, err := c.chargerConfig(c.charger)
	if err == nil && config.PhaseMode != 2 {
		c.log.WARN.Println("expected PhaseMode auto- switching phases will NOT work")
	}

	return c, err
}

// subscribe connects to the signalR hub
func (c *Easee) subscribe(ts oauth2.TokenSource) error {
	conn, err := signalr.NewHTTPConnection(context.Background(), "https://api.easee.cloud/hubs/chargers",
		signalr.WithHTTPHeadersOption(func() (res http.Header) {
			if tok, err := ts.Token(); err == nil {
				res = http.Header{
					"Authorization": []string{fmt.Sprintf("Bearer %s", tok.AccessToken)},
				}
			}
			return res
		}),
	)
	if err != nil {
		return err
	}

	client, err := signalr.NewClient(context.Background(), conn,
		signalr.Receiver(c),
		// signalr.Logger(easee.SignalrLogger(c.log.TRACE), false),
	)
	if err != nil {
		return err
	}

	if err := client.Start(); err != nil {
		return err
	}

	// retry connection
	go func(closed <-chan struct{}) {
		<-closed
		_ = retry.Do(func() error {
			err := c.subscribe(ts)
			if err != nil {
				c.log.ERROR.Println("connect:", err)
			}
			return err
		}, retry.Attempts(256))
	}(client.Closed())

	return <-client.Send("SubscribeWithCurrentState", c.charger, true)
}

func (c *Easee) observe(typ string, i json.RawMessage) {
	var res easee.Observation
	if err := json.Unmarshal(i, &res); err == nil {
		var floatValue float64
		var boolValue bool
		var intValue int
		switch res.DataType {
		case easee.Boolean:
			boolValue = res.Value == "1"
		case easee.Double:
			floatValue, err = strconv.ParseFloat(res.Value, 64)
			if err != nil {
				c.log.ERROR.Printf("float conversion: %s", res.Value)
				return
			}
		case easee.Integer:
			intValue, err = strconv.Atoi(res.Value)
			if err != nil {
				c.log.ERROR.Printf("int conversion: %s", res.Value)
				return
			}
		}

		c.mux.Lock()
		defer c.mux.Unlock()

		switch res.ID {
		case easee.IS_ENABLED:
			c.chargerEnabled = boolValue
		case easee.TOTAL_POWER:
			c.currentPower = 1e3 * floatValue
		case easee.SESSION_ENERGY:
			c.sessionEnergy = floatValue
		case easee.CIRCUIT_TOTAL_PHASE_CONDUCTOR_CURRENT_L1:
			c.circuitTotalPhaseConductorCurrentL1 = floatValue
		case easee.CIRCUIT_TOTAL_PHASE_CONDUCTOR_CURRENT_L2:
			c.circuitTotalPhaseConductorCurrentL2 = floatValue
		case easee.CIRCUIT_TOTAL_PHASE_CONDUCTOR_CURRENT_L3:
			c.circuitTotalPhaseConductorCurrentL3 = floatValue
		case easee.CHARGER_OP_MODE:
			switch intValue {
			case easee.ModeDisconnected:
				c.chargeStatus = api.StatusA
			case easee.ModeAwaitingStart, easee.ModeCompleted, easee.ModeReadyToCharge:
				c.chargeStatus = api.StatusB
			case easee.ModeCharging:
				c.chargeStatus = api.StatusC
			case easee.ModeError:
				c.chargeStatus = api.StatusF
			default:
				c.log.ERROR.Printf("unknown opmode: %d", intValue)
				c.chargeStatus = api.StatusNone
			}
			c.enabledStatus = intValue == easee.ModeCharging || intValue == easee.ModeReadyToCharge
		}
		c.updated = time.Now()
		c.log.TRACE.Printf("%s: %+v", typ, res)
	} else {
		c.log.ERROR.Printf("invalid message: %s %s %v", i, typ, err)
	}
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

func (c *Easee) chargerConfig(charger string) (res easee.ChargerConfig, err error) {
	uri := fmt.Sprintf("%s/chargers/%s/config", easee.API, charger)

	req, err := request.New(http.MethodGet, uri, nil, request.JSONEncoding)
	if err == nil {
		err = c.DoJSON(req, &res)
	}

	return res, err
}

func (c *Easee) chargerDetails(charger string) (res easee.Site, err error) {
	uri := fmt.Sprintf("%s/chargers/%s/site", easee.API, charger)

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

	return c.enabledStatus, nil
}

// Enable implements the api.Charger interface
func (c *Easee) Enable(enable bool) error {
	// enable charger once if it's switched off
	c.mux.Lock()
	enablingRequired := enable && !c.chargerEnabled
	c.mux.Unlock()
	if enablingRequired {
		data := easee.ChargerSettings{
			Enabled: &enable,
		}

		uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
		resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
		if err == nil {
			resp.Body.Close()
			return err
		}
	}

	// resume/stop charger
	action := "pause_charging"
	if enable {
		action = "resume_charging"
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
	var current23 float64
	if c.phases > 1 {
		current23 = current
	}

	data := easee.CircuitSettings{
		DynamicCircuitCurrentP1: &current,
		DynamicCircuitCurrentP2: &current23,
		DynamicCircuitCurrentP3: &current23,
	}

	uri := fmt.Sprintf("%s/sites/%d/circuits/%d/settings", easee.API, c.site, c.circuit)
	resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err == nil {
		resp.Body.Close()
		c.current = current
	}

	return err
}

var _ api.ChargePhases = (*Easee)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (c *Easee) Phases1p3p(phases int) error {
	c.phases = phases
	return c.MaxCurrentMillis(c.current)
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

	return c.circuitTotalPhaseConductorCurrentL1,
		c.circuitTotalPhaseConductorCurrentL2,
		c.circuitTotalPhaseConductorCurrentL3,
		nil
}
