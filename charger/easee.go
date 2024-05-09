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
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
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
	charger                 string
	site, circuit           int
	lastEnergyPollTriggered time.Time
	lastOpModePollTriggered time.Time
	log                     *util.Logger
	mux                     sync.RWMutex
	lastEnergyPollMux       sync.Mutex
	dynamicChargerCurrent   float64
	current                 float64
	chargerEnabled          bool
	smartCharging           bool
	authorize               bool
	enabled                 bool
	opMode                  int
	pilotMode               string
	reasonForNoCurrent      int
	phaseMode               int
	outputPhase             int
	sessionStartEnergy      *float64
	currentPower, sessionEnergy, totalEnergy,
	currentL1, currentL2, currentL3 float64
	rfid       string
	lp         loadpoint.API
	cmdC       chan easee.SignalRCommandResponse
	obsC       chan easee.Observation
	obsTime    map[easee.ObservationID]time.Time
	stopTicker chan struct{}
	startDone  func()
}

func init() {
	registry.Add("easee", NewEaseeFromConfig)
}

// NewEaseeFromConfig creates a Easee charger from generic config
func NewEaseeFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User      string
		Password  string
		Charger   string
		Timeout   time.Duration
		Authorize bool
	}{
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewEasee(cc.User, cc.Password, cc.Charger, cc.Timeout, cc.Authorize)
}

