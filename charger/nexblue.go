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

// https://prod-management.nexblue.com/swagger/dist/index.html

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const nexblueAPI = "https://api.nexblue.com/third_party/openapi"

// Nexblue charger implementation
type Nexblue struct {
	*request.Helper
	serial  string
	enabled bool
	statusG util.Cacheable[nexblueStatus]
}

type nexblueStatus struct {
	ChargingState  int       `json:"charging_state"`
	Power          float64   `json:"power"`           // kW
	Energy         float64   `json:"energy"`          // kWh (session)
	LifetimeEnergy float64   `json:"lifetime_energy"` // kWh (total)
	CurrentLimit   int       `json:"current_limit"`   // A
	VoltageList    []float64 `json:"voltage_list"`    // V per phase
}

func init() {
	registry.AddCtx("nexblue", NewNexblueFromConfig)
}

// NewNexblueFromConfig creates a Nexblue charger from generic config
func NewNexblueFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		User     string
		Password string
		Serial   string
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	return NewNexblue(ctx, cc.User, cc.Password, cc.Serial, cc.Cache)
}

// NewNexblue creates Nexblue charger
func NewNexblue(ctx context.Context, user, password, serial string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("nexblue").Redact(user, password)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	wb := &Nexblue{
		Helper: request.NewHelper(log),
	}

	// authHelper uses a separate client injected via context to avoid circular
	// dependency when oauth2.Transport later calls Token() on token refresh.
	authHelper := request.NewHelper(log)
	login := func(_ *oauth2.Token) (*oauth2.Token, error) {
		req, _ := request.New(http.MethodPost, nexblueAPI+"/account/login", request.MarshalJSON(struct {
			Username    string `json:"username"`
			Password    string `json:"password"`
			AccountType int    `json:"account_type"`
		}{
			user, password, 0,
		}), request.JSONEncoding)

		var res struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		}
		if err := authHelper.DoJSON(req, &res); err != nil {
			return nil, err
		}

		return &oauth2.Token{
			AccessToken: res.AccessToken,
			Expiry:      time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		}, nil
	}

	tok, err := login(nil)
	if err != nil {
		return nil, err
	}

	// Inject authHelper's client as base transport; oauth2.NewClient wraps it with oauth2.Transport.
	authCtx := context.WithValue(ctx, oauth2.HTTPClient, authHelper.Client)
	wb.Client = oauth2.NewClient(authCtx, oauth.RefreshTokenSource(tok, login))

	wb.serial, err = ensureCharger("", func() ([]string, error) {
		return wb.chargerSerials()
	})
	if err != nil {
		return nil, err
	}

	wb.statusG = util.ResettableCached(func() (nexblueStatus, error) {
		var res nexblueStatus
		return res, wb.GetJSON(fmt.Sprintf("%s/chargers/%s/cmd/status", nexblueAPI, wb.serial), &res)
	}, cache)

	return wb, nil
}

func (wb *Nexblue) chargerSerials() ([]string, error) {
	type charger = struct {
		SerialNumber string `json:"serial_number"`
	}
	var res struct {
		Data []charger
	}

	err := wb.GetJSON(nexblueAPI+"/chargers", &res)

	return lo.Map(res.Data, func(c charger, _ int) string {
		return c.SerialNumber
	}), err
}

// Status implements the api.Charger interface
func (wb *Nexblue) Status() (api.ChargeStatus, error) {
	res, err := wb.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}

	switch res.ChargingState {
	case 0: // Idle
		return api.StatusA, nil
	case 1: // Connected
		return api.StatusB, nil
	case 2: // Charging
		return api.StatusC, nil
	case 3: // Finished
		return api.StatusB, nil
	case 4: // Error
		return api.StatusE, nil
	case 5: // Load Balancing
		return api.StatusC, nil
	case 6: // Delayed
		return api.StatusB, nil
	case 7: // EV Waiting
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid status: %d", res.ChargingState)
	}
}

// Enabled implements the api.Charger interface
func (wb *Nexblue) Enabled() (bool, error) {
	return verifyEnabled(wb, wb.enabled)
}

// Enable implements the api.Charger interface
func (wb *Nexblue) Enable(enable bool) error {
	cmd := "stop_charging"
	if enable {
		cmd = "start_charging"
	}

	req, err := request.New(http.MethodPost, fmt.Sprintf("%s/chargers/%s/cmd/%s", nexblueAPI, wb.serial, cmd), nil, nil)
	if err != nil {
		return err
	}

	var res struct {
		Result int `json:"result"`
	}
	if err := wb.DoJSON(req, &res); err != nil {
		return err
	}
	if res.Result != 0 {
		return fmt.Errorf("command %s failed: %d", cmd, res.Result)
	}

	wb.enabled = enable
	wb.statusG.Reset()
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Nexblue) MaxCurrent(current int64) error {
	uri := fmt.Sprintf("%s/chargers/%s/cmd/set_current_limit", nexblueAPI, wb.serial)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(struct {
		CurrentLimit int64 `json:"current_limit"`
	}{
		current,
	}), request.JSONEncoding)

	var res struct {
		Result int `json:"result"`
	}
	if err := wb.DoJSON(req, &res); err != nil {
		return err
	}
	if res.Result != 0 {
		return fmt.Errorf("set current limit failed: %d", res.Result)
	}

	return nil
}

var _ api.Meter = (*Nexblue)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Nexblue) CurrentPower() (float64, error) {
	res, err := wb.statusG.Get()
	return res.Power * 1e3, err
}

var _ api.ChargeRater = (*Nexblue)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Nexblue) ChargedEnergy() (float64, error) {
	res, err := wb.statusG.Get()
	return res.Energy, err
}

var _ api.MeterEnergy = (*Nexblue)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Nexblue) TotalEnergy() (float64, error) {
	res, err := wb.statusG.Get()
	return res.LifetimeEnergy, err
}

var _ api.PhaseSwitcher = (*Nexblue)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Nexblue) Phases1p3p(phases int) error {
	uri := fmt.Sprintf("%s/v1/charger/%s/setting", nexblueAPI, wb.serial)
	req, _ := request.New(http.MethodPost, uri, request.MarshalJSON(struct {
		PhaseMode int `json:"phase_mode"`
	}{
		phases,
	}), request.JSONEncoding)

	_, err := wb.DoBody(req)
	return err
}
