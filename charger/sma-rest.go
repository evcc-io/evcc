package charger

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

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
	"github.com/evcc-io/evcc/charger/sma"
	smarest "github.com/evcc-io/evcc/charger/sma-rest"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
)

// smaevchager charger implementation
type SmaRest struct {
	*request.Helper
	log          *util.Logger
	uri          string // 192.168.XXX.XXX
	cache        time.Duration
	measurementG util.Cacheable[[]smarest.Measurements]
	parameterG   util.Cacheable[[]smarest.Parameters]
}

func init() {
	// TODO remove deprecated
	registry.Add("smaevcharger", NewSmaRestFromConfig)
}

// NewSmaRestFromConfig creates a SMA EV Charger from generic config
func NewSmaRestFromConfig(other map[string]any) (api.Charger, error) {
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

	return NewSmaRest(cc.Uri, cc.User, cc.Password, cc.Cache)
}

// NewSmaRest creates an SMA EV Charger
func NewSmaRest(uri, user, password string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("sma-rest").Redact(user, password)

	wb := &SmaRest{
		Helper: request.NewHelper(log),
		log:    log,
		uri:    util.DefaultScheme(strings.TrimRight(uri, "/"), "http") + "/api/v1",
		cache:  cache,
	}

	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	// setup cached values
	wb.measurementG = util.ResettableCached(wb._measurementData, wb.cache)
	wb.parameterG = util.ResettableCached(wb._parameterData, wb.cache)

	ts, err := smarest.TokenSource(log, wb.uri, user, password)
	if err != nil {
		return nil, err
	}

	// replace client transport with authenticated transport
	wb.Client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   wb.Client.Transport,
	}

	// get model
	model, err := wb.getParameter("Parameter.Nameplate.ModelStr")
	if err != nil {
		return nil, err
	}

	isLegacyHw := !strings.HasSuffix(model, "-20")

	// get version
	pkgRev, err := wb.getParameter("Parameter.Nameplate.PkgRev")
	if err != nil {
		return nil, err
	}
	swVersion, err := version.NewVersion(strings.TrimSuffix(pkgRev, ".R"))
	if err != nil {
		return nil, err
	}
	refVersion, err := version.NewVersion(smarest.MinAcceptedVersion)
	if err != nil {
		return nil, err
	}

	// check minimum version
	if swVersion.Compare(refVersion) < 0 {
		return nil, fmt.Errorf("charger firmware version not supported - please update >= %s", smarest.MinAcceptedVersion)
	}

	// app or RFID release enables charging on its own and conflicts with charging control
	lock, err := wb.getParameter(sma.ChannelAccessManagement)
	if err != nil {
		return nil, err
	}

	if lock == sma.AccessManagementOn.String() {
		log.WARN.Println(`access management must be set to "Unlocked" instead of "App / RFID"`)
	}

	// pv surplus and target modes would regulate charging on their own
	mode, err := wb.mode()
	if err != nil {
		return nil, err
	}

	if mode != sma.ModeStop && mode != sma.ModeFast {
		log.WARN.Printf("charging mode is %d, fast charging (%d) required", mode, sma.ModeFast)
	}

	// the eCharger has no rotary switch and reports a meaningless position
	if isLegacyHw {
		pos, err := wb.getMeasurement("Measurement.Chrg.ModSw")
		if err != nil {
			return nil, err
		}

		if pos == smarest.SwitchOeko {
			log.WARN.Println("rotary switch must be set to fast charging")
		}
	}

	return wb, nil
}

// Status implements the api.Charger interface
func (wb *SmaRest) Status() (api.ChargeStatus, error) {
	state, err := wb.status()
	if err != nil {
		return api.StatusNone, err
	}

	return sma.ChargeStatus(state)
}

func (wb *SmaRest) status() (sma.Code, error) {
	res, err := wb.getMeasurement(sma.ChannelStatus)
	return sma.Code(res), err
}

var _ api.StatusReasoner = (*SmaRest)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *SmaRest) StatusReason() (api.Reason, error) {
	state, err := wb.status()
	if err != nil || state != sma.StatusLocked {
		return api.ReasonUnknown, err
	}

	lock, err := wb.getParameter(sma.ChannelAccessManagement)
	if err != nil {
		return api.ReasonUnknown, err
	}

	return sma.StatusReason(state, lock == sma.AccessManagementOn.String()), nil
}

// Enabled implements the api.Charger interface
func (wb *SmaRest) Enabled() (bool, error) {
	mode, err := wb.mode()
	if err != nil {
		return false, err
	}

	// the eCharger falls back to its default pv surplus mode on its own, report it as disabled to restore fast charging
	return mode == sma.ModeFast, nil
}

