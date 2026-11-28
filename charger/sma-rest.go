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
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/charger/sma"
	smarest "github.com/evcc-io/evcc/charger/sma-rest"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2"
)

// smaevchager charger implementation
type SmaRest struct {
	implement.Caps
	*request.Helper
	log          *util.Logger
	uri          string // 192.168.XXX.XXX
	cache        time.Duration
	measurementG util.Cacheable[[]smarest.Measurements]
	parameterG   util.Cacheable[[]smarest.Parameters]
	phases       int
	current      float64
	useSetpoint  bool // active power setpoint instead of WMaxIn and mode switching
	mu           sync.Mutex
	setpoint     string // last written WSptMin and WSptMax
	updated      time.Time
	refresh      time.Duration
	enabled      bool
	power        float64
}

// setpoint refresh if the control timeout is unknown
const smaRestRefresh = 30 * time.Second

func init() {
	// TODO remove deprecated
	registry.AddCtx("sma-rest", NewSmaRestFromConfig)
	registry.AddCtx("smaevcharger", NewSmaRestFromConfig)
}

// NewSmaRestFromConfig creates a SMA EV Charger from generic config
func NewSmaRestFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
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

	return NewSmaRest(ctx, cc.Uri, cc.User, cc.Password, cc.Cache)
}

// NewSmaRest creates an SMA EV Charger
func NewSmaRest(ctx context.Context, uri, user, password string, cache time.Duration) (api.Charger, error) {
	log := util.NewLogger("sma-rest").Redact(user, password)

	wb := &SmaRest{
		Caps:    implement.New(),
		Helper:  request.NewHelper(log),
		log:     log,
		uri:     util.DefaultScheme(strings.TrimRight(uri, "/"), "http") + "/api/v1",
		cache:   cache,
		phases:  3,
		refresh: smaRestRefresh,
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

	// devices without the active power setpoint are controlled via WMaxIn and charging mode as before
	wb.useSetpoint = wb.editable(sma.SetpointWSptMin) && wb.editable(sma.SetpointWSptMax)

	// charging control requires write access, accounts without sufficient rights do not get these parameters or cannot edit them
	if !wb.useSetpoint {
		log.DEBUG.Println("active power setpoint not available, using maximum power and charging mode")

		for _, id := range []string{sma.ChrgActChaMod, sma.InverterWMaxIn} {
			if !wb.editable(id) {
				return nil, fmt.Errorf("%s not writable, user requires administrator or installer rights", id)
			}
		}
	}

	if wb.useSetpoint {
		if enabled, err := wb.Enabled(); err == nil {
			wb.enabled = enabled
		}
	} else if values, err := wb.possibleValues(sma.ChrgActChaMod); err == nil && !slices.Contains(values, sma.BoostCharging.String()) {
		// the EV Charger rotary switch in smart position removes fast charging from the possible modes
		log.WARN.Println("rotary switch must be set to fast charging")
	}

	// the charger selects 1p below the threshold power, the legacy path never sets less than its 4500 W minimum
	if maxPower := wb.maxPower(); wb.useSetpoint && maxPower > 0 && maxPower < sma.PhaseThreshold {
		wb.phases = 1
	} else if wb.useSetpoint {
		implement.Has(wb, implement.PhaseSwitcher(wb.phases1p3p))
		implement.Has(wb, implement.PhaseGetter(wb.getPhases))

		// the charger may already run a 1p setpoint after a restart
		if v, err := wb.getParameter(sma.SetpointWSptMin); err == nil {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f != 0 {
				wb.power = math.Abs(f)
				wb.phases = sma.Phases(wb.power)
			}
		}
	}

	// disconnecting after full charge ends the session on the charger side
	if v, err := wb.code(sma.ChrgStpWhenFl); err == nil && v == sma.Yes {
		log.WARN.Println(`"Disconnect after full charge" must be set to "No"`)
	}

	// relay switching pauses charging beyond the loadpoint phase switch grace period
	if v, err := wb.code(sma.ChrgMinSwTm); err == nil && (v == sma.SwitchingTime120s || v == sma.SwitchingTime90s) {
		log.WARN.Println(`"Minimum relay switching time" should be at most 60s`)
	}

	// setpoints fall back after the control timeout
	if wb.useSetpoint {
		if v, err := wb.getParameter(sma.InverterTmsOutPrc); err == nil {
			if t, err := strconv.ParseFloat(v, 64); err == nil && t > 0 {
				wb.refresh = max(time.Duration(t*float64(time.Second))/2, time.Second)
			}
		}

		go wb.watchdog(ctx)
	}

	return wb, nil
}

// watchdog rewrites the setpoint before the charger falls back after the control timeout
func (wb *SmaRest) watchdog(ctx context.Context) {
	for {
		wb.mu.Lock()
		wait := wb.refresh
		if wb.setpoint != "" {
			wait -= time.Since(wb.updated)
		}
		wb.mu.Unlock()

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return
		}

		wb.mu.Lock()
		if wb.setpoint != "" && time.Since(wb.updated) >= wb.refresh {
			if err := wb.sendSetpoint(wb.setpoint); err != nil {
				wb.log.ERROR.Println("watchdog:", err)
			}
			wb.updated = time.Now()
		}
		wb.mu.Unlock()
	}
}

