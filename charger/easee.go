package charger

// LICENSE

// Copyright (c) 2019-2022 andig

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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/philippseith/signalr"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// Easee charger implementation
type Easee struct {
	*request.Helper
	charger               string
	site, circuit         int
	updated               time.Time
	chargeStatus          api.ChargeStatus
	log                   *util.Logger
	mux                   *sync.Cond
	dynamicChargerCurrent float64
	current               float64
	chargerEnabled        bool
	smartCharging         bool
	enabledStatus         bool
	phaseMode             int
	currentPower, sessionEnergy, totalEnergy,
	currentL1, currentL2, currentL3 float64
	rfid string
	lp   loadpoint.API
}

func init() {
	registry.Add("easee", NewEaseeFromConfig)
}

// NewEaseeFromConfig creates a go-e charger from generic config
func NewEaseeFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		User     string
		Password string
		Charger  string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEasee(cc.User, cc.Password, cc.Charger)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string) (*Easee, error) {
	log := util.NewLogger("easee").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Easee{
		Helper:  request.NewHelper(log),
		charger: charger,
		log:     log,
		mux:     sync.NewCond(new(sync.Mutex)),
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
			return c, fmt.Errorf("cannot determine charger id, found: %v", lo.Map(chargers, func(c easee.Charger, _ int) string { return c.ID }))
		}

		c.charger = chargers[0].ID
	}

	// find site
	site, err := c.chargerSite(c.charger)
	if err != nil {
		return nil, err
	}

	// find single charger per circuit
	for _, circuit := range site.Circuits {
		if len(circuit.Chargers) > 1 {
			continue
		}

		for _, charger := range circuit.Chargers {
			if charger.ID == c.charger {
				c.site = site.ID
				c.circuit = circuit.ID
				break
			}
		}
	}

	client, err := signalr.NewClient(context.Background(),
		signalr.WithConnector(c.connect(ts)),
		signalr.WithReceiver(c),
		signalr.Logger(easee.SignalrLogger(c.log.TRACE), false),
	)

	if err == nil {
		c.subscribe(client)

		client.Start()

		ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
		defer cancel()
		err = <-client.WaitForState(ctx, signalr.ClientConnected)
	}

	// wait for first update
	done := make(chan struct{})
	go c.waitForInitialUpdate(done)

	select {
	case <-done:
	case <-time.After(request.Timeout):
		err = os.ErrDeadlineExceeded
	}

	return c, err
}

func (c *Easee) chargerSite(charger string) (easee.Site, error) {
	var res easee.Site
	uri := fmt.Sprintf("%s/chargers/%s/site", easee.API, charger)
	err := c.GetJSON(uri, &res)
	return res, err
}

// connect creates an HTTP connection to the signalR hub
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

// subscribe listen to state changes and sends subscription requests when connection is established
func (c *Easee) subscribe(client signalr.Client) {
	stateC := make(chan signalr.ClientState, 1)
	_ = client.ObserveStateChanged(stateC)

	go func() {
		for state := range stateC {
			if state == signalr.ClientConnected {
				if err := <-client.Send("SubscribeWithCurrentState", c.charger, true); err != nil {
					c.log.ERROR.Printf("SubscribeWithCurrentState: %v", err)
				}
			}
		}
	}()
}

// waitForInitialUpdate waits for observe to trigger the the timestamp updated condition
func (c *Easee) waitForInitialUpdate(done chan struct{}) {
	c.mux.L.Lock()
	c.mux.Wait()
	for c.updated.IsZero() {
		c.mux.Wait()
	}
	c.mux.L.Unlock()
	close(done)
}