// NewEasee creates Easee charger
func NewEasee(user, password, charger string, timeout time.Duration, authorize bool) (*Easee, error) {
	log := util.NewLogger("easee").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	done := make(chan struct{})

	c := &Easee{
		Helper:    request.NewHelper(log),
		charger:   charger,
		authorize: authorize,
		log:       log,
		current:   6, // default current
		startDone: sync.OnceFunc(func() { close(done) }),
		cmdC:      make(chan easee.SignalRCommandResponse),
		obsC:      make(chan easee.Observation),
		obsTime:   make(map[easee.ObservationID]time.Time),
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
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = time.Minute

	return func() (conn signalr.Connection, err error) {
		defer func() {
			if err != nil {
				time.Sleep(bo.NextBackOff())
			} else {
				bo.Reset()
			}
		}()

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
	var res easee.Observation

	if err := json.Unmarshal(i, &res); err != nil {
		c.log.ERROR.Printf("invalid message: %s %v", i, err)
		return
	}

	value, err := res.TypedValue()
	if err != nil {
		c.log.ERROR.Println(err)
		return
	}

	// https://github.com/evcc-io/evcc/issues/8009
	// logging might be slow or block, execute outside lock
	c.log.TRACE.Printf("ProductUpdate %s: (%v) %s %v", res.Mid, res.Timestamp, res.ID, value)

	c.mux.Lock()
	defer c.mux.Unlock()

	if prevTime, ok := c.obsTime[res.ID]; ok && prevTime.After(res.Timestamp) {
		// received observation is outdated, ignoring
		return
	}

	c.obsTime[res.ID] = res.Timestamp

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
		// SESSION_ENERGY must not be set to 0 by Productupdates, they occur erratic
		// Reset to 0 is done in case CHARGER_OP_MODE
		if value.(float64) != 0 {
			c.sessionEnergy = value.(float64)
		}
	case easee.LIFETIME_ENERGY:
		c.totalEnergy = value.(float64)
		if c.sessionStartEnergy == nil {
			f := c.totalEnergy
			c.sessionStartEnergy = &f
		}
	case easee.IN_CURRENT_T3:
		c.currentL1 = value.(float64)
	case easee.IN_CURRENT_T4:
		c.currentL2 = value.(float64)
	case easee.IN_CURRENT_T5:
		c.currentL3 = value.(float64)
	case easee.PHASE_MODE:
		c.phaseMode = value.(int)
	case easee.OUTPUT_PHASE:
		c.outputPhase = value.(int) / 10 // API gives 0,10,30 for 0,1,3p
	case easee.DYNAMIC_CHARGER_CURRENT:
		c.dynamicChargerCurrent = value.(float64)

	case easee.CHARGER_OP_MODE:
		opMode := value.(int)

		// New charging session pending, reset internal value of SESSION_ENERGY to 0, and its observation timestamp to "now".
		// This should be done in a proper way by the api, but it's not.
		// Remember value of LIFETIME_ENERGY as start value of the charging session
		if c.opMode <= easee.ModeDisconnected && opMode >= easee.ModeAwaitingStart {
			c.sessionEnergy = 0
			c.obsTime[easee.SESSION_ENERGY] = time.Now()
			c.sessionStartEnergy = nil
		}

		// OpMode changed TO charging. Start ticker for periodic requests to update LIFETIME_ENERGY
		if c.opMode != easee.ModeCharging && opMode == easee.ModeCharging {
			if c.stopTicker == nil {
				c.stopTicker = make(chan struct{})

				go func() {
					ticker := time.NewTicker(5 * time.Minute)
					for {
						select {
						case <-c.stopTicker:
							return
						case <-ticker.C:
							c.requestLifetimeEnergyUpdate()
						}
					}
				}()
			}
		}

		// OpMode changed FROM >1 ("car connected") TO  1/disconnected  - stop ticker if channel exists
		// channel may not exist regularly if the car was connected but charging never started
		if c.opMode != easee.ModeDisconnected && opMode == easee.ModeDisconnected && c.stopTicker != nil {
			close(c.stopTicker)
			c.stopTicker = nil
		}

		// for relevant OpModes changes indicating a start or stop of the charging session, request new update of LIFETIME_ENERGY
		// relevant OpModes: leaving op modes 1 (car connected, charging will start uncontrolled if unauthorized)
		// and 3 (charging stopped or pause), or reaching op mode 1 (car disconnected) and 7 (charging paused/ended by de-authenticating)
		if c.opMode != opMode && // only if op mode actually changed AND
			(c.opMode == easee.ModeDisconnected || c.opMode == easee.ModeCharging || // from these op modes
				opMode == easee.ModeDisconnected || opMode == easee.ModeAwaitingAuthentication) { // or to these op modes
			c.requestLifetimeEnergyUpdate()
		}

		c.opMode = opMode

		// startup completed
		c.startDone()

	case easee.REASON_FOR_NO_CURRENT:
		c.reasonForNoCurrent = value.(int)
	case easee.PILOT_MODE:
		c.pilotMode = value.(string)
	}

	select {
	case c.obsC <- res:
	default:
	}
}

// ChargerUpdate implements the signalr receiver
func (c *Easee) ChargerUpdate(i json.RawMessage) {
	c.log.TRACE.Printf("ChargerUpdate: %s", i)
}

// SubscribeToMyProduct implements the signalr receiver
func (c *Easee) SubscribeToMyProduct(i json.RawMessage) {
	c.log.TRACE.Printf("SubscribeToMyProduct: %s", i)
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
	case c.cmdC <- res:
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
	c.confirmStatusConsistency()

	c.mux.RLock()
	defer c.mux.RUnlock()

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
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.enabled, nil
}

// Enable implements the api.Charger interface
func (c *Easee) Enable(enable bool) (err error) {
	c.mux.Lock()
	enablingRequired := enable && !c.chargerEnabled
	opMode := c.opMode
	c.mux.Unlock()

	defer func() {
		if err == nil {
			c.enabled = enable
		}
	}()

	// enable charger once if it's switched off
	if enablingRequired {
		data := easee.ChargerSettings{
			Enabled: &enable,
		}

		uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)
		if _, err := c.postJSONAndWait(uri, data); err != nil {
			return err
		}
	}

	// do not send pause/resume if disconnected or unauthenticated without automatic authorization
	if opMode == easee.ModeDisconnected || (opMode == easee.ModeAwaitingAuthentication && !(enable && c.authorize)) {
		return nil
	}

	// resume/stop charger
	action := easee.ChargePause
	var targetCurrent float64
	if enable {
		action = easee.ChargeResume
		if opMode == easee.ModeAwaitingAuthentication && c.authorize {
			action = easee.ChargeStart
		}
		targetCurrent = 32
	}

	uri := fmt.Sprintf("%s/chargers/%s/commands/%s", easee.API, c.charger, action)
	if _, err := c.postJSONAndWait(uri, nil); err != nil {
		return err
	}

	if err := c.waitForChargerEnabledState(enable); err != nil {
		return err
	}

	if action == easee.ChargeStart { // ChargeStart does not mingle with DCC, no need for below operations
		return nil
	}

	if err := c.waitForDynamicChargerCurrent(targetCurrent); err != nil {
		return err
	}

	if enable {
		// reset currents after enable, as easee automatically resets to maxA
		return c.MaxCurrent(int64(c.current))
	}

	return nil
}

