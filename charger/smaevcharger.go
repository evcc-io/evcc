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
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/smaevcharger"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
)

// smaevchager charger implementation
type Smaevcharger struct {
	*request.Helper
	log          *util.Logger
	uri          string // 192.168.XXX.XXX
	cache        time.Duration
	oldstate     float64
	measurementG func() ([]smaevcharger.Measurements, error)
	parameterG   func() ([]smaevcharger.Parameters, error)
}

func init() {
	registry.Add("smaevcharger", NewSmaevchargerFromConfig)
}

// NewSmaevchargerFromConfig creates a SMA EV Charger from generic config
func NewSmaevchargerFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Uri      string
		User     string
		Password string
		Cache    time.Duration
	}{
		Cache: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Uri == "" {
		return nil, errors.New("missing uri")
	}

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	if cc.User == "admin" {
		return nil, errors.New(`user "admin" not allowed, create new user`)
	}

	return NewSmaevcharger(cc.Uri, cc.User, cc.Password, cc.Cache)
}

// NewSmaevcharger creates an SMA EV Charger
func NewSmaevcharger(uri, user, password string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("smaevcharger").Redact(user, password)

	wb := &Smaevcharger{
		Helper: request.NewHelper(log),
		log:    log,
		uri:    util.DefaultScheme(strings.TrimRight(uri, "/"), "http") + "/api/v1",
		cache:  cache,
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	// setup cached values
	wb.reset()

	ts, err := smaevcharger.TokenSource(log, wb.uri, user, password)
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

	if err == nil {
		// Prepare charger: disable App Lock functionality.
		// This option have been introduced with 1.2.23 and will lock the charger
		// until unlocked via SMA App. Unfortunately this Lock option will overwrite
		// the status of the charger and prevent ev detection
		err = wb.Send(
			value("Parameter.Chrg.ChrgLok", smaevcharger.ChargerAppLockDisabled),
			value("Parameter.Chrg.ChrgApv", smaevcharger.ChargerManualLockEnabled),
		)
	}

	return wb, err
}

// Status implements the api.Charger interface
func (wb *Smaevcharger) Status() (api.ChargeStatus, error) {
	state, err := wb.getMeasurement("Measurement.Operation.EVeh.ChaStt")
	if err != nil {
		return api.StatusNone, err
	}

	if state != wb.oldstate {
		// if the wallbox detects a car, it automatically switches to the charging state of the selector switch.
		// Since EVCC requires the fast charging option, the wallbox would immediately start charging with maximum charging power,
		// without taking into account the desired state of evcc. Since this is not desired,
		// the charging status must be changed / overwritten from fast charging to charging stop as soon as a vehicle is detected (StatusB)
		// After that, EVCC can decide which charging option should be selected.

		if state == smaevcharger.StatusB && wb.oldstate == smaevcharger.StatusA {
			if err := wb.Send(value("Parameter.Chrg.ActChaMod", smaevcharger.StopCharge)); err != nil {
				return api.StatusNone, err
			}
		}
		wb.oldstate = state
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
	mode, err := wb.getParameter("Parameter.Chrg.ActChaMod")
	if err != nil {
		return false, err
	}

	switch mode {
	case smaevcharger.FastCharge, // Schnellladen - 4718
		smaevcharger.OptiCharge, // Optimiertes Laden - 4719
		smaevcharger.PlanCharge: // Laden mit Vorgabe - 4720
		return true, nil
	case smaevcharger.StopCharge: // Ladestopp - 4721
		return false, nil
	default:
		return false, fmt.Errorf("invalid charge mode: %s", mode)
	}
}

// Enable implements the api.Charger interface
func (wb *Smaevcharger) Enable(enable bool) error {
	if enable {
		res, err := wb.getMeasurement("Measurement.Chrg.ModSw")
		if err != nil {
			return err
		}

		if res == smaevcharger.SwitchOeko {
			// Switch in PV Loading position
			// If the selector switch of the wallbox is in the wrong position (eco-charging and not fast charging),
			// the charging process is started with eco-charging when it is activated,
			// which may be desired when integrated with SHM.
			// Since evcc does not have full control over the charging station in this mode,
			// a corresponding error is returned to indicate the incorrect switch position.
			// If the wallbox is installed without SHM, charging in eco mode is not possible.
			_ = wb.Send(value("Parameter.Chrg.ActChaMod", smaevcharger.OptiCharge))
			return fmt.Errorf("switch position not on fast charging - SMA's own optimized charging was activated")
		}

		// Switch in Fast charging position
		return wb.Send(value("Parameter.Chrg.ActChaMod", smaevcharger.FastCharge))
	}

	// else
	return wb.Send(value("Parameter.Chrg.ActChaMod", smaevcharger.StopCharge))
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

	return wb.Send(value("Parameter.Inverter.AcALim", fmt.Sprintf("%.2f", current)))
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

var _ api.PhaseCurrents = (*Smaevcharger)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *Smaevcharger) Currents() (float64, float64, float64, error) {
	var curr []float64

	for _, phase := range []string{"A", "B", "C"} {
		val, err := wb.getMeasurement("Measurement.GridMs.A.phs" + phase)
		if err != nil {
			return 0, 0, 0, err
		}

		curr = append(curr, -val)
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

func (wb *Smaevcharger) Send(values ...smaevcharger.Value) error {
	uri := fmt.Sprintf("%s/parameters/IGULD:SELF/", wb.uri)
	data := smaevcharger.SendParameter{
		Values: values,
	}

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
		wb.reset()
	}

	return err
}

// value creates an smaevcharger.Value
func value(id, value string) smaevcharger.Value {
	return smaevcharger.Value{
		Timestamp: time.Now().UTC().Format(smaevcharger.TimestampFormat),
		ChannelId: id,
		Value:     value,
	}
}
