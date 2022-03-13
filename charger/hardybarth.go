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
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/echarge"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// http://apidoc.ecb1.de
// https://github.com/evcc-io/evcc/discussions/778

// HardyBarth charger implementation
type HardyBarth struct {
	*request.Helper
	uri           string
	chargecontrol int
	cache         time.Duration
}

func init() {
	registry.Add("hardybarth", NewHardyBarthFromConfig)
}

// NewHardyBarthFromConfig creates a HardyBarth cPH1 charger from generic config
func NewHardyBarthFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI           string
		ChargeControl int
		Cache         time.Duration
	}{
		ChargeControl: 1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHardyBarth(cc.URI, cc.ChargeControl, cc.Cache)
}

// NewHardyBarth creates HardyBarth charger
func NewHardyBarth(uri string, chargecontrol int, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("hardybarth")

	wb := &HardyBarth{
		Helper:        request.NewHelper(log),
		uri:           util.DefaultScheme(uri, "http"),
		chargecontrol: chargecontrol,
		cache:         cache,
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *HardyBarth) Status() (api.ChargeStatus, error) {
	uri := fmt.Sprintf("%s/api/v1/chargecontrols/%d", wb.uri, wb.chargecontrol)

	res := struct {
		ChargeControl struct {
			echarge.ChargeControl
		}
	}{}

	err := wb.GetJSON(uri, &res)
	if err != nil {
		return api.StatusNone, err
	}

	switch s := res.ChargeControl.ChargeControl.State[:1]; s {
	case "A", "B", "C":
		return api.ChargeStatus(s), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid state: %s", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *HardyBarth) Enabled() (bool, error) {
	return false, api.ErrNotAvailable
}

// Enable implements the api.Charger interface
func (wb *HardyBarth) Enable(enable bool) error {
	return api.ErrNotAvailable
}

// MaxCurrent implements the api.Charger interface
func (wb *HardyBarth) MaxCurrent(current int64) error {
	return api.ErrNotAvailable
}

var _ api.Meter = (*HardyBarth)(nil)

// CurrentPower implements the api.Meter interface
func (wb *HardyBarth) CurrentPower() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.ChargeRater = (*HardyBarth)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *HardyBarth) ChargedEnergy() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.MeterCurrent = (*HardyBarth)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *HardyBarth) Currents() (float64, float64, float64, error) {
	return 0, 0, 0, api.ErrNotAvailable
}

var _ api.Identifier = (*HardyBarth)(nil)

// Identify implements the api.Identifier interface
func (wb *HardyBarth) Identify() (string, error) {
	return "", api.ErrNotAvailable
}

var _ api.MeterEnergy = (*HardyBarth)(nil)

// TotalEnergy implements the api.MeterEnergy interface - v2 only
func (wb *HardyBarth) TotalEnergy() (float64, error) {
	return 0, api.ErrNotAvailable
}

var _ api.ChargePhases = (*HardyBarth)(nil)

// Phases1p3p implements the api.ChargePhases interface - v2 only
func (wb *HardyBarth) Phases1p3p(phases int) error {
	return api.ErrNotAvailable
}
