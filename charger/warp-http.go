package charger

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/warp"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/jpfielding/go-http-digest/pkg/digest"
	"github.com/samber/lo"
)

// WarpHTTP is the Warp charger HTTP implementation
type WarpHTTP struct {
	*request.Helper
	emHelper        *request.Helper
	log             *util.Logger
	uri             string
	emURI           string
	features        []string
	current         int64
	meterIndex      uint
	metersValuesMap map[int]int
	cache           time.Duration
	meterValuesG    util.Cacheable[warp.MeterValues]
	meterAllValuesG util.Cacheable[[]float64]
	metersValuesG   util.Cacheable[warp.MetersValues]
}

func init() {
	registry.Add("warp-http", NewWarpHTTPFromConfig)
}

//go:generate go tool decorate -f decorateWarpHTTP -b *WarpHTTP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewWarpHTTPFromConfig creates a new configurable charger
func NewWarpHTTPFromConfig(other map[string]any) (api.Charger, error) {
	var cc = struct {
		URI                    string
		User                   string
		Password               string
		EnergyManagerURI       string
		EnergyManagerUser      string
		EnergyManagerPassword  string
		DisablePhaseAutoSwitch bool
		EnergyMeterIndex       uint
		Cache                  time.Duration
	}{
		Cache: 5 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarpHTTP(cc.URI, cc.User, cc.Password, cc.Cache)
	if err != nil {
		return nil, err
	}

	var currentPower, totalEnergy func() (float64, error)
	var currents, voltages func() (float64, float64, float64, error)
	if wb.hasFeature(warp.FeatureMeters) {
		wb.meterIndex = cc.EnergyMeterIndex
		indices, err := wb.metersValueIds()
		if err != nil {
			return nil, err
		}
		wb.metersValuesMap = indices

		currentPower = wb.metersCurrentPower
		totalEnergy = wb.metersTotalEnergy
		currents = wb.metersCurrents
		voltages = wb.metersVoltages
	} else if wb.hasFeature(warp.FeatureMeter) {
		// fallback to meter api
		currentPower = wb.meterCurrentPower
		totalEnergy = wb.meterTotalEnergy
		if wb.hasFeature(warp.FeatureMeterPhases) {
			currents = wb.meterCurrents
			voltages = wb.meterVoltages
		}
	}

	var identity func() (string, error)
	if wb.hasFeature(warp.FeatureNfc) {
		identity = wb.identify
	}

	if wb.hasFeature(warp.FeaturePhaseSwitch) {
		wb.emURI = wb.uri
		wb.emHelper = wb.Helper
	} else if cc.EnergyManagerURI != "" { // fallback to Energy Manager
		wb.emURI = util.DefaultScheme(strings.TrimRight(cc.EnergyManagerURI, "/"), "http")
		wb.emHelper = request.NewHelper(wb.log)
		wb.emHelper.Client.Timeout = warp.Timeout
		if cc.EnergyManagerUser != "" {
			wb.emHelper.Client.Transport = digest.NewTransport(cc.EnergyManagerUser, cc.EnergyManagerPassword, wb.emHelper.Client.Transport)
		}
	}

	var phases func(int) error
	var getPhases func() (int, error)
	if wb.emHelper != nil {
		if res, err := wb.emState(); err == nil && res.ExternalControl != warp.ExternalControlDeactivated {
			phases = wb.phases1p3p
			getPhases = wb.getPhases
		}
	}

	if cc.DisablePhaseAutoSwitch {
		// unfortunately no feature to check for, instead this is set in template
		if err := wb.disablePhaseAutoSwitch(); err != nil {
			return nil, err
		}
	}

	return decorateWarpHTTP(wb, currentPower, totalEnergy, currents, voltages, identity, phases, getPhases), nil
}

// NewWarpHTTP creates a new configurable charger
func NewWarpHTTP(uri, user, password string, cache time.Duration) (*WarpHTTP, error) {
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
		cache:   cache,
	}

	wb.meterValuesG = util.ResettableCached(wb._meterValues, wb.cache)
	wb.meterAllValuesG = util.ResettableCached(wb._meterAllValues, wb.cache)
	wb.metersValuesG = util.ResettableCached(wb._metersValues, wb.cache)

	return wb, nil
}

