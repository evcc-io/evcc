package charger

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
)

// WarpHTTP is the Warp charger HTTP implementation
type WarpHTTP struct {
	*request.Helper
	emHelper *request.Helper
	log      *util.Logger
	uri      string
	emURI    string
	features []string
	current  int64
}

func init() {
	registry.Add("warp-http", NewWarpHTTPFromConfig)
}

//go:generate go tool decorate -f decorateWarpHTTP -b *WarpHTTP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewWarpHTTPFromConfig creates a new configurable charger
func NewWarpHTTPFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI                   string
		User                  string
		Password              string
		EnergyManagerURI      string
		EnergyManagerUser     string
		EnergyManagerPassword string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarpHTTP(cc.URI, cc.User, cc.Password, cc.EnergyManagerURI, cc.EnergyManagerUser, cc.EnergyManagerPassword)
	if err != nil {
		return nil, err
	}

	var currentPower, totalEnergy func() (float64, error)
	if wb.hasFeature(wb.uri, warp.FeatureMeter) {
		currentPower = wb.currentPower
		totalEnergy = wb.totalEnergy
	}

	var currents, voltages func() (float64, float64, float64, error)
	if wb.hasFeature(wb.uri, warp.FeatureMeterPhases) {
		currents = wb.currents
		voltages = wb.voltages
	}

	var identity func() (string, error)
	if wb.hasFeature(wb.uri, warp.FeatureNfc) {
		identity = wb.identify
	}

	var phases func(int) error
	var getPhases func() (int, error)
	if cc.EnergyManagerURI != "" {
		if res, err := wb.emState(); err == nil && res.ExternalControl != 1 {
			phases = wb.phases1p3p
			getPhases = wb.getPhases
		}
	}

	return decorateWarpHTTP(wb, currentPower, totalEnergy, currents, voltages, identity, phases, getPhases), nil
}

// NewWarpHTTP creates a new configurable charger
func NewWarpHTTP(uri, user, password, emURI, emUser, emPassword string) (*WarpHTTP, error) {
	log := util.NewLogger("warp-http")

	client := request.NewHelper(log)
	client.Client.Timeout = warp.Timeout
	if user != "" {
		client.Client.Transport = digest.NewTransport(user, password, client.Client.Transport)
	}

	wb := &WarpHTTP{
		Helper:  client,
		log:     log,
		uri:     util.DefaultScheme(strings.TrimRight(uri, "/"), "http"),
		current: 6000, // mA
	}

	if emURI != "" {
		wb.emURI = util.DefaultScheme(strings.TrimRight(emURI, "/"), "http")
		wb.emHelper = request.NewHelper(log)
		wb.emHelper.Client.Timeout = warp.Timeout
		if emUser != "" {
			wb.emHelper.Client.Transport = digest.NewTransport(emUser, emPassword, wb.emHelper.Client.Transport)
		}
	}

	return wb, nil
}

func (wb *WarpHTTP) hasFeature(root, feature string) bool {
	if wb.features == nil {
		var features []string
		uri := fmt.Sprintf("%s/info/features", root)

		if err := wb.GetJSON(uri, &features); err == nil {
			wb.features = features
		}
	}
	return slices.Contains(wb.features, feature)
}

// Enable implements the api.Charger interface
func (wb *WarpHTTP) Enable(enable bool) error {
	var current int64
	if enable {
		current = wb.current
	}
	return wb.setMaxCurrent(current)
}

