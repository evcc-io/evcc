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
	"strings"
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
	log                   *util.Logger
	mux                   sync.Mutex
	done                  chan struct{}
	dynamicChargerCurrent float64
	current               float64
	currentUpdated        time.Time
	chargerEnabled        bool
	smartCharging         bool
	opMode                int
	phaseMode             int
	currentPower, sessionEnergy, totalEnergy,
	currentL1, currentL2, currentL3 float64
	rfid     string
	lp       loadpoint.API
	respChan chan easee.SignalRCommandResponse
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
		Timeout  time.Duration
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEasee(cc.User, cc.Password, cc.Charger, cc.Timeout)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string, timeout time.Duration) (*Easee, error) {
	log := util.NewLogger("easee").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Easee{
		Helper:   request.NewHelper(log),
		charger:  charger,
		log:      log,
		current:  6, // default current
		done:     make(chan struct{}),
		respChan: make(chan easee.SignalRCommandResponse),
	}

	c.Client.Timeout = timeout

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
	select {
	case <-c.done:
	case <-time.After(request.Timeout):
		err = os.ErrDeadlineExceeded
	}

	if err == nil {
		go c.heartbeat()
	}

	return c, err
}

// heartbeat ensures tokens are refreshed even when not charging for longer time
func (c *Easee) heartbeat() {
	for range time.Tick(6 * time.Hour) {
		if _, err := c.chargerSite(c.charger); err != nil {
			c.log.ERROR.Println("heartbeat:", err)
		}
	}
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

		return signalr.NewHTTPConnection(ctx, "https://streams.easee.com/hubs/chargers",
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

// ProductUpdate implements the signalr receiver
func (c *Easee) ProductUpdate(i json.RawMessage) {
	var (
		once sync.Once
		res  easee.Observation
	)

	if err := json.Unmarshal(i, &res); err != nil {
		c.log.ERROR.Printf("invalid message: %s %v", i, err)
		return
	}

	var (
		value interface{}
		err   error
	)

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

	// https://github.com/evcc-io/evcc/issues/8009
	// logging might be slow or block, execute outside lock
	c.log.TRACE.Printf("ProductUpdate %s: %s %v", res.Mid, res.ID, value)

	c.mux.Lock()
	defer c.mux.Unlock()

	if c.updated.IsZero() {
		defer once.Do(func() {
			close(c.done)
		})
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
		if c.dynamicChargerCurrent > 0 && c.dynamicChargerCurrent != c.current &&
			time.Since(c.currentUpdated) > 10*time.Second {
			c.log.DEBUG.Printf("current mismatch, expected %.1f, got %.1f", c.current, c.dynamicChargerCurrent)
		}
	case easee.CHARGER_OP_MODE:
		c.opMode = value.(int)
	}
}

// ChargerUpdate implements the signalr receiver
func (c *Easee) ChargerUpdate(i json.RawMessage) {
	// c.observe("ChargerUpdate", i)
}

// CommandResponse implements the signalr receiver
func (c *Easee) CommandResponse(i json.RawMessage) {
	var res easee.SignalRCommandResponse

	if err := json.Unmarshal(i, &res); err != nil {
		c.log.ERROR.Printf("invalid message: %s %v", i, err)
		return
	}
	c.log.TRACE.Printf("CommandResponse %s: %+v", res.SerialNumber, res)

	select {
	case c.respChan <- res:
	default:
	}
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

	c.mux.Lock()
	defer c.mux.Unlock()

	res := api.StatusNone

	switch c.opMode {
	case easee.ModeDisconnected:
		res = api.StatusA
	case easee.ModeAwaitingStart, easee.ModeCompleted, easee.ModeReadyToCharge,
		easee.ModeAwaitingAuthentication, easee.ModeDeauthenticating:
		res = api.StatusB
	case easee.ModeCharging:
		res = api.StatusC
	default:
		return res, fmt.Errorf("invalid opmode: %d", c.opMode)
	}

	return res, nil
}

// Enabled implements the api.Charger interface
func (c *Easee) Enabled() (bool, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	enabled := c.opMode == easee.ModeCharging ||
		c.opMode == easee.ModeAwaitingStart ||
		c.opMode == easee.ModeCompleted ||
		c.opMode == easee.ModeReadyToCharge

	return enabled && c.dynamicChargerCurrent > 0, nil
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
		if err := c.postJSONAndWait(uri, data); err != nil {
			return err
		}
	}

	// resume/stop charger
	action := easee.ChargePause
	if enable {
		action = easee.ChargeResume
	}

	uri := fmt.Sprintf("%s/chargers/%s/commands/%s", easee.API, c.charger, action)
	if err := c.postJSONAndWait(uri, nil); err != nil {
		return err
	}

	if enable {
		// reset currents after enable, as easee automatically resets to maxA
		return c.MaxCurrent(int64(c.current))
	}

	return nil
}

// posts JSON to the Easee API endpoint and waits for the async response
func (c *Easee) postJSONAndWait(uri string, data any) error {
	resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 { // sync call
		return nil
	}

	if resp.StatusCode == 202 { // async call, wait for response
		var cmd easee.RestCommandResponse

		if strings.Contains(uri, "/commands/") { // command endpoint
			if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
				return err
			}
		} else { // settings endpoint
			var cmdArr []easee.RestCommandResponse
			if err := json.NewDecoder(resp.Body).Decode(&cmdArr); err != nil {
				return err
			}

			if len(cmdArr) != 0 {
				cmd = cmdArr[0]
			}
		}

		if cmd.Ticks == 0 { // api thinks this was a noop
			return nil
		}

		return c.waitForTickResponse(cmd.Ticks)
	}

	// all other response codes lead to an error
	return fmt.Errorf("invalid status: %d", resp.StatusCode)
}

