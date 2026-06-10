package meter

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

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
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/easee"
	easeeMeter "github.com/evcc-io/evcc/meter/easee"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/philippseith/signalr"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// observationTimeout is the maximum age of the most recently received meter
// observation before power and current readings are considered stale and zeroed.
const observationTimeout = 1 * time.Minute

// Easee meter implementation
type Easee struct {
	*request.Helper
	meter                           string
	site                            int
	log                             *util.Logger
	mux                             sync.RWMutex
	importPower                     float64
	exportPower                     float64
	currentPower                    float64
	totalEnergyValue                float64
	returnEnergyValue               float64
	currentL1, currentL2, currentL3 float64
	voltageL1, voltageL2, voltageL3 float64
	maxACPower                      float64
	siteStructure                   easeeMeter.SiteStructure

	dispatcher *easee.CommandDispatcher

	obsC            chan easeeMeter.Observation
	obsTime         map[easeeMeter.ObservationID]time.Time
	lastObsReceived time.Time
	startDone       func()
}

func init() {
	registry.AddCtx("easee", NewEaseeFromConfig)
}

// NewEaseeFromConfig creates an Easee meter from generic config
func NewEaseeFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	cc := struct {
		User     string
		Password string
		Meter    string
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

	return NewEasee(ctx, cc.User, cc.Password, cc.Meter, cc.Timeout)
}

// NewEasee creates Easee Meter
func NewEasee(ctx context.Context, user, password, meter string, timeout time.Duration) (api.Meter, error) {
	log := util.NewLogger("easee").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	done := make(chan struct{})

	c := &Easee{
		Helper:    request.NewHelper(log),
		meter:     meter,
		log:       log,
		startDone: sync.OnceFunc(func() { close(done) }),
		obsC:      make(chan easeeMeter.Observation),
		obsTime:   make(map[easeeMeter.ObservationID]time.Time),
	}

	c.Client.Timeout = timeout

	c.dispatcher = easee.NewCommandDispatcher(c.Helper, log, timeout)

	ts, err := easee.TokenSource(log, user, password)
	if err != nil {
		return nil, err
	}

	// replace client transport with authenticated transport
	c.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   c.Client.Transport,
	}

	// find meter
	if meter == "" {
		return nil, errors.New("missing equalizer serial number")
	} else {
		// find site to validate serial number
		site, err := c.meterSite(c.meter)
		if err != nil {
			return nil, err
		}

		c.site = site.ID
	}

	client, err := signalr.NewClient(ctx,
		signalr.WithConnector(c.connect(ts)),
		signalr.WithBackoff(func() backoff.BackOff {
			return backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(0)) // prevents SignalR stack to silently give up after 15 mins
		}),
		signalr.WithReceiver(c),
		signalr.Logger(easee.SignalrLogger(c.log.TRACE), false),
	)

	if err == nil {
		c.subscribe(client)

		client.Start()

		connCtx, cancel := context.WithTimeout(ctx, request.Timeout)
		defer cancel()
		err = <-client.WaitForState(connCtx, signalr.ClientConnected)
	}

	if err == nil {
		select {
		case <-done:
		case <-ctx.Done():
			err = ctx.Err()
		case <-time.After(request.Timeout):
			err = os.ErrDeadlineExceeded
		}
	}

	if err == nil {
		c.waitForOptionalState()
	}

	return c, err
}

