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
	"net/url"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/echarge"
	"github.com/evcc-io/evcc/charger/echarge/salia"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// http://apidoc.ecb1.de
// https://github.com/evcc-io/evcc/discussions/778

// Salia charger implementation
type Salia struct {
	*request.Helper
	uri           string
	chargecontrol int
	meter         int
	current       int64
}

func init() {
	registry.Add("hardybarth-salia", NewSaliaFromConfig)
}

// NewSaliaFromConfig creates a Salia cPH1 charger from generic config
func NewSaliaFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI           string
		ChargeControl int
		Meter         int
	}{
		ChargeControl: 1,
		Meter:         1,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSalia(cc.URI, cc.ChargeControl, cc.Meter)
}

// NewSalia creates Hardy Barth charger with Salia controller
func NewSalia(uri string, chargecontrol, meter int) (api.Charger, error) {
	log := util.NewLogger("salia")

	uri = strings.TrimSuffix(uri, "/") + "/api"

	wb := &Salia{
		Helper:        request.NewHelper(log),
		uri:           util.DefaultScheme(uri, "http"),
		chargecontrol: chargecontrol,
		meter:         meter,
		current:       6,
	}

	// if !sponsor.IsAuthorized() {
	// 	return nil, api.ErrSponsorRequired
	// }

	uri = fmt.Sprintf("%s/api/v1/chargecontrols/%d/mode", wb.uri, wb.chargecontrol)
	data := url.Values{"mode": {echarge.ModeManual}}
	err := wb.post(uri, data)

	res, err := wb.get()
	fmt.Printf("%+v", res)

	return wb, err
}

func (wb *Salia) get() (salia.Api, error) {
	var res salia.Api
	err := wb.GetJSON(wb.uri, &res)
	return res, err
}

// Status implements the api.Charger interface
func (wb *Salia) Status() (api.ChargeStatus, error) {
	res, err := wb.get()
	if err != nil {
		return api.StatusNone, err
	}

	switch s := res.Secc.Port0.Ci.Charge.Cp.Status; s {
	case "A", "B", "C":
		return api.ChargeStatus(s), nil
	default:
		return api.StatusNone, fmt.Errorf("invalid state: %s", s)
	}
}

// Enabled implements the api.Charger interface
func (wb *Salia) Enabled() (bool, error) {
	res, err := wb.get()
	if err == nil && res.Secc.Port0.Salia.ChargeMode != echarge.ModeManual {
		err = fmt.Errorf("invalid mode: %s", res.Secc.Port0.Salia.ChargeMode)
	}
	return res.Secc.Port0.Ci.Evse.Basic.OfferedCurrentLimit > 0, err
}

// Enable implements the api.Charger interface
func (wb *Salia) Enable(enable bool) error {
	var current int64
	if enable {
		current = wb.current
	}

	return wb.setCurrent(current)
}

func (wb *Salia) post(uri string, data url.Values) error {
	resp, err := wb.PostForm(uri, data)
	if err == nil {
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusBadRequest {
			err = fmt.Errorf("invalid status: %s", resp.Status)
		}
	}

	return err
}

func (wb *Salia) setCurrent(current int64) error {
	uri := fmt.Sprintf("%s/api/v1/chargecontrols/%d/mode/manual/ampere", wb.uri, wb.chargecontrol)
	data := url.Values{"manualmodeamp": {fmt.Sprintf("%d", current)}}
	return wb.post(uri, data)
}

// MaxCurrent implements the api.Charger interface
func (wb *Salia) MaxCurrent(current int64) error {
	err := wb.setCurrent(current)
	if err == nil {
		wb.current = current
	}
	return err
}

var _ api.Meter = (*Salia)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Salia) CurrentPower() (float64, error) {
	res, err := wb.get()
	return res.Secc.Port0.Metering.Power.ActiveTotal.Actual / 10, err
}

var _ api.MeterEnergy = (*Salia)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *Salia) TotalEnergy() (float64, error) {
	res, err := wb.get()
	return res.Secc.Port0.Metering.Energy.ActiveImport.Actual / 1e3, err
}

var _ api.MeterCurrent = (*Salia)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Salia) Currents() (float64, float64, float64, error) {
	res, err := wb.get()
	i := res.Secc.Port0.Metering.Current.AC
	return i.L1.Actual / 1e3, i.L2.Actual / 1e3, i.L3.Actual / 1e3, err
}

// var _ api.Identifier = (*Salia)(nil)

// // Identify implements the api.Identifier interface
// func (wb *Salia) Identify() (string, error) {
// 	return "", api.ErrNotAvailable
// }