func (c *Easee) inExpectedOpMode(enable bool) bool {
	c.mux.RLock()
	defer c.mux.RUnlock()

	// start/resume
	if enable {
		return c.opMode == easee.ModeCharging ||
			c.opMode == easee.ModeCompleted ||
			c.opMode == easee.ModeAwaitingStart ||
			c.opMode == easee.ModeReadyToCharge
	}

	// paused/stopped
	return c.opMode == easee.ModeAwaitingStart || c.opMode == easee.ModeAwaitingAuthentication
}

// posts JSON to the Easee API endpoint and waits for the async response
func (c *Easee) postJSONAndWait(uri string, data any) (bool, error) {
	resp, err := c.Post(uri, request.JSONContent, request.MarshalJSON(data))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 { // sync call
		return false, nil
	}

	if resp.StatusCode == 202 { // async call, wait for response
		var cmd easee.RestCommandResponse

		if strings.Contains(uri, "/commands/") { // command endpoint
			if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
				return false, err
			}
		} else { // settings endpoint
			var cmdArr []easee.RestCommandResponse
			if err := json.NewDecoder(resp.Body).Decode(&cmdArr); err != nil {
				return false, err
			}

			if len(cmdArr) != 0 {
				cmd = cmdArr[0]
			}
		}

		if cmd.Ticks == 0 { // api thinks this was a noop
			return true, nil
		}

		return false, c.waitForTickResponse(cmd.Ticks)
	}

	// all other response codes lead to an error
	return false, fmt.Errorf("invalid status: %d", resp.StatusCode)
}

func (c *Easee) waitForTickResponse(expectedTick int64) error {
	for {
		select {
		case cmdResp := <-c.cmdC:
			if cmdResp.Ticks == expectedTick {
				if !cmdResp.WasAccepted {
					return fmt.Errorf("command rejected: %d", cmdResp.Ticks)
				}
				return nil
			}
		case <-time.After(c.Client.Timeout):
			return api.ErrTimeout
		}
	}
}

// wait for opMode become expected op mode
func (c *Easee) waitForChargerEnabledState(expEnabled bool) error {
	// check any updates received meanwhile
	if c.inExpectedOpMode(expEnabled) {
		return nil
	}

	timer := time.NewTimer(c.Client.Timeout)
	for {
		select {
		case obs := <-c.obsC:
			if obs.ID != easee.CHARGER_OP_MODE {
				continue
			}
			if c.inExpectedOpMode(expEnabled) {
				return nil
			}
		case <-timer.C: // time is up, bail after one final check
			if c.inExpectedOpMode(expEnabled) {
				return nil
			}
			return api.ErrTimeout
		}
	}
}

