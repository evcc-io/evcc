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
	"errors"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	wattpilot "github.com/mabunixda/wattpilot"
)

// Wattpilot charger implementation
type Wattpilot struct {
	api *wattpilot.Wattpilot
}

func init() {
	registry.Add("wattpilot", NewWattpilotFromConfig)
}

// NewWattpilotFromConfig creates a wattpilot charger from generic config
func NewWattpilotFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI      string
		Password string
		Cache    time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" || cc.Password == "" {
		return nil, errors.New("must have one  uri and password")
	}

	return NewWattpilot(cc.URI, cc.Password, cc.Cache)
}

// NewWattpilot creates Wattpilot charger
func NewWattpilot(uri, password string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("wattpilot")
	c := &Wattpilot{}

	log.INFO.Println("Connecting wattpilot local", uri)
	c.api = wattpilot.NewWattpilot(uri, password)
	if connected, err := c.api.Connect(); !connected || err != nil {
		return nil, err
	}

	return c, nil
}

// Status implements the api.Charger interface
func (c *Wattpilot) Status() (api.ChargeStatus, error) {

	car, err := c.api.GetProperty("car")
	if err != nil {
		return api.StatusNone, err
	}

	switch car.(float64) {
	case 1.0:
		return api.StatusA, nil
	case 2.0, 5.0:
		return api.StatusC, nil
	case 3.0, 4.0:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("car unknown result: %d", car)
	}
}

// Enabled implements the api.Charger interface
func (c *Wattpilot) Enabled() (bool, error) {
	resp, err := c.api.GetProperty("alw")
	if err != nil {
		return false, err
	}
	return resp.(bool), nil
}

// Enable implements the api.Charger interface
func (c *Wattpilot) Enable(enable bool) error {
	forceState := 0 // neutral
	if !enable {
		forceState = 1 // off
	}

	return c.api.SetProperty("frc", forceState)
}

// MaxCurrent implements the api.Charger interface
func (c *Wattpilot) MaxCurrent(current int64) error {
	return c.api.SetCurrent(float64(current))
}

var _ api.Meter = (*Wattpilot)(nil)

// CurrentPower implements the api.Meter interface
func (c *Wattpilot) CurrentPower() (float64, error) {
	return c.api.GetPower()
}

var _ api.ChargeRater = (*Wattpilot)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *Wattpilot) ChargedEnergy() (float64, error) {
	resp, err := c.api.GetProperty("wh")
	if err != nil {
		return 0, err
	}
	return resp.(float64) / 1e3, err
}

var _ api.MeterCurrent = (*Wattpilot)(nil)

// Currents implements the api.MeterCurrent interface
func (c *Wattpilot) Currents() (float64, float64, float64, error) {
	return c.api.GetCurrents()
}

var _ api.Identifier = (*Wattpilot)(nil)

// Identify implements the api.Identifier interface
func (c *Wattpilot) Identify() (string, error) {
	return c.api.GetRFID()
}
