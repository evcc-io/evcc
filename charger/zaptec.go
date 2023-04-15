package charger

// LICENSE

// Copyright (c) 2022 andig

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
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// https://api.zaptec.com/help/index.html
// https://api.zaptec.com/.well-known/openid-configuration/

// Zaptec charger implementation
type Zaptec struct {
	*request.Helper
	log         *util.Logger
	statusCache provider.Cacheable[zaptec.StateResponse]
	id          string
	enabled     bool
	priority    bool
	cache       time.Duration
}

func init() {
	registry.Add("zaptec", NewZaptecFromConfig)
}

// NewZaptecFromConfig creates a Zaptec Pro charger from generic config
func NewZaptecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User, Password string
		Id             string
		Priority       bool
		Cache          time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewZaptec(cc.User, cc.Password, cc.Id, cc.Priority, cc.Cache)
}

// NewZaptec creates Zaptec charger
func NewZaptec(user, password, id string, priority bool, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("zaptec").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Zaptec{
		Helper:   request.NewHelper(log),
		log:      log,
		id:       id,
		priority: priority,
		cache:    cache,
	}

	// setup cached values
	c.statusCache = provider.ResettableCached(func() (zaptec.StateResponse, error) {
		var res zaptec.StateResponse

		uri := fmt.Sprintf("%s/api/chargers/%s/state", zaptec.ApiURL, c.id)
		err := c.GetJSON(uri, &res)

		return res, err
	}, c.cache)

	provider, err := oidc.NewProvider(context.Background(), zaptec.ApiURL+"/")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %s", err)
	}

	oc := &oauth2.Config{
		Endpoint: provider.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID,
			oidc.ScopeOfflineAccess,
		},
	}

	ctx := context.WithValue(
		context.Background(),
		oauth2.HTTPClient,
		c.Client,
	)

	token, err := oc.PasswordCredentialsToken(ctx, user, password)

	if err == nil {
		c.Transport = &oauth2.Transport{
			Source: oc.TokenSource(context.Background(), token),
			Base:   c.Transport,
		}
	}

	if err == nil {
		c.id, err = ensureCharger(c.id, c.chargers)
	}

	return c, err
}

func (c *Zaptec) chargers() ([]string, error) {
	var res zaptec.ChargersResponse

	uri := fmt.Sprintf("%s/api/chargers", zaptec.ApiURL)
	err := c.GetJSON(uri, &res)
	if err == nil {
		return lo.Map(res.Data, func(c zaptec.Charger, _ int) string {
			return c.Id
		}), nil
	}

	return nil, err
}

// Status implements the api.Charger interface
func (c *Zaptec) Status() (api.ChargeStatus, error) {
	res, err := c.statusCache.Get()
	if err != nil {
		return api.StatusA, err
	}

	switch i, err := res.ObservationByID(zaptec.ChargerOperationMode).Int(); i {
	case zaptec.OpModeDisconnected:
		return api.StatusA, err
	case zaptec.OpModeConnectedRequesting, zaptec.OpModeConnectedFinished:
		return api.StatusB, err
	case zaptec.OpModeConnectedCharging:
		return api.StatusC, err
	default:
		if err == nil {
			err = fmt.Errorf("unknown status: %d", i)
		}
		return api.StatusNone, err
	}
}

// Enabled implements the api.Charger interface
func (c *Zaptec) Enabled() (bool, error) {
	res, err := c.statusCache.Get()
	return c.enabled && !res.ObservationByID(zaptec.FinalStopActive).Bool(), err
}

// Enable implements the api.Charger interface
func (c *Zaptec) Enable(enable bool) error {
	cmd := zaptec.CmdStopChargingFinal
	if enable {
		cmd = zaptec.CmdResumeCharging
	}

	uri := fmt.Sprintf("%s/api/chargers/%s/sendCommand/%d", zaptec.ApiURL, c.id, cmd)

	req, err := request.New(http.MethodPost, uri, nil, request.JSONEncoding)
	if err == nil {
		_, err = c.DoBody(req)
		c.enabled = enable
		c.statusCache.Reset()
	}

	return err
}

func (c *Zaptec) chargerUpdate(data zaptec.Update) error {
	uri := fmt.Sprintf("%s/api/chargers/%s/update", zaptec.ApiURL, c.id)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = c.DoBody(req)
		c.statusCache.Reset()
	}

	return err
}

func (c *Zaptec) sessionPriority(session string, data zaptec.SessionPriority) error {
	uri := fmt.Sprintf("%s/api/session/%s/priority", zaptec.ApiURL, session)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = c.DoBody(req)
		c.statusCache.Reset()
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Zaptec) MaxCurrent(current int64) error {
	curr := int(current)
	data := zaptec.Update{
		MaxChargeCurrent: &curr,
	}

	return c.chargerUpdate(data)
}

var _ api.Meter = (*Zaptec)(nil)

// CurrentPower implements the api.Meter interface
func (c *Zaptec) CurrentPower() (float64, error) {
	res, err := c.statusCache.Get()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePower).Float64()
}

var _ api.ChargeRater = (*Zaptec)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Zaptec) ChargedEnergy() (float64, error) {
	res, err := c.statusCache.Get()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePowerSession).Float64()
}

var _ api.PhaseCurrents = (*Zaptec)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Zaptec) Currents() (float64, float64, float64, error) {
	res, err := c.statusCache.Get()
	if err != nil {
		return 0, 0, 0, err
	}

	var f [3]float64
	for i, l := range []zaptec.ObservationID{zaptec.CurrentPhase1, zaptec.CurrentPhase2, zaptec.CurrentPhase3} {
		if f[i], err = res.ObservationByID(l).Float64(); err != nil {
			break
		}
	}

	return f[0], f[1], f[2], err
}

var _ api.PhaseSwitcher = (*Zaptec)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (c *Zaptec) Phases1p3p(phases int) error {
	res, err := c.statusCache.Get()

	if err == nil {
		data := zaptec.Update{
			MaxChargePhases: &phases,
		}

		err = c.chargerUpdate(data)
	}

	if err == nil && c.priority {
		data := zaptec.SessionPriority{
			PrioritizedPhases: &phases,
		}

		if session := res.ObservationByID(zaptec.SessionIdentifier); session != nil {
			err = c.sessionPriority(session.ValueAsString, data)
		} else {
			err = errors.New("unknown session")
		}
	}

	return err
}

var _ api.Diagnosis = (*Zaptec)(nil)

// Diagnosis implements the api.ChargePhases interface
func (c *Zaptec) Diagnose() {
	res, _ := c.statusCache.Get()

	// sort for printing
	sort.Slice(res, func(i, j int) bool {
		return res[i].StateId < res[j].StateId
	})

	for _, k := range res {
		fmt.Printf("%d. %s %s\n", k.StateId, k.StateId, k)
	}
}
