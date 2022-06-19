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
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/chargeiq"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// ChargeIq charger implementation
type ChargeIq struct {
	*request.Helper
	uri    string
	curr   int64
	meterG func() (chargeiq.MeterStatus, error)
	cache  time.Duration
}

func init() {
	registry.Add("chargeiq", NewChargeIqFromConfig)
}

// NewChargeIqFromConfig creates a ChargeIq charger from generic config
func NewChargeIqFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewChargeIq(cc.URI, cc.Cache)
}

// NewChargeIq creates ChargeIq charger
func NewChargeIq(uri string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("chargeiq")

	wb := &ChargeIq{
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
		curr:   6,
		cache:  cache,
	}

	// cache meter readings
	wb.meterG = provider.Cached(func() (chargeiq.MeterStatus, error) {
		var res chargeiq.MeterStatus
		uri := fmt.Sprintf("%s/meter/status", wb.uri)
		err := wb.GetJSON(uri, &res)
		return res, err
	}, wb.cache)

	return wb, nil
}

func (wb *ChargeIq) status() (chargeiq.ChargeStatus, error) {
	var res chargeiq.ChargeStatus
	uri := fmt.Sprintf("%s/charge/status", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res, err
}

// Status implements the api.Charger interface
func (wb *ChargeIq) Status() (api.ChargeStatus, error) {
	resp, err := wb.status()

	res := api.StatusNone
	switch resp.Status {
	case "ready":
		res = api.StatusA
	case "ev":
		res = api.StatusB
	case "charging":
		res = api.StatusC
	default:
		if err == nil {
			err = fmt.Errorf("invalid status: %s", resp.Status)
		}
	}

	return res, err
}

// Enabled implements the api.Charger interface
func (wb *ChargeIq) Enabled() (bool, error) {
	var res chargeiq.ChargeMaxAmps
	uri := fmt.Sprintf("%s/charge/max_amps", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.Max > 0, err
}

// Enable implements the api.Charger interface
func (wb *ChargeIq) Enable(enable bool) error {
	var curr int64
	if enable {
		curr = wb.curr
	}
	return wb.setCurrent(curr)
}

func (wb *ChargeIq) setCurrent(current int64) error {
	uri := fmt.Sprintf("%s/charge/max_amps", wb.uri)

	data := struct {
		Max int64 `json:"max"`
	}{
		Max: current,
	}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (wb *ChargeIq) MaxCurrent(current int64) error {
	err := wb.setCurrent(current)
	if err == nil {
		wb.curr = current
	}
	return err
}

var _ api.Meter = (*ChargeIq)(nil)

// CurrentPower implements the api.Meter interface
func (wb *ChargeIq) CurrentPower() (float64, error) {
	res, err := wb.meterG()
	if err != nil {
		return 0, err
	}
	return (res.Pow[0] + res.Pow[1] + res.Pow[2]) * 1e3, nil
}

var _ api.MeterEnergy = (*ChargeIq)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *ChargeIq) TotalEnergy() (float64, error) {
	var res chargeiq.MeterRead
	uri := fmt.Sprintf("%s/meter/read", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.Energy, err
}

var _ api.MeterCurrent = (*ChargeIq)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *ChargeIq) Currents() (float64, float64, float64, error) {
	res, err := wb.meterG()
	if err != nil {
		return 0, 0, 0, err
	}
	return res.Curr[0], res.Curr[1], res.Curr[2], nil
}