// sendSetpoint writes WSptMax and WSptMin to the same value, making the setpoint exact in all charging modes
func (wb *SmaRest) sendSetpoint(val string) error {
	return wb.Send(value(sma.SetpointWSptMax, val), value(sma.SetpointWSptMin, val))
}

// applySetpoint writes the setpoint, zero while disabled
func (wb *SmaRest) applySetpoint() error {
	val := "0"
	if wb.enabled {
		// consumption is negative
		val = strconv.Itoa(-int(math.Round(wb.power)))
	}

	if err := wb.sendSetpoint(val); err != nil {
		return err
	}

	wb.setpoint = val
	wb.updated = time.Now()

	return nil
}

func (wb *SmaRest) editable(id string) bool {
	p, err := wb.parameter(id)
	return err == nil && p.Editable
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
	res, err := wb.getMeasurement(sma.OperationEVehChaStt)
	return sma.Code(res), err
}

var _ api.StatusReasoner = (*SmaRest)(nil)

// StatusReason implements the api.StatusReasoner interface
func (wb *SmaRest) StatusReason() (api.Reason, error) {
	access, err := wb.code(sma.ChrgChrgLok)
	if err != nil {
		return api.ReasonUnknown, err
	}

	release, err := wb.code(sma.ChrgChrgApv)
	if err != nil {
		return api.ReasonUnknown, err
	}

	return sma.StatusReason(access, release), nil
}

// Enabled implements the api.Charger interface
func (wb *SmaRest) Enabled() (bool, error) {
	if wb.useSetpoint {
		v, err := wb.getParameter(sma.SetpointWSptMin)
		if err != nil {
			return false, err
		}

		// an empty value means no setpoint, the charger is not limited
		f, err := strconv.ParseFloat(v, 64)
		return err != nil || f != 0, nil
	}

	mode, err := wb.mode()
	if err != nil {
		return false, err
	}

	// the eCharger falls back to its default pv surplus mode on its own, report it as disabled to restore fast charging
	return mode == sma.BoostCharging, nil
}

