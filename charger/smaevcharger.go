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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/smaevcharger"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
)

// smaevchager charger implementation
type Smaevcharger struct {
	*request.Helper
	log          *util.Logger
	uri          string // 192.168.XXX.XXX
	user         string // LOGIN user
	password     string // password
	cache        time.Duration
	oldstate     float64
	measurementG func() ([]smaevcharger.Measurements, error)
	parameterG   func() ([]smaevcharger.Parameters, error)
}

func init() {
	registry.Add("smaevcharger", NewSmaevchargerFromConfig)
}

// NewSmaevchargerFromConfig creates a Smaevcharger charger from generic config
func NewSmaevchargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Host     string
		User     string
		Password string
		Cache    time.Duration
	}{
		Cache: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Host == "" {
		return nil, errors.New("missing host")
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.User == "admin" {
		return nil, errors.New("user admin not allowed, create new user")
	}

	return NewSmaevcharger(cc.Host, cc.User, cc.Password, cc.Cache)
}

// NewSmaevcharger creates Smaevcharger charger
func NewSmaevcharger(host string, user string, password string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("smaevcharger").Redact(user, password)

	baseUri := "http://" + host

	wb := &Smaevcharger{
		Helper:   request.NewHelper(log),
		log:      log,
		uri:      baseUri + "/api/v1",
		user:     user,
		password: password,
		cache:    cache,
	}

	// cached values
	wb.reset()

	ts, err := smaevcharger.TokenSource(log, baseUri, wb.user, wb.password)
	if err != nil {
		return wb, err
	}

	// replace client transport with authenticated transport
	wb.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   wb.Client.Transport,
	}

	var swVersion, refVersion *version.Version

	pkgRev, err := wb.getParameter("Parameter.Nameplate.PkgRev")
	if err == nil {
		swVersion, err = version.NewVersion(strings.TrimSuffix(pkgRev, ".R"))
	}
	if err == nil {
		refVersion, err = version.NewVersion(smaevcharger.MinAcceptedVersion)
	}
	if err == nil && swVersion.Compare(refVersion) < 0 {
		err = errors.New("charger software version not supported - please update >= " + smaevcharger.MinAcceptedVersion)
	}
	if err != nil {
		return nil, err
	}

	// lock Chargers Auto load functionality to prevent "charger out of sync"
	wb.SendMultiParameter([]smaevcharger.SendData{{
		ChannelId: "Parameter.Chrg.ChrgLok",
		Value:     smaevcharger.ChargerAppLockDisabled,
	}, {
		ChannelId: "Parameter.Chrg.ChrgApv",
		Value:     smaevcharger.ChargerManualLockEnabled,
	}})

	// TODO handle error return
	wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.StopCharge) //need to send this command as a second command to prevent auto state change

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *Smaevcharger) Status() (api.ChargeStatus, error) {
	state, err := wb.getMeasurement("Measurement.Operation.EVeh.ChaStt")
	if err != nil {
		return api.StatusNone, err
	}

	if state != wb.oldstate {
		wb.oldstate = state

		// TODO why does status B require require refresh? please add comment.
		if state == smaevcharger.StatusB {
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.StopCharge)
			wb.reset()
		}
	}

	switch state {
	case smaevcharger.StatusA:
		return api.StatusA, nil
	case smaevcharger.StatusB:
		return api.StatusB, nil
	case smaevcharger.StatusC:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("invalid state: %.0f", state)
	}
}

// Enabled implements the api.Charger interface
func (wb *Smaevcharger) Enabled() (bool, error) {
	StateChargerMode, err := wb.getParameter("Parameter.Chrg.ActChaMod")
	if err != nil {
		return false, err
	}
	switch StateChargerMode {
	case smaevcharger.FastCharge: // Schnellladen - 4718
		return true, nil
	case smaevcharger.OptiCharge: // Optimiertes Laden - 4719
		return true, nil
	case smaevcharger.PlanCharge: // Laden mit Vorgabe - 4720
		return true, nil
	case smaevcharger.StopCharge: // Ladestopp - 4721
		return false, nil
	}
	return false, fmt.Errorf("SMA EV Charger  charge mode: %s", StateChargerMode)
}

// Enable implements the api.Charger interface
func (wb *Smaevcharger) Enable(enable bool) error {
	StateChargerSwitch, err := wb.getMeasurement("Measurement.Chrg.ModSw")
	if err != nil {
		return err
	}

	if enable {
		switch StateChargerSwitch {
		case smaevcharger.SwitchOeko: // Switch PV Loading
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.OptiCharge)
			wb.reset()
			return fmt.Errorf("error while activating the charging process, switch position not on fast charging - SMA's own optimized charging was activated")
		case smaevcharger.SwitchFast: // Fast charging
			wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.FastCharge)
		}
	} else {
		wb.SendParameter("Parameter.Chrg.ActChaMod", smaevcharger.StopCharge)
	}

	wb.reset()
	return nil
}

