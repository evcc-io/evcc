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
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// Blueprint charger implementation
type Blueprint struct {
	*request.Helper
	cache time.Duration
}

func init() {
	// registry.Add("foo", NewBlueprintFromConfig)
}

// NewBlueprintFromConfig creates a blueprint charger from generic config
func NewBlueprintFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc struct {
		URI   string
		Cache time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewBlueprint(cc.URI, cc.Cache)
}

// NewBlueprint creates Blueprint charger
func NewBlueprint(uri string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("foo")

	wb := &Blueprint{
		Helper: request.NewHelper(log),
		cache:  cache,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Blueprint) Status() (api.ChargeStatus, error) {
	return api.StatusNone, api.ErrNotAvailable
}

// Enabled implements the api.Charger interface
func (wb *Blueprint) Enabled() (bool, error) {
	return false, api.ErrNotAvailable
}

// Enable implements the api.Charger interface
func (wb *Blueprint) Enable(enable bool) error {
	return api.ErrNotAvailable
}

// MaxCurrent implements the api.Charger interface
func (wb *Blueprint) MaxCurrent(current int64) error {
	return api.ErrNotAvailable
}

var _ api.Meter = (*Blueprint)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Blueprint) CurrentPower() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.ChargeRater = (*Blueprint)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Blueprint) ChargedEnergy() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.MeterEnergy = (*Blueprint)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Blueprint) TotalEnergy() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.PhaseCurrents = (*Blueprint)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Blueprint) Currents() (float64, float64, float64, error) {
	return 0, 0, 0, api.ErrNotAvailable
}

var _ api.Identifier = (*Blueprint)(nil)

// Identify implements the api.Identifier interface
func (wb *Blueprint) Identify() (string, error) {
	return "", api.ErrNotAvailable
}

var _ api.PhaseSwitcher = (*Blueprint)(nil)

// Phases1p3p implements the api.PhaseSwitcher interface
func (wb *Blueprint) Phases1p3p(phases int) error {
	return api.ErrNotAvailable
}