// Enable implements the api.Charger interface
func (wb *SmaRest) Enable(enable bool) error {
	if !wb.useSetpoint {
		mode := sma.ChargeStop
		if enable {
			mode = sma.BoostCharging
		}

		return wb.Send(value(sma.ChrgActChaMod, mode.String()))
	}

	// the loadpoint sets the current before enabling, minimum current otherwise
	power := wb.power
	if enable && power == 0 {
		power = wb.calcPower(6, wb.phases)
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	enabled := wb.enabled
	wb.enabled = enable
	wb.power = power

	if err := wb.applySetpoint(); err != nil {
		wb.enabled = enabled
		return err
	}

	return nil
}

func (wb *SmaRest) mode() (sma.Code, error) {
	return wb.code(sma.ChrgActChaMod)
}

// code reads an enum parameter
func (wb *SmaRest) code(channel string) (sma.Code, error) {
	res, err := wb.getParameter(channel)
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

	if err := wb.setPower(current, wb.phases); err != nil {
		return err
	}

	wb.current = current

	return nil
}

// setPower writes the power limit, the charger selects 1p or 3p internally based on it
func (wb *SmaRest) setPower(current float64, phases int) error {
	power := wb.calcPower(current, phases)

	if !wb.useSetpoint {
		return wb.Send(value(sma.InverterWMaxIn, strconv.Itoa(int(math.Round(power)))))
	}

	wb.mu.Lock()
	defer wb.mu.Unlock()

	prev := wb.power
	wb.power = power

	if !wb.enabled {
		return nil
	}

	if err := wb.applySetpoint(); err != nil {
		wb.power = prev
		return err
	}

	return nil
}

// calcPower converts current and phases to the charging power within the charger limits
func (wb *SmaRest) calcPower(current float64, phases int) float64 {
	power := grid.CurrentToPower(current, phases)

	p, err := wb.parameter(sma.InverterWMaxIn)
	if err != nil {
		return power
	}

	if wb.useSetpoint {
		// the charger selects the phases based on the setpoint
		if phases > 1 {
			power = max(power, sma.MinPower3p)
		} else {
			power = min(power, sma.MaxPower1p)
		}

		// the setpoint is limited by the configured maximum power
		if v, err := strconv.ParseFloat(p.Value, 64); err == nil && v != 0 {
			power = min(power, math.Abs(v))
		}
		return power
	}

	// the charger rejects values outside the parameter range, e.g. a 4500 W minimum on the EV Charger
	if p.Min != nil {
		power = max(power, *p.Min)
	}
	if p.Max != nil {
		power = min(power, *p.Max)
	}

	return power
}

// maxPower returns the configured maximum power, 0 if unknown
func (wb *SmaRest) maxPower() float64 {
	p, err := wb.parameter(sma.InverterWMaxIn)
	if err != nil {
		return 0
	}

	v, _ := strconv.ParseFloat(p.Value, 64)
	return math.Abs(v)
}

var _ api.PowerLimiter = (*SmaRest)(nil)

// GetMinMaxPower implements the api.PowerLimiter interface
func (wb *SmaRest) GetMinMaxPower() (float64, float64, error) {
	// the legacy path controls via the maximum power, it is no limit there
	if !wb.useSetpoint {
		return 0, 0, api.ErrNotAvailable
	}

	maxPower := wb.maxPower()
	if maxPower == 0 {
		return 0, 0, api.ErrNotAvailable
	}

	// the minimum power is not exposed, 0 leaves the minimum current in effect
	return 0, maxPower, nil
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *SmaRest) phases1p3p(phases int) error {
	// loadpoint may not rewrite an unchanged current, hence apply the new phases immediately
	if wb.current > 0 {
		if err := wb.setPower(wb.current, phases); err != nil {
			return err
		}
	}

	wb.phases = phases

	return nil
}

// getPhases implements the api.PhaseGetter interface
func (wb *SmaRest) getPhases() (int, error) {
	v, err := wb.getParameter(sma.SetpointWSptMin)
	if err != nil {
		return 0, err
	}

	power, err := strconv.ParseFloat(v, 64)
	switch {
	case err != nil: // not limited, the charger uses its configured maximum
		power = wb.maxPower()
	case power == 0: // disabled, phases follow the stored setpoint on enable
		wb.mu.Lock()
		power = wb.power
		wb.mu.Unlock()
	}

	if power == 0 {
		return 0, nil
	}

	return sma.Phases(math.Abs(power)), nil
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
	res, err := wb.getMeasurement(sma.ChaSessWhIn)
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
		if el.ChannelId == id && len(el.Values) > 0 {
			return el.Values[0].Value, nil
		}
	}

	return 0, &smarest.ErrUnknownMeasurement{Measurement: id}
}

func (wb *SmaRest) parameter(id string) (smarest.Parameter, error) {
	res, err := wb.parameterG.Get()
	if err != nil {
		return smarest.Parameter{}, err
	}

	if len(res) == 0 {
		return smarest.Parameter{}, fmt.Errorf("unknown parameter: %s", id)
	}

	for _, el := range res[0].Values {
		if el.ChannelId == id {
			return el, nil
		}
	}

	return smarest.Parameter{}, fmt.Errorf("unknown parameter: %s", id)
}

func (wb *SmaRest) getParameter(id string) (string, error) {
	p, err := wb.parameter(id)
	return p.Value, err
}

func (wb *SmaRest) possibleValues(id string) ([]string, error) {
	p, err := wb.parameter(id)
	return p.PossibleValues, err
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