func (wb *WarpHTTP) hasFeature(feature string) bool {
	if wb.features == nil {
		var features []string
		uri := fmt.Sprintf("%s/info/features", wb.uri)

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
	if err := wb.GetJSON(uri, &status); err != nil {
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
		return res, fmt.Errorf("invalid status: %d", status.Iec61851State)
	}

	return res, nil
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
	uri := fmt.Sprintf("%s/evse/external_current", wb.uri)
	data := map[string]int64{"current": current}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = wb.Do(req)
	return err
}

// CurrentPower implements the api.Meter interface
func (wb *WarpHTTP) meterCurrentPower() (float64, error) {
	res, err := wb.meterValues()
	return res.Power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WarpHTTP) meterTotalEnergy() (float64, error) {
	res, err := wb.meterValues()
	return res.EnergyAbs, err
}

func (wb *WarpHTTP) meterValues() (warp.MeterValues, error) {
	return wb.meterValuesG.Get()
}

func (wb *WarpHTTP) _meterValues() (warp.MeterValues, error) {
	var res warp.MeterValues
	uri := fmt.Sprintf("%s/meter/values", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res, err
}

func (wb *WarpHTTP) meterAllValues() ([]float64, error) {
	return wb.meterAllValuesG.Get()
}

func (wb *WarpHTTP) _meterAllValues() ([]float64, error) {
	var res []float64
	uri := fmt.Sprintf("%s/meter/all_values", wb.uri)
	err := wb.GetJSON(uri, &res)

	if err == nil && len(res) < 6 {
		err = fmt.Errorf("invalid length: %d", len(res))
	}

	return res, err
}

// currents implements the api.MeterCurrents interface
func (wb *WarpHTTP) meterCurrents() (float64, float64, float64, error) {
	res, err := wb.meterAllValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return res[3], res[4], res[5], nil
}

// voltages implements the api.PhaseVoltages interface
func (wb *WarpHTTP) meterVoltages() (float64, float64, float64, error) {
	res, err := wb.meterAllValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return res[0], res[1], res[2], nil
}

// metersValueIds returns a map from meter value IDs to their positions in the values array.
// It covers the following IDs: VoltageL1N, VoltageL2N, VoltageL3N, CurrentImExSumL1, CurrentImExSumL2,
// CurrentImExSumL3, PowerImExSum, and EnergyAbsImSum.
func (wb *WarpHTTP) metersValueIds() (map[int]int, error) {
	var res []int
	uri := fmt.Sprintf("%s/meters/%d/value_ids", wb.uri, wb.meterIndex)
	if err := wb.GetJSON(uri, &res); err != nil {
		return nil, err
	}

	requiredIDs := []int{
		warp.MetersValueIDVoltageL1N,
		warp.MetersValueIDVoltageL2N,
		warp.MetersValueIDVoltageL3N,
		warp.MetersValueIDCurrentImExSumL1,
		warp.MetersValueIDCurrentImExSumL2,
		warp.MetersValueIDCurrentImExSumL3,
		warp.MetersValueIDPowerImExSum,
		warp.MetersValueIDEnergyAbsImSum,
	}

	// Check if all required IDs are present
	missing, _ := lo.Difference(requiredIDs, res)
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required meter value IDs: %v", missing)
	}

	// Build indices map
	var indices = make(map[int]int)
	for i, valueIdx := range res {
		if lo.Contains(requiredIDs, valueIdx) {
			indices[valueIdx] = i
		}
	}

	return indices, nil
}

func (wb *WarpHTTP) metersValues() (warp.MetersValues, error) {
	return wb.metersValuesG.Get()
}

func (wb *WarpHTTP) _metersValues() (warp.MetersValues, error) {
	var res []float64
	uri := fmt.Sprintf("%s/meters/%d/values", wb.uri, wb.meterIndex)
	err := wb.GetJSON(uri, &res)
	if err != nil {
		return warp.MetersValues{}, err
	}

	var result warp.MetersValues

	get := func(id int, name string) float64 {
		if err != nil {
			return 0
		}

		idx, ok := wb.metersValuesMap[id]
		if !ok {
			err = fmt.Errorf("%s value ID not found", name)
			return 0
		}
		if idx >= len(res) {
			err = fmt.Errorf("%s index out of range: idx=%d, len(values)=%d", name, idx, len(res))
			return 0
		}
		return res[idx]
	}

	result.VoltageL1N = get(warp.MetersValueIDVoltageL1N, "voltage L1N")
	result.VoltageL2N = get(warp.MetersValueIDVoltageL2N, "voltage L2N")
	result.VoltageL3N = get(warp.MetersValueIDVoltageL3N, "voltage L3N")
	result.CurrentImExSumL1 = get(warp.MetersValueIDCurrentImExSumL1, "current L1")
	result.CurrentImExSumL2 = get(warp.MetersValueIDCurrentImExSumL2, "current L2")
	result.CurrentImExSumL3 = get(warp.MetersValueIDCurrentImExSumL3, "current L3")
	result.PowerImExSum = get(warp.MetersValueIDPowerImExSum, "power")
	result.EnergyAbsImSum = get(warp.MetersValueIDEnergyAbsImSum, "energy")

	return result, err
}

// CurrentPower implements the api.Meter interface
func (wb *WarpHTTP) metersCurrentPower() (float64, error) {
	values, err := wb.metersValues()

	return values.PowerImExSum, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WarpHTTP) metersTotalEnergy() (float64, error) {
	values, err := wb.metersValues()

	return values.EnergyAbsImSum, err
}

// currents implements the api.PhaseCurrents interface
func (wb *WarpHTTP) metersCurrents() (float64, float64, float64, error) {
	values, err := wb.metersValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return values.CurrentImExSumL1, values.CurrentImExSumL2, values.CurrentImExSumL3, nil
}

// voltages implements the api.PhaseVoltages interface
func (wb *WarpHTTP) metersVoltages() (float64, float64, float64, error) {
	values, err := wb.metersValues()
	if err != nil {
		return 0, 0, 0, err
	}

	return values.VoltageL1N, values.VoltageL2N, values.VoltageL3N, nil
}

func (wb *WarpHTTP) identify() (string, error) {
	var res warp.ChargeTrackerCurrentCharge
	uri := fmt.Sprintf("%s/charge_tracker/current_charge", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.AuthorizationInfo.TagId, err
}

func (wb *WarpHTTP) disablePhaseAutoSwitch() error {
	uri := fmt.Sprintf("%s/evse/phase_auto_switch", wb.uri)
	data := map[string]bool{"enabled": false}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return fmt.Errorf("disabling phase auto switch failed: %v", err)
	}

	if _, err := wb.Do(req); err != nil {
		return fmt.Errorf("disabling phase auto switch failed: %v", err)
	}
	return nil
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

	if res.ExternalControl > warp.ExternalControlAvailable {
		return fmt.Errorf("external control not available: %s", res.ExternalControl.String())
	}

	uri := fmt.Sprintf("%s/power_manager/external_control", wb.emURI)
	data := map[string]int{"phases_wanted": phases}

	req, err := request.New(http.MethodPost, uri, request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}

	_, err = wb.emHelper.Do(req)
	return err
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