func (c *Easee) waitForOptionalState() {
	for range 30 {
		if c.optionalStatePresent() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	c.log.WARN.Println("did not receive full state from cloud")
}

// check c.obsTime for presence of ALL of the following keys: easeeMeter.ACTIVE_POWER_IMPORT, easeeMeter.CUMULATIVE_ACTIVE_POWER_IMPORT, easeeMeter.CUMULATIVE_ACTIVE_POWER_EXPORT
func (c *Easee) optionalStatePresent() bool {
	c.mux.Lock()
	defer c.mux.Unlock()
	wanted := []easeeMeter.ObservationID{easeeMeter.ACTIVE_POWER_IMPORT, easeeMeter.CUMULATIVE_ACTIVE_POWER_IMPORT, easeeMeter.CUMULATIVE_ACTIVE_POWER_EXPORT}
	return len(wanted) == len(lo.Intersect(wanted, lo.Keys(c.obsTime)))
}

func (c *Easee) meterSite(meter string) (easee.Site, error) {
	var res easee.Site
	c.log.INFO.Printf("looking for site of meter %s", meter)
	uri := fmt.Sprintf("%s/equalizers/%s/site", easee.API, meter)
	err := c.GetJSON(uri, &res)
	return res, err
}

// connect creates an HTTP connection to the signalR hub
func (c *Easee) connect(ts oauth2.TokenSource) func() (signalr.Connection, error) {
	bo := backoff.NewExponentialBackOff(backoff.WithMaxInterval(time.Minute), backoff.WithMaxElapsedTime(0))

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
			signalr.WithHTTPHeaders(func() http.Header {
				return http.Header{
					"Authorization": []string{"Bearer " + tok.AccessToken},
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
				if err := <-client.Send("SubscribeWithCurrentState", c.meter, true); err != nil {
					c.log.ERROR.Printf("SubscribeWithCurrentState: %v", err)
				}
			}
		}
	}()
}

// ProductUpdate implements the signalr receiver
func (c *Easee) ProductUpdate(i json.RawMessage) {
	var res easeeMeter.Observation

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

	// Update liveness timestamp for observations with a fresh charger-side timestamp.
	// Stale cloud replay on restart has old timestamps and must not refresh this.
	if time.Since(res.Timestamp) < observationTimeout {
		c.lastObsReceived = time.Now()
		c.startDone()
	}

	switch res.ID {
	case easeeMeter.ACTIVE_POWER_IMPORT:
		c.importPower = 1e3 * value.(float64)
		c.currentPower = c.importPower - c.exportPower
	case easeeMeter.ACTIVE_POWER_EXPORT:
		c.exportPower = 1e3 * value.(float64)
		c.currentPower = c.importPower - c.exportPower
	case easeeMeter.CUMULATIVE_ACTIVE_POWER_IMPORT:
		c.totalEnergyValue = value.(float64)
	case easeeMeter.CUMULATIVE_ACTIVE_POWER_EXPORT:
		c.returnEnergyValue = value.(float64)
	case easeeMeter.CURRENT_L1:
		c.currentL1 = value.(float64)
	case easeeMeter.CURRENT_L2:
		c.currentL2 = value.(float64)
	case easeeMeter.CURRENT_L3:
		c.currentL3 = value.(float64)
	case easeeMeter.VOLTAGE_L1_L2:
		c.voltageL1 = value.(float64)
	case easeeMeter.VOLTAGE_L1_L3:
		c.voltageL2 = value.(float64)
	case easeeMeter.VOLTAGE_L2_L3:
		c.voltageL3 = value.(float64)
	case easeeMeter.SITE_STRUCTURE:
		var structureBytes []byte

		switch v := value.(type) {
		case string:
			structureBytes = []byte(v)
		default:
			structureBytes, err = json.Marshal(v)
			if err != nil {
				c.log.ERROR.Printf("invalid site structure: %v", err)
				return
			}
		}

		if err := json.Unmarshal(structureBytes, &c.siteStructure); err != nil {
			// Some payloads are double-encoded and arrive as a JSON string containing
			// an object. Try one additional unquote+unmarshal pass.
			var wrapped string
			if err2 := json.Unmarshal(structureBytes, &wrapped); err2 != nil {
				c.log.ERROR.Printf("invalid site structure: %v", err)
				return
			}
			if err2 := json.Unmarshal([]byte(wrapped), &c.siteStructure); err2 != nil {
				c.log.ERROR.Printf("invalid site structure: %v", err2)
				return
			}
		}
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
	c.dispatcher.Dispatch(res)
}

// CurrentPower implements the api.Meter interface
func (c *Easee) CurrentPower() (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if time.Since(c.lastObsReceived) > observationTimeout {
		return 0, nil
	}

	return c.currentPower, nil
}

// totalEnergy implements the api.MeterEnergy interface
func (c *Easee) TotalEnergy() (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if time.Since(c.lastObsReceived) > observationTimeout {
		return 0, nil
	}

	return c.totalEnergyValue, nil
}

// returnEnergy implements the api.MeterReturnEnergy interface
func (c *Easee) ReturnEnergy() (float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if time.Since(c.lastObsReceived) > observationTimeout {
		return 0, nil
	}

	return c.returnEnergyValue, nil
}

// currents implements the api.PhaseCurrents interface
func (c *Easee) Currents() (float64, float64, float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if time.Since(c.lastObsReceived) > observationTimeout {
		return 0, 0, 0, nil
	}

	return c.currentL1, c.currentL2, c.currentL3, nil
}

// voltages implements the api.PhaseVoltages interface
func (c *Easee) Voltages() (float64, float64, float64, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if time.Since(c.lastObsReceived) > observationTimeout {
		return 0, 0, 0, nil
	}

	return c.voltageL1, c.voltageL2, c.voltageL3, nil
}

// MaxACPower implements the api.MaxACPowerGetter interface
func (c *Easee) MaxACPower() float64 {
	c.mux.RLock()
	defer c.mux.RUnlock()

	if time.Since(c.lastObsReceived) > observationTimeout {
		return 0
	}

	return c.siteStructure.MaxAllocatedCurrent * 230 // API returns A, convert to W
}
