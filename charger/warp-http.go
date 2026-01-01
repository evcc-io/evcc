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
	emHelper        *request.Helper
	log             *util.Logger
	uri             string
	emURI           string
	features        []string
	current         int64
	meterIndex      uint
	metersValuesMap map[int]int
}

func init() {
	registry.Add("warp-http", NewWarpHTTPFromConfig)
}

//go:generate go tool decorate -f decorateWarpHTTP -b *WarpHTTP -r api.Charger -t "api.Meter,CurrentPower,func() (float64, error)" -t "api.MeterEnergy,TotalEnergy,func() (float64, error)" -t "api.PhaseCurrents,Currents,func() (float64, float64, float64, error)" -t "api.PhaseVoltages,Voltages,func() (float64, float64, float64, error)" -t "api.Identifier,Identify,func() (string, error)" -t "api.PhaseSwitcher,Phases1p3p,func(int) error" -t "api.PhaseGetter,GetPhases,func() (int, error)"

// NewWarpHTTPFromConfig creates a new configurable charger
func NewWarpHTTPFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI                    string
		User                   string
		Password               string
		EnergyManagerURI       string
		EnergyManagerUser      string
		EnergyManagerPassword  string
		DisablePhaseAutoSwitch bool
		EnergyMeterIndex       uint
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	wb, err := NewWarpHTTP(cc.URI, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	var currentPower, totalEnergy func() (float64, error)
	var currents, voltages func() (float64, float64, float64, error)
	if wb.hasFeature(wb.uri, warp.FeatureMeters) {
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
	}

	// fallback to meter api
	if (currentPower == nil || totalEnergy == nil) && wb.hasFeature(wb.uri, warp.FeatureMeter) {
		currentPower = wb.meterCurrentPower
		totalEnergy = wb.meterTotalEnergy
		if (currents == nil || voltages == nil) && wb.hasFeature(wb.uri, warp.FeatureMeterPhases) {
			currents = wb.meterCurrents
			voltages = wb.meterVoltages
		}
	}

	var identity func() (string, error)
	if wb.hasFeature(wb.uri, warp.FeatureNfc) {
		identity = wb.identify
	}

	if wb.hasFeature(wb.uri, warp.FeaturePhaseSwitch) {
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
		if res, err := wb.emState(); err == nil && res.ExternalControl != 1 {
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
func NewWarpHTTP(uri, user, password string) (*WarpHTTP, error) {
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
	var res warp.MeterValues
	uri := fmt.Sprintf("%s/meter/values", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.Power, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WarpHTTP) meterTotalEnergy() (float64, error) {
	var res warp.MeterValues
	uri := fmt.Sprintf("%s/meter/values", wb.uri)
	err := wb.GetJSON(uri, &res)
	return res.EnergyAbs, err
}

func (wb *WarpHTTP) meterAllValues() ([]float64, error) {
	var res []float64
	uri := fmt.Sprintf("%s/meter/all_values", wb.uri)
	err := wb.GetJSON(uri, &res)

	if err == nil && len(res) < 6 {
		err = fmt.Errorf("invalid length: %d", len(res))
	}

	return res, err
}

// currents implements the api.MeterCurrrents interface
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

// metersValueIds returns an array of 6 indices mapping meter value IDs to their positions
// in the values array: [VoltageL1N, VoltageL2N, VoltageL3N, CurrentImExSumL1, CurrentImExSumL2, CurrentImExSumL3]
func (wb *WarpHTTP) metersValueIds() (map[int]int, error) {
	var res []int
	uri := fmt.Sprintf("%s/meters/%d/value_ids", wb.uri, wb.meterIndex)
	if err := wb.GetJSON(uri, &res); err != nil {
		return nil, err
	}

	var foundIDs = map[int]bool{
		warp.MetersValueIDVoltageL1N:       false,
		warp.MetersValueIDVoltageL2N:       false,
		warp.MetersValueIDVoltageL3N:       false,
		warp.MetersValueIDCurrentImExSumL1: false,
		warp.MetersValueIDCurrentImExSumL2: false,
		warp.MetersValueIDCurrentImExSumL3: false,
		warp.MetersValueIDPowerImExSum:     false,
		warp.MetersValueIDEnergyAbsImExSum: false,
	}

	var indices = make(map[int]int)
	for i, valueIdx := range res {
		if _, exists := foundIDs[valueIdx]; exists {
			foundIDs[valueIdx] = true
			indices[valueIdx] = i
		}
	}

	// Check if all required IDs were found
	for id, found := range foundIDs {
		if !found {
			return nil, fmt.Errorf("missing required meter value ID: %d", id)
		}
	}

	return indices, nil
}

func (wb *WarpHTTP) metersValues() (warp.MetersValues, error) {
	var res []float64
	metersValues := warp.MetersValues{}
	uri := fmt.Sprintf("%s/meters/%d/values", wb.uri, wb.meterIndex)
	err := wb.GetJSON(uri, &res)

	if err != nil {
		return metersValues, err
	}

	if idx, ok := wb.metersValuesMap[warp.MetersValueIDVoltageL1N]; !ok {
		return metersValues, fmt.Errorf("voltage L1N value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("voltage L1N index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.VoltageL1N = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDVoltageL2N]; !ok {
		return metersValues, fmt.Errorf("voltage L2N value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("voltage L2N index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.VoltageL2N = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDVoltageL3N]; !ok {
		return metersValues, fmt.Errorf("voltage L3N value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("voltage L3N index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.VoltageL3N = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDCurrentImExSumL1]; !ok {
		return metersValues, fmt.Errorf("current L1 value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("current L1 index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.CurrentImExSumL1 = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDCurrentImExSumL2]; !ok {
		return metersValues, fmt.Errorf("current L2 value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("current L2 index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.CurrentImExSumL2 = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDCurrentImExSumL3]; !ok {
		return metersValues, fmt.Errorf("current L3 value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("current L3 index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.CurrentImExSumL3 = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDPowerImExSum]; !ok {
		return metersValues, fmt.Errorf("power value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("power index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.PowerImExSum = res[idx]
	}
	if idx, ok := wb.metersValuesMap[warp.MetersValueIDEnergyAbsImExSum]; !ok {
		return metersValues, fmt.Errorf("energy value ID not found")
	} else if idx >= len(res) {
		return metersValues, fmt.Errorf("energy index out of range: idx=%d, len(values)=%d", idx, len(res))
	} else {
		metersValues.EnergyAbsImExSum = res[idx]
	}

	return metersValues, nil
}

// CurrentPower implements the api.Meter interface
func (wb *WarpHTTP) metersCurrentPower() (float64, error) {
	values, err := wb.metersValues()

	return values.PowerImExSum, err
}

// TotalEnergy implements the api.MeterEnergy interface
func (wb *WarpHTTP) metersTotalEnergy() (float64, error) {
	values, err := wb.metersValues()

	return values.EnergyAbsImExSum, err
}

// currents implements the api.PhaseCurrrents interface
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

	if res.ExternalControl > 0 {
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
