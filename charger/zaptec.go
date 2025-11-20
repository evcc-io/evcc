package charger

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
	"sort"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"golang.org/x/oauth2"
)

// https://api.zaptec.com/help/index.html
// https://api.zaptec.com/.well-known/openid-configuration/

// Zaptec charger implementation
type Zaptec struct {
	*request.Helper
	log        *util.Logger
	statusG    util.Cacheable[zaptec.StateResponse]
	instance   zaptec.Charger
	maxCurrent int
	version    int
	enabled    bool
	priority   bool
	passive    bool
}

func init() {
	registry.AddCtx("zaptec", NewZaptecFromConfig)
}

// passwordTokenSource implements oauth2.TokenSource for password grant flow
type passwordTokenSource struct {
	ctx    context.Context
	config *oauth2.Config
	user   string
	pass   string
}

func (p passwordTokenSource) Token() (*oauth2.Token, error) {
	return p.config.PasswordCredentialsToken(p.ctx, p.user, p.pass)
}

//go:generate go tool decorate -f decorateZaptec -b *Zaptec -r api.Charger -t "api.PhaseSwitcher,Phases1p3p,func(int) error"

// NewZaptecFromConfig creates a Zaptec Pro charger from generic config
func NewZaptecFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		User, Password string
		Id             string
		Priority       bool
		Passive        bool
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

	return NewZaptec(ctx, cc.User, cc.Password, cc.Id, cc.Priority, cc.Passive, cc.Cache)
}

// NewZaptec creates Zaptec charger
func NewZaptec(ctx context.Context, user, password, id string, priority bool, passive bool, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("zaptec").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Zaptec{
		Helper:   request.NewHelper(log),
		log:      log,
		priority: priority,
		passive:  passive,
	}

	// setup cached values
	c.statusG = util.ResettableCached(func() (zaptec.StateResponse, error) {
		var res zaptec.StateResponse

		uri := fmt.Sprintf("%s/api/chargers/%s/state", zaptec.ApiURL, c.instance.Id)
		err := c.GetJSON(uri, &res)

		return res, err
	}, cache)

	provider, err := oidc.NewProvider(ctx, zaptec.ApiURL+"/")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OIDC provider: %s", err)
	}

	oc := &oauth2.Config{
		Endpoint: provider.Endpoint(),
		Scopes: []string{
			oidc.ScopeOpenID,
		},
	}

	// Create a separate HTTP client for OAuth token requests to avoid circular dependency
	// (c.Transport will be modified to use oauth2.Transport, which would create a loop)
	tokenClient := &http.Client{
		Transport: c.Transport,
	}

	oauthCtx := context.WithValue(
		ctx,
		oauth2.HTTPClient,
		tokenClient,
	)

	token, err := oc.PasswordCredentialsToken(oauthCtx, user, password)
	if err != nil {
		return nil, err
	}

	// Create custom token source that always uses password grant (no refresh tokens)
	ts := oauth2.ReuseTokenSource(token, passwordTokenSource{
		ctx:    oauthCtx,
		config: oc,
		user:   user,
		pass:   password,
	})

	c.Transport = &oauth2.Transport{
		Source: ts,
		Base:   c.Transport,
	}

	c.instance, err = ensureChargerEx(id, c.chargers, func(charger zaptec.Charger) (string, error) {
		return charger.Id, nil
	})
	if err != nil {
		return nil, err
	}

	c.version, err = c.detectVersion()
	if err != nil {
		return nil, err
	}

	var phases1p3p func(int) error
	if maxCurrent, err := c.getInstallationMaxCurrent(); err == nil {
		phases1p3p = c.phases1p3p
		c.maxCurrent = maxCurrent
	}

	return decorateZaptec(c, phases1p3p), nil
}

func (c *Zaptec) detectVersion() (int, error) {
	var capabilities zaptec.CapabilitiesResponse

	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	capResp := res.ObservationByID(zaptec.Capabilities)
	if err := json.Unmarshal([]byte(capResp.ValueAsString), &capabilities); err != nil {
		return 0, err
	}

	if capabilities.ProductVariant == "Go2" {
		return zaptec.ZaptecGo2, nil
	}

	return zaptec.ZaptecGo1_Pro, nil
}

func (c *Zaptec) chargers() ([]zaptec.Charger, error) {
	var res zaptec.ChargersResponse

	uri := fmt.Sprintf("%s/api/chargers", zaptec.ApiURL)
	if err := c.GetJSON(uri, &res); err != nil {
		return nil, err
	}

	return res.Data, nil
}

// Status implements the api.Charger interface
func (c *Zaptec) Status() (api.ChargeStatus, error) {
	res, err := c.statusG.Get()
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
	res, err := c.statusG.Get()
	return c.enabled && !res.ObservationByID(zaptec.FinalStopActive).Bool(), err
}

