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
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/echarge"
	"github.com/evcc-io/evcc/charger/echarge/salia"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/hashicorp/go-version"
)

// https://github.com/evcc-io/evcc/discussions/778

// Salia charger implementation
type Salia struct {
	*request.Helper
	log     *util.Logger
	uri     string
	current int64
	fw      int // 2 if fw 2.0
	apiG    util.Cacheable[salia.Api]
}

func init() {
	registry.AddCtx("hardybarth-salia", NewSaliaFromConfig)
}

//go:generate go tool decorate -f decorateSalia -b *Salia -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewSaliaFromConfig creates a Salia cPH2 charger from generic config
func NewSaliaFromConfig(ctx context.Context, other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewSalia(ctx, cc.URI, cc.Cache)
}

// NewSalia creates Hardy Barth charger with Salia controller
func NewSalia(ctx context.Context, uri string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("salia")

	uri = strings.TrimSuffix(uri, "/") + "/api"

	wb := &Salia{
		log:     log,
		Helper:  request.NewHelper(log),
		uri:     util.DefaultScheme(uri, "http"),
		current: 6,
	}

	wb.apiG = util.ResettableCached(func() (salia.Api, error) {
		var res salia.Api
		err := wb.GetJSON(wb.uri, &res)
		return res, err
	}, cache)

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	// set chargemode manual
	res, err := wb.apiG.Get()
	if err != nil {
		return nil, err
	}

	v, err := version.NewSemver(res.Device.SoftwareVersion)
	if err != nil {
		return nil, err
	}

	if v.GreaterThanOrEqual(version.Must(version.NewSemver("2.0.0"))) {
		wb.fw = 2
	}

	if res.Secc.Port0.Salia.ChargeMode != echarge.ModeManual {
		if err = wb.post(salia.ChargeMode, echarge.ModeManual); err == nil {
			res, err = wb.apiG.Get()
		}

		if err == nil && res.Secc.Port0.Salia.ChargeMode != echarge.ModeManual {
			err = errors.New("could not change chargemode to manual")
		}
	}

	if err != nil {
		return nil, err
	}

	go wb.heartbeat(ctx)

	wb.pause(false)

	var (
		currentPower func() (float64, error)
		totalEnergy  func() (float64, error)
		currents     func() (float64, float64, float64, error)
		phasesG      func() (int, error)
		phasesS      func(int) error
	)

	if res.Secc.Port0.Metering.Meter.Available > 0 {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
		currents = wb.currents
	}

	if res.Secc.Port0.Salia.PhaseSwitching.Actual > 0 {
		phasesG = wb.getPhases
		phasesS = wb.phases1p3p
	}

	return decorateSalia(wb, currentPower, totalEnergy, currents, phasesS, phasesG), nil
}

func (wb *Salia) heartbeat(ctx context.Context) {
	bo := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(5*time.Second),
		backoff.WithMaxInterval(time.Minute))

	for tick := time.Tick(30 * time.Second); ; {
		select {
		case <-tick:
		case <-ctx.Done():
			return
		}

		if err := backoff.Retry(func() error {
			return wb.post(salia.HeartBeat, "alive")
		}, bo); err != nil {
			wb.log.ERROR.Println("heartbeat:", err)
		}
	}
}

func (wb *Salia) post(key, val string) error {
	data := map[string]string{key: val}
	uri := fmt.Sprintf("%s/secc", wb.uri)

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		var res struct {
			Result string
		}

		if err = wb.DoJSON(req, &res); err == nil {
			if res.Result != "ok" {
				err = fmt.Errorf("invalid result: %s", res.Result)
			}
		}
	}

	wb.apiG.Reset()

	return err
}

// Status implements the api.Charger interface
func (wb *Salia) Status() (api.ChargeStatus, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return api.StatusNone, err
	}
	return api.ChargeStatusString(res.Secc.Port0.Ci.Charge.Cp.Status)
}

// Enabled implements the api.Charger interface
func (wb *Salia) Enabled() (bool, error) {
	res, err := wb.apiG.Get()
	if err == nil && res.Secc.Port0.Salia.ChargeMode != echarge.ModeManual {
		err = fmt.Errorf("invalid mode: %s", res.Secc.Port0.Salia.ChargeMode)
	}

	if wb.fw < 2 {
		return res.Secc.Port0.GridCurrentLimit > 0 && res.Secc.Port0.Salia.PauseCharging == 0, err
	}

	return res.Secc.Port0.Ci.Evse.Basic.OfferedCurrentLimit > 0 && res.Secc.Port0.Salia.PauseCharging == 0, err
}

func (wb *Salia) pause(enable bool) {
	// ignore error for FW <1.52
	offOn := map[bool]string{false: "1", true: "0"}
	_ = wb.post(salia.PauseCharging, offOn[enable])
}

// Enable implements the api.Charger interface
func (wb *Salia) Enable(enable bool) error {
	var current int64
	if enable {
		current = wb.current
	}

	err := wb.setCurrent(current)
	if err == nil {
		wb.pause(enable)
	}

	return err
}

func (wb *Salia) setCurrent(current int64) error {
	return wb.post(salia.GridCurrentLimit, strconv.Itoa(int(current)))
}

// MaxCurrent implements the api.Charger interface
func (wb *Salia) MaxCurrent(current int64) error {
	err := wb.setCurrent(current)
	if err == nil {
		wb.current = current
	}
	return err
}

// currentPower implements the api.Meter interface
func (wb *Salia) currentPower() (float64, error) {
	res, err := wb.apiG.Get()
	return res.Secc.Port0.Metering.Power.ActiveTotal.Actual / 10, err
}

// totalEnergy implements the api.MeterEnergy interface
func (wb *Salia) totalEnergy() (float64, error) {
	res, err := wb.apiG.Get()
	return res.Secc.Port0.Metering.Energy.ActiveImport.Actual / 1e3, err
}

// currents implements the api.PhaseCurrents interface
func (wb *Salia) currents() (float64, float64, float64, error) {
	res, err := wb.apiG.Get()
	i := res.Secc.Port0.Metering.Current.AC
	return i.L1.Actual / 1e3, i.L2.Actual / 1e3, i.L3.Actual / 1e3, err
}

func (wb *Salia) Identify() (string, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return "", err
	}
	return res.Secc.Port0.RFID.AuthorizationRequest.Key, nil
}

func (wb *Salia) getPhases() (int, error) {
	res, err := wb.apiG.Get()
	if err != nil {
		return 0, err
	}

	if res.Secc.Port0.Salia.PhaseSwitching.Actual == 0 {
		return 0, api.ErrNotAvailable
	}

	return res.Secc.Port0.Salia.PhaseSwitching.Actual, nil
}

func (wb *Salia) phases1p3p(phases int) error {
	p, err := wb.getPhases()
	if err != nil {
		return err
	}
	if p == phases {
		return nil
	}
	return wb.post(salia.SetPhase, "toggle")
}

var _ api.Diagnosis = (*Salia)(nil)

// Diagnose implements the api.Diagnosis interface
func (wb *Salia) Diagnose() {
	res, err := wb.apiG.Get()
	if err == nil {
		fmt.Printf("Model name: %s\n", res.Device.ModelName)
		fmt.Printf("Software version: %s\n", res.Device.SoftwareVersion)
	}
}