func (c *Easee) waitForTickResponse(expectedTick int64) error {
	for {
		select {
		case cmdResp := <-c.respChan:
			if cmdResp.Ticks == expectedTick {
				if !cmdResp.WasAccepted {
					return fmt.Errorf("command rejected: %d", cmdResp.Ticks)
				}
				return nil
			}
		case <-time.After(10 * time.Second):
			return api.ErrTimeout
		}
	}
}

// MaxCurrent implements the api.Charger interface
func (c *Easee) MaxCurrent(current int64) error {
	cur := float64(current)
	data := easee.ChargerSettings{
		DynamicChargerCurrent: &cur,
	}

	uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
	if err := c.postJSONAndWait(uri, data); err != nil {
		return err
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	c.current = cur
	c.currentUpdated = time.Now()

	return nil
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

		err = c.postJSONAndWait(uri, data)
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

			if err = c.postJSONAndWait(uri, data); err != nil {
				return err
			}

			// disable charger to activate changed settings (loadpoint will reenable it)
			err = c.Enable(false)
		}
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

var _ api.PhaseCurrents = (*Easee)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.currentL1, c.currentL2, c.currentL3, nil
}

var _ api.MeterEnergy = (*Easee)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Easee) TotalEnergy() (float64, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.totalEnergy, nil
}

var _ api.Identifier = (*Easee)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Easee) Identify() (string, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	return c.rfid, nil
}

// Set smart charging status to update the chargers led (smart=blue, fast=white)
func (c *Easee) updateSmartCharging() {
	if c.lp == nil {
		return
	}

	mode := c.lp.GetMode()
	isSmartCharging := mode == api.ModePV || mode == api.ModeMinPV

	c.mux.Lock()
	updateNeeded := isSmartCharging != c.smartCharging
	c.mux.Unlock()

	if updateNeeded {
		data := easee.ChargerSettings{
			SmartCharging: &isSmartCharging,
		}

		uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)

		err := c.postJSONAndWait(uri, data)
		if err != nil {
			c.log.WARN.Printf("smart charging: %v", err)
			return
		}

		c.mux.Lock()
		c.smartCharging = isSmartCharging
		c.mux.Unlock()
	}
}

// LoadpointControl implements loadpoint.Controller
func (c *Easee) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}