// wait for current become targetCurrent
func (c *Easee) waitForDynamicChargerCurrent(targetCurrent float64) error {
	// check any updates received meanwhile
	c.mux.RLock()
	if c.dynamicChargerCurrent == targetCurrent {
		c.mux.RUnlock()
		return nil
	}
	c.mux.RUnlock()

	timer := time.NewTimer(c.Client.Timeout)
	for {
		select {
		case obs := <-c.obsC:
			if obs.ID != easee.DYNAMIC_CHARGER_CURRENT {
				continue
			}
			value, err := obs.TypedValue()
			if err != nil {
				continue
			}
			if value.(float64) == targetCurrent {
				return nil
			}
		case <-timer.C: // time is up, bail after one final check
			c.mux.RLock()
			defer c.mux.RUnlock()
			if c.dynamicChargerCurrent == targetCurrent {
				return nil
			}
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
	noop, err := c.postJSONAndWait(uri, data)
	if err != nil {
		return err
	}

	if !noop {
		if err := c.waitForDynamicChargerCurrent(float64(current)); err != nil {
			return err
		}
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	c.current = cur

	return nil
}

var _ api.CurrentGetter = (*Easee)(nil)

// GetMaxCurrent implements the api.CurrentGetter interface
func (c *Easee) GetMaxCurrent() (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.dynamicChargerCurrent, nil
}

var _ api.Meter = (*Easee)(nil)

// CurrentPower implements the api.Meter interface
func (c *Easee) CurrentPower() (float64, error) {
	if status, err := c.Status(); err != nil || status == api.StatusA {
		return 0, err
	}

	c.mux.RLock()
	defer c.mux.RUnlock()

	return c.currentPower, nil
}

func (c *Easee) requestLifetimeEnergyUpdate() {
	c.lastEnergyPollMux.Lock()
	defer c.lastEnergyPollMux.Unlock()
	if time.Since(c.lastEnergyPollTriggered) > time.Minute*3 { // api rate limit, max once in 3 minutes
		uri := fmt.Sprintf("%s/chargers/%s/commands/%s", easee.API, c.charger, easee.PollLifetimeEnergy)
		if _, err := c.Post(uri, request.JSONContent, request.MarshalJSON(nil)); err != nil {
			c.log.WARN.Printf("Failed to trigger an update of LIFETIME_ENERGY: %v", err)
		}
		c.lastEnergyPollTriggered = time.Now()
	}
}

var _ api.ChargeRater = (*Easee)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Easee) ChargedEnergy() (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	// return either the self calced session energy (current LIFETIME_ENERGY minus remembered start value),
	// or the SESSION_ENERGY value by the API. Each value could be lower than the other, depending on
	// order and receive timestamp of the product update. We want to return the higher (and newer) value.
	if c.sessionStartEnergy != nil {
		return max(c.sessionEnergy, c.totalEnergy-*c.sessionStartEnergy), nil
	}

	return c.sessionEnergy, nil
}

var _ api.PhaseCurrents = (*Easee)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.currentL1, c.currentL2, c.currentL3, nil
}

var _ api.MeterEnergy = (*Easee)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *Easee) TotalEnergy() (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.totalEnergy, nil
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

		_, err = c.postJSONAndWait(uri, data)
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

			if _, err = c.postJSONAndWait(uri, data); err != nil {
				return err
			}

			// disable charger to activate changed settings (loadpoint will reenable it)
			err = c.Enable(false)
		}
	}

	return err
}

var _ api.PhaseGetter = (*Easee)(nil)

// GetPhases implements the api.PhaseGetter interface
func (c *Easee) GetPhases() (int, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	return c.outputPhase, nil
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
	updateNeeded := c.opMode != easee.ModeDisconnected && isSmartCharging != c.smartCharging
	c.mux.Unlock()

	if updateNeeded {
		data := easee.ChargerSettings{
			SmartCharging: &isSmartCharging,
		}

		uri := fmt.Sprintf("%s/chargers/%s/settings", easee.API, c.charger)

		if _, err := c.postJSONAndWait(uri, data); err != nil {
			c.log.WARN.Printf("smart charging: %v", err)
			return
		}

		c.mux.Lock()
		c.smartCharging = isSmartCharging
		c.mux.Unlock()
	}
}

var _ loadpoint.Controller = (*Easee)(nil)

// LoadpointControl implements loadpoint.Controller
func (c *Easee) LoadpointControl(lp loadpoint.API) {
	c.lp = lp
}

// checks that opMode matches powerflow and polls if inconsistent
func (c *Easee) confirmStatusConsistency() {
	c.mux.Lock()
	opCharging := c.opMode == easee.ModeCharging
	pilotCharging := c.pilotMode == "C"
	powerFlowing := c.currentPower > 0
	c.mux.Unlock()

	if (!opCharging && powerFlowing) || opCharging != pilotCharging {
		// poll opMode from charger as API can give outdated data after SignalR (re)connect
		if time.Since(c.lastOpModePollTriggered) > time.Minute*3 { // api rate limit, max once in 3 minutes
			uri := fmt.Sprintf("%s/chargers/%s/commands/poll_chargeropmode", easee.API, c.charger)
			if _, err := c.Post(uri, request.JSONContent, nil); err != nil {
				c.log.WARN.Printf("failed to poll CHARGER_OP_MODE, results may vary: %v", err)
			}
			c.lastOpModePollTriggered = time.Now()
		}
	}
}
