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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
)

// https://go-e.co/app/api.pdf
// https://github.com/Zapteccharger/go-eCharger-API-v1/
// https://github.com/Zapteccharger/go-eCharger-API-v2/

// Zaptec charger implementation
type Zaptec struct {
	*util.Helper
}

func init() {
	registry.Add("zaptec", NewZaptecFromConfig)
}

// NewZaptecFromConfig creates a Zaptec Pro charger from generic config
func NewZaptecFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Token string
		URI   string
		Cache time.Duration
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI != "" && cc.Token != "" {
		return nil, errors.New("should only have one of uri/token")
	}
	if cc.URI == "" && cc.Token == "" {
		return nil, errors.New("must have one of uri/token")
	}

	return NewZaptec(cc.URI, cc.Token, cc.Cache)
}

// NewZaptec creates Zaptec charger
func NewZaptec(uri, token string, cache time.Duration) (api.Charger, error) {
	c := &Zaptec{}

	log := util.NewLogger("zaptec").Redact(token)

	if token != "" {
		c.api = Zaptec.NewCloud(log, token, cache)
	} else {
		c.api = Zaptec.NewLocal(log, util.DefaultScheme(uri, "http"), cache)
	}

	if c.api.IsV2() {
		var phases func(int) error
		if sponsor.IsAuthorized() {
			phases = c.phases1p3p
		} else {
			log.WARN.Println("automatic 1p3p phase switching requires sponsor token")
		}

		return decorateZaptec(c, c.totalEnergy, phases), nil
	}

	return c, nil
}

// Status implements the api.Charger interface
func (c *Zaptec) Status() (api.ChargeStatus, error) {
	resp, err := c.api.Status()
	if err != nil {
		return api.StatusNone, err
	}

	switch car := resp.Status(); car {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusC, nil
	case 3, 4:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("car unknown result: %d", car)
	}
}

// Enabled implements the api.Charger interface
func (c *Zaptec) Enabled() (bool, error) {
	resp, err := c.api.Status()
	if err != nil {
		return false, err
	}

	return resp.Enabled(), nil
}

// Enable implements the api.Charger interface
func (c *Zaptec) Enable(enable bool) error {
	var b int
	if enable {
		b = 1
	}

	param := map[bool]string{false: "alw", true: "frc"}[c.api.IsV2()]
	if c.api.IsV2() {
		b += 1
	}

	return c.api.Update(fmt.Sprintf("%s=%d", param, b))
}

// MaxCurrent implements the api.Charger interface
func (c *Zaptec) MaxCurrent(current int64) error {
	param := map[bool]string{false: "amx", true: "amp"}[c.api.IsV2()]
	return c.api.Update(fmt.Sprintf("%s=%d", param, current))
}

var _ api.Meter = (*Zaptec)(nil)

// CurrentPower implements the api.Meter interface
func (c *Zaptec) CurrentPower() (float64, error) {
	resp, err := c.api.Status()
	if err != nil {
		return 0, err
	}

	return resp.CurrentPower(), err
}

var _ api.ChargeRater = (*Zaptec)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Zaptec) ChargedEnergy() (float64, error) {
	resp, err := c.api.Status()
	if err != nil {
		return 0, err
	}

	return resp.ChargedEnergy(), err
}

var _ api.MeterCurrent = (*Zaptec)(nil)

// Currents implements the api.MeterCurrent interface
func (c *Zaptec) Currents() (float64, float64, float64, error) {
	resp, err := c.api.Status()
	if err != nil {
		return 0, 0, 0, err
	}

	i1, i2, i3 := resp.Currents()

	return i1, i2, i3, err
}

var _ api.Identifier = (*Zaptec)(nil)

// Identify implements the api.Identifier interface
func (c *Zaptec) Identify() (string, error) {
	resp, err := c.api.Status()
	if err != nil {
		return "", err
	}
	return resp.Identify(), nil
}

// totalEnergy implements the api.MeterEnergy interface - v2 only
func (c *Zaptec) totalEnergy() (float64, error) {
	resp, err := c.api.Status()
	if err != nil {
		return 0, err
	}

	var val float64
	if res, ok := resp.(*Zaptec.StatusResponse2); ok {
		val = res.TotalEnergy()
	}

	return val, err
}

// phases1p3p implements the api.ChargePhases interface - v2 only
func (c *Zaptec) phases1p3p(phases int) error {
	if phases == 3 {
		phases = 2
	}

	return c.api.Update(fmt.Sprintf("psm=%d", phases))
}