// Enable implements the api.Charger interface
func (wb *SmaRest) Enable(enable bool) error {
	mode := sma.ModeStop
	if enable {
		mode = sma.ModeFast
	}

	return wb.Send(value(sma.ChannelMode, mode.String()))
}

func (wb *SmaRest) mode() (sma.Code, error) {
	res, err := wb.getParameter(sma.ChannelMode)
	if err != nil {
		return 0, err
	}

	return sma.ParseCode(res)
}

// MaxCurrent implements the api.Charger interface
func (wb *SmaRest) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*SmaRest)(nil)

// maxCurrentMillis implements the api.ChargerEx interface
func (wb *SmaRest) MaxCurrentMillis(current float64) error {
	if current < 6 {
		return fmt.Errorf("invalid current %.5g", current)
	}

	return wb.Send(value("Parameter.Inverter.AcALim", fmt.Sprintf("%.2f", current)))
}

var _ api.MeterEnergy = (*SmaRest)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (wb *SmaRest) TotalEnergy() (float64, error) {
	res, err := wb.getMeasurement("Measurement.Metering.GridMs.TotWhIn")
	if _, ok := errors.AsType[*smarest.ErrUnknownMeasurement](err); ok {
		res, err = wb.getMeasurement("Measurement.Metering.GridMs.TotWhIn.ChaSta")
	}
	return res / 1e3, err
}

var _ api.Meter = (*SmaRest)(nil)

// CurrentPower implements the api.Meter interface
func (wb *SmaRest) CurrentPower() (float64, error) {
	res, err := wb.getMeasurement("Measurement.Metering.GridMs.TotWIn")
	if _, ok := errors.AsType[*smarest.ErrUnknownMeasurement](err); ok {
		res, err = wb.getMeasurement("Measurement.Metering.GridMs.TotWIn.ChaSta")
	}
	return res, err
}

var _ api.ChargeRater = (*SmaRest)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (wb *SmaRest) ChargedEnergy() (float64, error) {
	res, err := wb.getMeasurement(sma.ChannelSessionEnergy)
	if _, ok := errors.AsType[*smarest.ErrUnknownMeasurement](err); ok {
		res, err = wb.getMeasurement("Measurement.Metering.GridMs.TotWhIn.ChaSta")
	}
	return res / 1e3, err
}

var _ api.PhaseCurrents = (*SmaRest)(nil)

// Currents implements the api.PhaseCurrents interface
func (wb *SmaRest) Currents() (float64, float64, float64, error) {
	var res [3]float64

	for i, phase := range []string{"A", "B", "C"} {
		val, err := wb.getMeasurement("Measurement.GridMs.A.phs" + phase)
		if err != nil {
			return 0, 0, 0, err
		}

		res[i] = -val
	}

	return res[0], res[1], res[2], nil
}

// reset cache
func (wb *SmaRest) reset() {
	wb.measurementG.Reset()
	wb.parameterG.Reset()
}

func (wb *SmaRest) _measurementData() ([]smarest.Measurements, error) {
	var res []smarest.Measurements
	uri := fmt.Sprintf("%s/measurements/live", wb.uri)
	data := `[{"componentId": "IGULD:SELF"}]`

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), request.JSONEncoding)
	if err == nil {
		err = wb.DoJSON(req, &res)
	}

	return res, err
}

func (wb *SmaRest) _parameterData() ([]smarest.Parameters, error) {
	var res []smarest.Parameters
	uri := fmt.Sprintf("%s/parameters/search/", wb.uri)
	data := `{"queryItems":[{"componentId":"IGULD:SELF"}]}`

	req, err := request.New(http.MethodPost, uri, strings.NewReader(data), request.JSONEncoding)
	if err == nil {
		err = wb.DoJSON(req, &res)
	}

	return res, err
}

func (wb *SmaRest) getMeasurement(id string) (float64, error) {
	res, err := wb.measurementG.Get()
	if err != nil {
		return 0, err
	}

	for _, el := range res {
		if el.ChannelId == id {
			return el.Values[0].Value, nil
		}
	}

	return 0, &smarest.ErrUnknownMeasurement{Measurement: id}
}

func (wb *SmaRest) getParameter(id string) (string, error) {
	res, err := wb.parameterG.Get()
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

func (wb *SmaRest) Send(values ...smarest.Value) error {
	uri := fmt.Sprintf("%s/parameters/IGULD:SELF/", wb.uri)
	data := smarest.SendParameter{
		Values: values,
	}

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err == nil {
		_, err = wb.DoBody(req)
		wb.reset()
	}

	return err
}

// value creates an smarest.Value
func value(id, value string) smarest.Value {
	return smarest.Value{
		Timestamp: time.Now().UTC().Format(smarest.TimestampFormat),
		ChannelId: id,
		Value:     value,
	}
}