// observe handles the subscription messages
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
	case easee.String:
		value = res.Value
	}

	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	if c.updated.IsZero() {
		go func() {
			<-time.After(3 * time.Second)
			c.mux.Broadcast()
		}()
	}
	c.updated = time.Now()

	switch res.ID {
	case easee.USER_IDTOKEN:
		c.rfid = res.Value
	case easee.IS_ENABLED:
		c.chargerEnabled = value.(bool)
	case easee.SMART_CHARGING:
		c.smartCharging = value.(bool)
	case easee.TOTAL_POWER:
		c.currentPower = 1e3 * value.(float64)
	case easee.SESSION_ENERGY:
		c.sessionEnergy = value.(float64)
	case easee.LIFETIME_ENERGY:
		c.totalEnergy = value.(float64)
	case easee.IN_CURRENT_T3:
		c.currentL1 = value.(float64)
	case easee.IN_CURRENT_T4:
		c.currentL2 = value.(float64)
	case easee.IN_CURRENT_T5:
		c.currentL3 = value.(float64)
	case easee.PHASE_MODE:
		c.phaseMode = value.(int)
	case easee.DYNAMIC_CHARGER_CURRENT:
		c.dynamicChargerCurrent = value.(float64)
		// ensure that charger current matches evcc's expectation
		if c.dynamicChargerCurrent > 0 && c.dynamicChargerCurrent != c.current {
			if err = c.MaxCurrent(int64(c.current)); err != nil {
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

	c.log.TRACE.Printf("%s %s: %s %v", typ, res.Mid, res.ID, value)
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

func (c *Easee) chargers() ([]easee.Charger, error) {
	var res []easee.Charger
	uri := fmt.Sprintf("%s/chargers", easee.API)
	err := c.GetJSON(uri, &res)
	return res, err
}

// Status implements the api.Charger interface
func (c *Easee) Status() (api.ChargeStatus, error) {
	c.updateSmartCharging()

	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.chargeStatus, nil
}

// Enabled implements the api.Charger interface
func (c *Easee) Enabled() (bool, error) {
	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.enabledStatus && c.dynamicChargerCurrent > 0, nil
}

// Enable implements the api.Charger interface
func (c *Easee) Enable(enable bool) error {
	c.mux.L.Lock()
	enablingRequired := enable && !c.chargerEnabled
	c.mux.L.Unlock()

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
	cur := float64(current)
	data := easee.ChargerSettings{
		DynamicChargerCurrent: &cur,
	}

	uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
	resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err == nil {
		c.current = cur
		resp.Body.Close()
	}

	return err
}

var _ api.PhaseSwitcher = (*Easee)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (c *Easee) Phases1p3p(phases int) error {
	var err error
	if c.circuit != 0 {
		// circuit level
		uri := fmt.Sprintf("%s/sites/%d/circuits/%d/settings", easee.API, c.site, c.circuit)

		var res easee.CircuitSettings
		if err := c.GetJSON(uri, &res); err != nil {
			return err
		}

		if res.MaxCircuitCurrentP1 == nil || res.MaxCircuitCurrentP2 == nil || res.MaxCircuitCurrentP3 == nil {
			return errors.New("MaxCircuitCurrent must not be nil")
		}

		var zero float64
		max1 := *res.MaxCircuitCurrentP1
		max2 := *res.MaxCircuitCurrentP2
		max3 := *res.MaxCircuitCurrentP3

		data := easee.CircuitSettings{
			DynamicCircuitCurrentP1: &max1,
			DynamicCircuitCurrentP2: &zero,
			DynamicCircuitCurrentP3: &zero,
		}

		if phases > 1 {
			data.DynamicCircuitCurrentP2 = &max2
			data.DynamicCircuitCurrentP3 = &max3
		}

		var resp *http.Response
		if resp, err = c.Post(uri, request.JSONContent, request.MarshalJSON(data)); err == nil {
			resp.Body.Close()
		}
	} else {
		// charger level
		if phases == 3 {
			phases = 2 // mode 2 means 3p
		}

		// change phaseMode only if necessary
		if phases != c.phaseMode {
			data := easee.ChargerSettings{
				PhaseMode: &phases,
			}

			uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)

			var resp *http.Response
			if resp, err = c.Post(uri, request.JSONContent, request.MarshalJSON(data)); err == nil {
				resp.Body.Close()
			}
		}
	}

	return err
}

var _ api.Meter = (*Easee)(nil)

// CurrentPower implements the api.Meter interface
func (c *Easee) CurrentPower() (float64, error) {
	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.currentPower, nil
}

var _ api.ChargeRater = (*Easee)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Easee) ChargedEnergy() (float64, error) {
	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.sessionEnergy, nil
}

var _ api.PhaseCurrents = (*Easee)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.currentL1, c.currentL2, c.currentL3, nil
}

var _ api.MeterEnergy = (*Easee)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Easee) TotalEnergy() (float64, error) {
	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.totalEnergy, nil
}

var _ api.Identifier = (*Easee)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Easee) Identify() (string, error) {
	c.mux.L.Lock()
	defer c.mux.L.Unlock()

	return c.rfid, nil
}

// Set smart charging status to update the chargers led (smart=blue, fast=white)
func (c *Easee) updateSmartCharging() {
	if c.lp == nil {
		return
	}

	mode := c.lp.GetMode()
	isSmartCharging := mode == api.ModePV || mode == api.ModeMinPV

	c.mux.L.Lock()
	updateNeeded := isSmartCharging != c.smartCharging
	c.mux.L.Unlock()

	if updateNeeded {
		data := easee.ChargerSettings{
			SmartCharging: &isSmartCharging,
		}

		uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
		req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
		if err == nil {
			_, err = c.DoBody(req)
		}
		if err != nil {
			c.log.WARN.Printf("smart charging: %v", err)
		}

		c.mux.L.Lock()
		c.smartCharging = isSmartCharging
		c.mux.L.Unlock()
	}
}

// LoadpointControl implements loadpoint.Controller
func (c *Easee) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
