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
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

// https://api.zaptec.com/help/index.html

// Zaptec charger implementation
type Zaptec struct {
	*request.Helper
	cache   time.Duration
	id      string
	updated time.Time
	status  zaptec.StateResponse
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
	}{}

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
		Helper: request.NewHelper(log),
		cache:  cache,
	}

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

	var res zaptec.ChargersResponse
	if err == nil {
		uri = fmt.Sprintf("%s/api/chargers", zaptec.ApiURL)
		if err = c.GetJSON(uri, &res); err == nil {
			// TODO
			c.id = res.Data[0].Id

			chargers := lo.Map(res.Data, func(c zaptec.Charger, _ int) string {
				return c.Id
			})

			fmt.Println(chargers)
			panic(0)
		}
	}

	// TODO IsStandAlone

	// panic(0)
	return c, err
}

func (c *Zaptec) state() (zaptec.StateResponse, error) {
	var err error
	if time.Since(c.updated) >= c.cache {
		var res zaptec.StateResponse

		uri := fmt.Sprintf("%s/api/chargers/%s/state", zaptec.ApiURL, c.id)
		if err = c.GetJSON(uri, &res); err == nil {
			for _, s := range res {
				if s.ValueAsString != "" {
					fmt.Println(zaptec.ObservationID(s.StateId).String(), s.ValueAsString)
				}
			}

			c.updated = time.Now()
			c.status = res
		}
	}

	return c.status, err
}

// Status implements the api.Charger interface
func (c *Zaptec) Status() (api.ChargeStatus, error) {
	res, err := c.state()
	if err != nil {
		return api.StatusA, err
	}

	switch i, err := res.ObservationByID(zaptec.ChargerOperationMode).Int(); i {
	case 1:
		return api.StatusA, err
	case 2, 5:
		return api.StatusB, err
	case 3:
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
	res, err := c.state()
	return res.ObservationByID(zaptec.IsEnabled).Bool(), err
}

// Enable implements the api.Charger interface
func (c *Zaptec) Enable(enable bool) error {
	return api.ErrMustRetry
}

func (c *Zaptec) update(data zaptec.Update) error {
	uri := fmt.Sprintf("%s/api/chargers/%s/update", zaptec.ApiURL, c.id)

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		err = c.DoJSON(req, &data)
		c.updated = time.Time{}
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *Zaptec) MaxCurrent(current int64) error {
	curr := int(current)
	data := zaptec.Update{
		MaxChargeCurrent: &curr,
	}

	return c.update(data)
}

var _ api.Meter = (*Zaptec)(nil)

// CurrentPower implements the api.Meter interface
func (c *Zaptec) CurrentPower() (float64, error) {
	res, err := c.state()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePower).Float64()
}

var _ api.ChargeRater = (*Zaptec)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Zaptec) ChargedEnergy() (float64, error) {
	res, err := c.state()
	if err != nil {
		return 0, err
	}

	return res.ObservationByID(zaptec.TotalChargePowerSession).Float64()
}

var _ api.MeterCurrent = (*Zaptec)(nil)

// Currents implements the api.MeterCurrent interface
func (c *Zaptec) Currents() (float64, float64, float64, error) {
	res, err := c.state()
	if err != nil {
		return 0, 0, 0, err
	}

	i1, err := res.ObservationByID(zaptec.CurrentPhase1).Float64()

	var i2, i3 float64
	if err == nil {
		i2, err = res.ObservationByID(zaptec.CurrentPhase2).Float64()
	}
	if err == nil {
		i3, err = res.ObservationByID(zaptec.CurrentPhase3).Float64()
	}

	return i1, i2, i3, err
}

var _ api.ChargePhases = (*Zaptec)(nil)

// Phases1p3p implements the api.ChargePhases interface
func (c *Zaptec) Phases1p3p(phases int) error {
	data := zaptec.Update{
		MaxChargePhases: &phases,
	}

	return c.update(data)
}