// MaxCurrent implements the api.Charger interface
func (wb *Smaevcharger) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*Smaevcharger)(nil)

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *Smaevcharger) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}
	wb.SendParameter("Parameter.Inverter.AcALim", strconv.FormatFloat(current, 'f', 2, 64))
	time.Sleep(time.Second)
	return nil
}

var _ api.Meter = (*Smaevcharger)(nil)

// CurrentPower implements the api.Meter interface
func (wb *Smaevcharger) CurrentPower() (float64, error) {
	return wb.getMeasurement("Measurement.Metering.GridMs.TotWIn")
}

var _ api.ChargeRater = (*Smaevcharger)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *Smaevcharger) ChargedEnergy() (float64, error) {
	res, err := wb.getMeasurement("Measurement.ChaSess.WhIn")
	return res / 1e3, err
}

var _ api.MeterCurrent = (*Smaevcharger)(nil)

// Currents implements the api.MeterCurrent interface
func (wb *Smaevcharger) Currents() (float64, float64, float64, error) {
	var curr []float64

	for _, phase := range []string{"A", "B", "C"} {
		val, err := wb.getMeasurement("Measurement.GridMs.A.phs" + phase)
		if err != nil {
			return 0, 0, 0, err
		}

		curr = append(curr, val)
	}

	return curr[0], curr[1], curr[2], nil
}

// reset cache
func (wb *Smaevcharger) reset() {
	wb.measurementG = provider.Cached(wb._measurementData, wb.cache)
	wb.parameterG = provider.Cached(wb._parameterData, wb.cache)
}

func (wb *Smaevcharger) _measurementData() ([]smaevcharger.Measurements, error) {
	var res []smaevcharger.Measurements

	uri := fmt.Sprintf("%s/measurements/live", wb.uri)
	data := `[{"componentId": "IGULD:SELF"}]`

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), request.JSONEncoding)
	if err == nil {
		err = wb.DoJSON(req, &res)
	}

	return res, err
}

func (wb *Smaevcharger) _parameterData() ([]smaevcharger.Parameters, error) {
	var res []smaevcharger.Parameters
	uri := fmt.Sprintf("%s/parameters/search/", wb.uri)
	data := `{"queryItems":[{"componentId":"IGULD:SELF"}]}`

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), request.JSONEncoding)
	if err == nil {
		err = wb.DoJSON(req, &res)
	}

	return res, err
}

func (wb *Smaevcharger) getMeasurement(id string) (float64, error) {
	res, err := wb.measurementG()
	if err != nil {
		return 0, err
	}

	for _, el := range res {
		if el.ChannelId == id {
			return el.Values[0].Value, nil
		}
	}

	return 0, fmt.Errorf("unknown measurement: %s", id)
}

func (wb *Smaevcharger) getParameter(id string) (string, error) {
	res, err := wb.parameterG()
	if err != nil {
		return "", err
	}

	for _, el := range res[0].Values {
		if el.ChannelId == id {
			return el.Value, nil
		}
	}

	return "", fmt.Errorf("unknown parameter: %s", id)
}

// TODO return error instead of true/false
func (wb *Smaevcharger) SendParameter(id string, value string) bool {
	wb.reset()

	data := smaevcharger.SendParameter{
		Values: []smaevcharger.SendData{{
			Timestamp: time.Now().UTC().Format(smaevcharger.SendParameterFormat),
			ChannelId: id,
			Value:     value,
		}},
	}

	uri := fmt.Sprintf("%s/parameters/IGULD:SELF/", wb.uri)

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		var res any
		err = wb.DoJSON(req, &res)
		return err == nil
	}

	return false
}

// TODO return error instead of true/false
func (wb *Smaevcharger) SendMultiParameter(send []smaevcharger.SendData) bool {
	wb.reset()

	var data smaevcharger.SendParameter

	for _, el := range send {
		data.Values = append(data.Values, smaevcharger.SendData{
			Timestamp: time.Now().UTC().Format(smaevcharger.SendParameterFormat),
			ChannelId: el.ChannelId,
			Value:     el.Value,
		})
	}

	uri := fmt.Sprintf("%s/parameters/IGULD:SELF/", wb.uri)

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		var res any
		err = wb.DoJSON(req, &res)
		return err == nil
	}

	return false
}
