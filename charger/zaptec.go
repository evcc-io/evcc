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
	"golang.org/x/oauth2"
)

// https://api.zaptec.com/help/index.html

// Zaptec charger implementation
type Zaptec struct {
	*request.Helper
	token *oauth2.Token
}

func init() {
	registry.Add("zaptec", NewZaptecFromConfig)
}

// NewZaptecFromConfig creates a Zaptec Pro charger from generic config
func NewZaptecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		User, Password string
		Cache          time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.User == "" || cc.Password == "" {
		return nil, errors.New("need user and password")
	}

	return NewZaptec(cc.User, cc.Password, cc.Cache)
}

// NewZaptec creates Zaptec charger
func NewZaptec(user, password string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("zaptec").Redact(user, password)

	c := &Zaptec{
		Helper: request.NewHelper(log),
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
		err = c.GetJSON(uri, &res)

		fmt.Printf("%+v\n", res)
	}

	if err == nil {
		var res2 []zaptec.State
		uri = fmt.Sprintf("%s/api/chargers/%s/state", zaptec.ApiURL, res.Data[0].Id)
		err = c.GetJSON(uri, &res2)

		fmt.Printf("%+v\n", res)
	}

	return c, err
}

// Status implements the api.Charger interface
func (c *Zaptec) Status() (api.ChargeStatus, error) {
	// resp, err := c.api.Status()
	// if err != nil {
	// 	return api.StatusNone, err
	// }

	// switch car := resp.Status(); car {
	// case 1:
	// 	return api.StatusA, nil
	// case 2:
	// 	return api.StatusC, nil
	// case 3, 4:
	// 	return api.StatusB, nil
	// default:
	// 	return api.StatusNone, fmt.Errorf("car unknown result: %d", car)
	// }

	return api.StatusA, nil
}

// Enabled implements the api.Charger interface
func (c *Zaptec) Enabled() (bool, error) {
	return false, nil
}

// Enable implements the api.Charger interface
func (c *Zaptec) Enable(enable bool) error {

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c *Zaptec) MaxCurrent(current int64) error {
	return nil
}

// var _ api.Meter = (*Zaptec)(nil)

// // CurrentPower implements the api.Meter interface
// func (c *Zaptec) CurrentPower() (float64, error) {
// 	resp, err := c.api.Status()
// 	if err != nil {
// 		return 0, err
// 	}

// 	return resp.CurrentPower(), err
// }

// var _ api.ChargeRater = (*Zaptec)(nil)

// // ChargedEnergy implements the api.ChargeRater interface
// func (c *Zaptec) ChargedEnergy() (float64, error) {
// 	resp, err := c.api.Status()
// 	if err != nil {
// 		return 0, err
// 	}

// 	return resp.ChargedEnergy(), err
// }

// var _ api.MeterCurrent = (*Zaptec)(nil)

// // Currents implements the api.MeterCurrent interface
// func (c *Zaptec) Currents() (float64, float64, float64, error) {
// 	resp, err := c.api.Status()
// 	if err != nil {
// 		return 0, 0, 0, err
// 	}

// 	i1, i2, i3 := resp.Currents()

// 	return i1, i2, i3, err
// }

// var _ api.Identifier = (*Zaptec)(nil)

// // Identify implements the api.Identifier interface
// func (c *Zaptec) Identify() (string, error) {
// 	resp, err := c.api.Status()
// 	if err != nil {
// 		return "", err
// 	}
// 	return resp.Identify(), nil
// }

// // totalEnergy implements the api.MeterEnergy interface - v2 only
// func (c *Zaptec) totalEnergy() (float64, error) {
// 	resp, err := c.api.Status()
// 	if err != nil {
// 		return 0, err
// 	}

// 	var val float64
// 	if res, ok := resp.(*Zaptec.StatusResponse2); ok {
// 		val = res.TotalEnergy()
// 	}

// 	return val, err
// }

// // phases1p3p implements the api.ChargePhases interface - v2 only
// func (c *Zaptec) phases1p3p(phases int) error {
// 	if phases == 3 {
// 		phases = 2
// 	}

// 	return c.api.Update(fmt.Sprintf("psm=%d", phases))
// }
