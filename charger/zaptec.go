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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/zaptec"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// https://api.zaptec.com/help/index.html

// Zaptec charger implementation
type Zaptec struct {
	*request.Helper
	log     *util.Logger
	statusG func() (zaptec.StateResponse, error)
	id      string
	current int64
	cache   time.Duration
}

func init() {
	registry.Add("zaptec", NewZaptecFromConfig)
}

// NewZaptecFromConfig creates a Zaptec Pro charger from generic config
func NewZaptecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User, Password string
		Id             string
		Cache          time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("need user and password")
	}

	return NewZaptec(cc.User, cc.Password, cc.Id, cc.Cache)
}

// NewZaptec creates Zaptec charger
func NewZaptec(user, password, id string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("zaptec").Redact(user, password)

	c := &Zaptec{
		Helper:  request.NewHelper(log),
		log:     log,
		id:      id,
		current: 6, // assume min current
		cache:   cache,
	}

	// setup cached values
	c.reset()

	data := url.Values{
		"grant_type": {"password"},
		"username":   {user},
		"password":   {password},
	}

	uri := fmt.Sprintf("%s/oauth/token", zaptec.ApiURL)
	req, err := request.New(http.MethodPost, uri, strings.NewReader(data.Encode()), request.URLEncoding)
	if err == nil {
		var token oauth2.Token
		if err = c.DoJSON(req, &token); err == nil {
			c.Transport = &oauth2.Transport{
				Source: oauth2.StaticTokenSource(&token),
				Base:   c.Transport,
			}
		}
	}

	if err == nil {
		c.id, err = ensureCharger(c.id, c.chargers)
	}

	return c, err
}

func (c *Zaptec) reset() {
	c.statusG = provider.Cached(func() (zaptec.StateResponse, error) {
		var res zaptec.StateResponse

		uri := fmt.Sprintf("%s/api/chargers/%s/state", zaptec.ApiURL, c.id)
		err := c.GetJSON(uri, &res)

		return res, err
	}, c.cache)
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
	res, err := c.statusG()
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
	res, err := c.statusG()
	if !res.ObservationByID(zaptec.IsEnabled).Bool() ||
		res.ObservationByID(zaptec.FinalStopActive).Bool() {
		return false, err
	}

	return c.current > 0, err
}

// Enable implements the api.Charger interface
func (c *Zaptec) Enable(enable bool) error {
	var (
		current int64
		res     zaptec.StateResponse
		err     error
	)

	if enable {
		current = c.current

		res, err = c.statusG()
		if err != nil {
			return err
		}

		if res.ObservationByID(zaptec.FinalStopActive).Bool() {
			return errors.New("cannot enable: final stop active")
		}

		if !res.ObservationByID(zaptec.IsEnabled).Bool() {
			uri := fmt.Sprintf("%s/api/chargers/%s/sendCommand/%d", zaptec.ApiURL, c.id, zaptec.CmdResume)

			var req *http.Request
			req, err = request.New(http.MethodPost, uri, nil, request.JSONEncoding)
			if err == nil {
				_, err = c.DoBody(req)
				c.reset()
			}
		}
	}

	if err == nil {
		err = c.setCurrent(current)
	}

	return err
}

func (c *Zaptec) update(data zaptec.Update) error {
	uri := fmt.Sprintf("%s/api/chargers/%s/update", zaptec.ApiURL, c.id)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = c.DoBody(req)
		c.reset()
	}

	return err
}

// setCurrent sets the charger current
func (c *Zaptec) setCurrent(current int64) error {
	curr := int(current)
	data := zaptec.Update{
		MaxChargeCurrent: &curr,
	}

	return c.update(data)
}

// MaxCurrent implements the api.Charger interface
func (c *Zaptec) MaxCurrent(current int64) error {
	err := c.setCurrent(current)
	if err == nil {
		c.current = current
	}

	return err
}

var _ api.Meter = (*Zaptec)(nil)

// CurrentPower implements the api.Meter interface
func (c *Zaptec) CurrentPower() (float64, error) {
	res, err := c.statusG()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePower).Float64()
}

var _ api.ChargeRater = (*Zaptec)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Zaptec) ChargedEnergy() (float64, error) {
	res, err := c.statusG()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePowerSession).Float64()
}

var _ api.MeterCurrent = (*Zaptec)(nil)

// Currents implements the api.MeterCurrent interface
func (c *Zaptec) Currents() (float64, float64, float64, error) {
	res, err := c.statusG()
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
	data := zaptec.Update{
		MaxChargePhases: &phases,
	}

	return c.update(data)
}