// Enabled implements the api.Charger interface
func (wb *WarpHTTP) Enabled() (bool, error) {
	var res warp.EvseExternalCurrent
	uri := fmt.Sprintf("%s/evse/external_current", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.Current >= 6000, err
}

// Status implements the api.Charger interface
func (wb *WarpHTTP) Status() (api.ChargeStatus, error) {
	res := api.StatusNone

	var status warp.EvseState
	uri := fmt.Sprintf("%s/evse/state", wb.uri)
	err := wb.GetJSON(uri, &status)
	if err != nil {
		return res, err
	}

	switch status.Iec61851State {
	case 0:
		res = api.StatusA
	case 1:
		res = api.StatusB
	case 2:
		res = api.StatusC
	default:
		err = fmt.Errorf("invalid status: %d", status.Iec61851State)
	}

	return res, err
}

// MaxCurrent implements the api.Charger interface
func (wb *WarpHTTP) MaxCurrent(current int64) error {
	return wb.MaxCurrentMillis(float64(current))
}

var _ api.ChargerEx = (*WarpHTTP)(nil)

// MaxCurrentMillis implements the api.ChargerEx interface
func (wb *WarpHTTP) MaxCurrentMillis(current float64) error {
	curr := int64(current * 1e3)
	err := wb.setMaxCurrent(curr)
	if err == nil {
		wb.current = curr
	}
	return err
}

func (wb *WarpHTTP) setMaxCurrent(current int64) error {
	uri := fmt.Sprintf("%s/evse/external_current_update", wb.uri)
	data := map[string]int64{"current": current}

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var res interface{}
	return wb.DoJSON(req, &res)
}

// CurrentPower implements the api.Meter interface
func (wb *WarpHTTP) currentPower() (float64, error) {
	var res warp.MeterValues
	uri := fmt.Sprintf("%s/meter/values", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.Power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WarpHTTP) totalEnergy() (float64, error) {
	var res warp.MeterValues
	uri := fmt.Sprintf("%s/meter/values", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.EnergyAbs, err
}

func (wb *WarpHTTP) meterValues() ([]float64, error) {
	var res []float64
	uri := fmt.Sprintf("%s/meter/all_values", wb.uri)
	err := wb.GetJSON(uri, &res)

	if err == nil && len(res) < 6 {
		err = fmt.Errorf("invalid length: %d", len(res))
	}

	return res, err
}

// currents implements the api.MeterCurrrents interface
func (wb *WarpHTTP) currents() (float64, float64, float64, error) {
	res, err := wb.meterValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return res[3], res[4], res[5], nil
}

// voltages implements the api.MeterVoltages interface
func (wb *WarpHTTP) voltages() (float64, float64, float64, error) {
	res, err := wb.meterValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return res[0], res[1], res[2], nil
}

func (wb *WarpHTTP) identify() (string, error) {
	var res warp.ChargeTrackerCurrentCharge
	uri := fmt.Sprintf("%s/charge_tracker/current_charge", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.AuthorizationInfo.TagId, err
}

func (wb *WarpHTTP) emState() (warp.EmState, error) {
	var res warp.EmState
	uri := fmt.Sprintf("%s/power_manager/state", wb.emURI)
	err := wb.emHelper.GetJSON(uri, &res)
	return res, err
}

func (wb *WarpHTTP) emLowLevelState() (warp.EmLowLevelState, error) {
	var res warp.EmLowLevelState
	uri := fmt.Sprintf("%s/power_manager/low_level_state", wb.emURI)
	err := wb.emHelper.GetJSON(uri, &res)
	return res, err
}

// phases1p3p implements the api.PhaseSwitcher interface
func (wb *WarpHTTP) phases1p3p(phases int) error {
	res, err := wb.emState()
	if err != nil {
		return err
	}

	if res.ExternalControl > 0 {
		return fmt.Errorf("external control not available: %s", res.ExternalControl.String())
	}

	uri := fmt.Sprintf("%s/power_manager/external_control_update", wb.emURI)
	data := map[string]int{"phases_wanted": phases}

	req, err := request.New(http.MethodPut, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	var resp interface{}
	return wb.emHelper.DoJSON(req, &resp)
}

// getPhases implements the api.PhaseGetter interface
func (wb *WarpHTTP) getPhases() (int, error) {
	res, err := wb.emLowLevelState()
	if err != nil {
		return 0, err
	}

	if res.Is3phase {
		return 3, nil
	}

	return 1, nil
}