// Enable implements the api.Charger interface
func (c *Zaptec) Enable(enable bool) error {
	cmd := zaptec.CmdStopChargingFinal
	if enable {
		cmd = zaptec.CmdResumeCharging
	}

	uri := fmt.Sprintf("%s/api/chargers/%s/sendCommand/%d", zaptec.ApiURL, c.instance.Id, cmd)
	req, _ := request.New(http.MethodPost, uri, nil, request.JSONEncoding)

	var res struct {
		Code int
	}

	// ignore 528: Charging is not Paused nor Scheduled; Resume command cannot be sent
	if err := c.DoJSON(req, &res); err == nil || res.Code == 528 {
		c.enabled = enable
		c.statusG.Reset()
	}

	return nil
}

func (c *Zaptec) chargerUpdate(data zaptec.Update) error {
	if c.passive {
		if data.MaxChargeCurrent != nil || data.MinChargeCurrent != nil || data.OfflineChargeCurrent != nil {
			c.log.DEBUG.Println("zaptec: passive mode: skipping chargerUpdate with current fields set")
			return nil
		}
	}

	uri := fmt.Sprintf("%s/api/chargers/%s/update", zaptec.ApiURL, c.instance.Id)

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	_, err := c.DoBody(req)
	if err == nil {
		c.statusG.Reset()
	}

	return err
}

func (c *Zaptec) sessionPriority(session string, data zaptec.SessionPriority) error {
	uri := fmt.Sprintf("%s/api/session/%s/priority", zaptec.ApiURL, session)

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	_, err := c.DoBody(req)
	if err == nil {
		c.statusG.Reset()
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
	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePower).Float64()
}

var _ api.ChargeRater = (*Zaptec)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Zaptec) ChargedEnergy() (float64, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePowerSession).Float64()
}

var _ api.PhaseCurrents = (*Zaptec)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *Zaptec) Currents() (float64, float64, float64, error) {
	res, err := c.statusG.Get()
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

// phases1p3p implements the api.PhaseSwitcher interface
func (c *Zaptec) phases1p3p(phases int) error {
	err := c.switchPhases(phases)
	if err != nil || !c.priority {
		return err
	}

	// priority configured
	data := zaptec.SessionPriority{
		PrioritizedPhases: &phases,
	}

	res, err := c.statusG.Get()
	if err != nil {
		return err
	}

	if session := res.ObservationByID(zaptec.SessionIdentifier); session != nil {
		return c.sessionPriority(session.ValueAsString, data)
	}

	return errors.New("unknown session")
}

func (c *Zaptec) switchPhases(phases int) error {
	if c.version != zaptec.ZaptecGo2 {
		data := zaptec.Update{
			MaxChargePhases: &phases,
		}
		return c.chargerUpdate(data)
	}

	var zero int
	data := zaptec.UpdateInstallation{
		AvailableCurrentPhase1: &c.maxCurrent,
		AvailableCurrentPhase2: &zero,
		AvailableCurrentPhase3: &zero,
	}
	if phases == 3 {
		data = zaptec.UpdateInstallation{
			AvailableCurrentPhase1: &c.maxCurrent,
			AvailableCurrentPhase2: &c.maxCurrent,
			AvailableCurrentPhase3: &c.maxCurrent,
		}
	}

	return c.installationUpdate(data)
}

var _ api.Identifier = (*Zaptec)(nil)

// Identify implements the api.Identifier interface
func (c *Zaptec) Identify() (string, error) {
	res, err := c.statusG.Get()
	if err != nil {
		return "", err
	}

	if id := res.ObservationByID(zaptec.ChargerCurrentUserUuid); id != nil {
		return id.ValueAsString, nil
	}

	return "", nil
}

func (c *Zaptec) getInstallationMaxCurrent() (int, error) {
	var res zaptec.Installation

	uri := fmt.Sprintf("%s/api/installation/%s", zaptec.ApiURL, c.instance.InstallationId)
	if err := c.GetJSON(uri, &res); err != nil {
		return 0, err
	}

	return int(res.MaxCurrent), nil
}

func (c *Zaptec) installationUpdate(data zaptec.UpdateInstallation) error {
	uri := fmt.Sprintf("%s/api/installation/%s/update", zaptec.ApiURL, c.instance.InstallationId)

	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	_, err := c.DoBody(req)
	if err == nil {
		c.statusG.Reset()
	}

	return err
}

var _ api.Diagnosis = (*Zaptec)(nil)

// Diagnosis implements the api.Diagnosis interface
func (c *Zaptec) Diagnose() {
	res, _ := c.statusG.Get()

	// sort for printing
	sort.Slice(res, func(i, j int) bool {
		return res[i].StateId < res[j].StateId
	})

	for _, k := range res {
		fmt.Printf("%d. %s %s\n", k.StateId, k.StateId, k)
	}
}
