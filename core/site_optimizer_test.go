package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestLoadpointProfile(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetMode().Return(api.ModeMinPV).AnyTimes()
	lp.EXPECT().GetStatus().Return(api.StatusC).AnyTimes()
	lp.EXPECT().GetChargePower().Return(10000.0).AnyTimes()   //  10 kW
	lp.EXPECT().EffectiveMinPower().Return(1000.0).AnyTimes() //   1 kW
	lp.EXPECT().GetRemainingEnergy().Return(1.8).AnyTimes()   // 1.8 kWh

	// expected slots: 0.25 kWh...
	require.Equal(t, []float64{250, 250, 250, 250, 250, 250, 250, 50}, loadpointProfile(lp, 8))
}

func TestOptimizerHorizon(t *testing.T) {
	ts := time.Date(2025, 1, 1, 10, 30, 0, 0, time.Local)
	horizon := optimizerHorizon(ts)

	// 48h plus end of day
	assert.Equal(t, time.Date(2025, 1, 3, 23, 59, 59, int(time.Second-time.Nanosecond), time.Local), horizon)

	// before 6:00 the day is not extended
	assert.Equal(t, time.Date(2025, 1, 3, 5, 30, 0, 0, time.Local),
		optimizerHorizon(time.Date(2025, 1, 1, 5, 30, 0, 0, time.Local)))

	rates := make(api.Rates, 0, 4*96)
	for slot := ts.Truncate(tariff.SlotDuration); len(rates) < cap(rates); slot = slot.Add(tariff.SlotDuration) {
		rates = append(rates, api.Rate{Start: slot, End: slot.Add(tariff.SlotDuration)})
	}

	// 4 days of slots from 10:30, capped at the last slot of Jan 3rd
	assert.Equal(t, 246, slotsUntil(rates, horizon, len(rates)))
	assert.Equal(t, time.Date(2025, 1, 3, 23, 45, 0, 0, time.Local), rates[245].Start)

	// shorter forecast is not extended
	assert.Equal(t, 8, slotsUntil(rates, horizon, 8))
}

func TestApplyPrecondition(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().EffectiveMaxPower().Return(8000.0).AnyTimes() // 2 kWh per slot

	// no precondition configured
	lp.EXPECT().EffectivePlanStrategy().Return(api.PlanStrategy{}).Times(1)
	assert.Nil(t, applyPrecondition(lp, nil, 8))

	// no plan
	lp.EXPECT().EffectivePlanStrategy().Return(api.PlanStrategy{Precondition: time.Hour}).Times(1)
	lp.EXPECT().EffectivePlanTime().Return(time.Time{}).Times(1)
	assert.Nil(t, applyPrecondition(lp, nil, 8))

	// plan in 1h, 40min precondition: slots 1 (10min) and 2, 3 (full)
	lp.EXPECT().EffectivePlanTime().Return(time.Now().Add(time.Hour)).Times(1)
	lp.EXPECT().EffectivePlanStrategy().Return(api.PlanStrategy{Precondition: 40 * time.Minute}).Times(1)
	res := applyPrecondition(lp, nil, 8)
	require.Len(t, res, 8)
	assert.InDeltaSlice(t, []float32{0, 2000. / 1.5, 2000, 2000, 0, 0, 0, 0}, res, 1)

	// existing demand is kept where higher
	lp.EXPECT().EffectivePlanTime().Return(time.Now().Add(time.Hour)).Times(1)
	lp.EXPECT().EffectivePlanStrategy().Return(api.PlanStrategy{Precondition: 30 * time.Minute}).Times(1)
	res = applyPrecondition(lp, []float32{3000, 3000, 3000, 3000, 0, 0, 0, 0}, 8)
	assert.InDeltaSlice(t, []float32{3000, 3000, 3000, 3000, 0, 0, 0, 0}, res, 1)

	// plan beyond horizon
	lp.EXPECT().EffectivePlanTime().Return(time.Now().Add(24 * time.Hour)).Times(1)
	lp.EXPECT().EffectivePlanStrategy().Return(api.PlanStrategy{Precondition: time.Hour}).Times(1)
	assert.Nil(t, applyPrecondition(lp, nil, 8))
}

func TestLoadpointCurrentAction(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		status  api.ChargeStatus
		soc     float64
		want    string
	}{
		{"charging", true, api.StatusC, 0, actionCharge},
		{"enabled but idle (e.g. vehicle finished at limit)", true, api.StatusB, 0, actionStop},
		{"disabled", false, api.StatusB, 0, actionStop},
		{"charging at 100% soc with no explicit limit", true, api.StatusC, 100, actionStop},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lp := &Loadpoint{enabled: tc.enabled, status: tc.status, vehicleSoc: tc.soc}
			assert.Equal(t, tc.want, loadpointCurrentAction(lp))
		})
	}
}

func TestAsTimestamps(t *testing.T) {
	// now is 10 minutes into a 15-minute slot
	now := time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC)

	// dt[0]=300 means first event is 300s (5min) before end of current slot
	// dt[1..] just mark subsequent slot boundaries
	dt := []int{60 * 5, 60 * 15, 60 * 15}

	got := asTimestamps(dt, now)

	// current slot: 12:00–12:15
	// first timestamp: 12:15 - 5min = 12:10
	// subsequent: 12:15, 12:30
	assert.Equal(t, []time.Time{
		time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 15, 0, 0, time.UTC),
		time.Date(2025, 1, 1, 12, 30, 0, 0, time.UTC),
	}, got)
}

func TestUnmodelledPower(t *testing.T) {
	ctrl := gomock.NewController(t)

	for _, tc := range []struct {
		name            string
		mode            api.ChargeMode
		status          api.ChargeStatus
		power, minPower float64
		expected        float64
	}{
		{"pv charging", api.ModePV, api.StatusC, 4000, 1380, 4000},
		{"pv connected", api.ModePV, api.StatusB, 0, 1380, 0},
		{"minpv floor before meter caught up", api.ModeMinPV, api.StatusC, 0, 4000, 4000},
		{"minpv floor must not lower measured", api.ModeMinPV, api.StatusC, 4000, 1000, 4000},
		{"minpv floor only applies while charging", api.ModeMinPV, api.StatusB, 0, 4000, 0},
		{"negative measurement clamped", api.ModePV, api.StatusC, -100, 0, 0},
	} {
		lp := loadpoint.NewMockAPI(ctrl)
		lp.EXPECT().GetMode().Return(tc.mode).AnyTimes()
		lp.EXPECT().GetStatus().Return(tc.status).AnyTimes()
		lp.EXPECT().GetChargePower().Return(tc.power).AnyTimes()
		lp.EXPECT().EffectiveMinPower().Return(tc.minPower).AnyTimes()

		assert.Equal(t, tc.expected, unmodelledPower(lp), tc.name)
	}
}

func TestBatteryForecastSocExtremes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		req       []optimizer.BatteryConfig
		soc       [][]float32
		high, low *batteryForecastSlot
	}{
		{
			"no home battery",
			[]optimizer.BatteryConfig{{SMax: 80}}, // SCapacity unset → vehicle
			[][]float32{{1000, 2000}},
			nil, nil,
		},
		{
			"single home battery rising — reaches full",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 1000}},
			[][]float32{{200, 500, 1000}},
			&batteryForecastSlot{slot: 2, soc: 100, limit: true},
			&batteryForecastSlot{slot: 0, soc: 20, limit: false},
		},
		{
			"single home battery falling — reaches empty",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 1000}},
			[][]float32{{900, 500, 0}},
			&batteryForecastSlot{slot: 0, soc: 90, limit: false},
			&batteryForecastSlot{slot: 2, soc: 0, limit: true},
		},
		{
			"single home battery — local extremes (no limit reached)",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 900, SMin: 100}},
			[][]float32{{500, 800, 200}},
			&batteryForecastSlot{slot: 1, soc: 80, limit: false},
			&batteryForecastSlot{slot: 2, soc: 20, limit: false},
		},
		{
			"two home batteries aggregated",
			[]optimizer.BatteryConfig{
				{SCapacity: 1000, SMax: 1000},
				{SCapacity: 1000, SMax: 1000},
			},
			[][]float32{
				{200, 400, 1000},
				{800, 400, 1000},
			},
			&batteryForecastSlot{slot: 2, soc: 100, limit: true},
			&batteryForecastSlot{slot: 1, soc: 40, limit: false},
		},
		{
			"vehicle and home battery — vehicle ignored",
			[]optimizer.BatteryConfig{
				{SMax: 80},                    // vehicle
				{SCapacity: 1000, SMax: 1000}, // home
			},
			[][]float32{
				{0, 0, 80},
				{200, 500, 900},
			},
			&batteryForecastSlot{slot: 2, soc: 90, limit: false},
			&batteryForecastSlot{slot: 0, soc: 20, limit: false},
		},
		{
			"first slot at SMax wins for highest",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 1000}},
			[][]float32{{500, 1000, 1000}},
			&batteryForecastSlot{slot: 1, soc: 100, limit: true},
			&batteryForecastSlot{slot: 0, soc: 50, limit: false},
		},
		{
			"already full — no highest",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 1000}},
			[][]float32{{1000, 1000, 500}},
			nil,
			&batteryForecastSlot{slot: 2, soc: 50, limit: false},
		},
		{
			"already empty — no lowest",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 1000, SMin: 100}},
			[][]float32{{100, 100, 500}},
			&batteryForecastSlot{slot: 2, soc: 50, limit: false},
			nil,
		},
		{
			"near SMax is not full",
			[]optimizer.BatteryConfig{{SCapacity: 1000, SMax: 1000}},
			[][]float32{{500, 999, 800}},
			&batteryForecastSlot{slot: 1, soc: 99.9, limit: false},
			&batteryForecastSlot{slot: 0, soc: 50, limit: false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := make([]optimizer.BatteryResult, len(tc.soc))
			for i, s := range tc.soc {
				resp[i] = optimizer.BatteryResult{StateOfCharge: s}
			}

			high, low := batteryForecastSocExtremes(tc.req, resp)

			if tc.high == nil {
				assert.Nil(t, high, "high")
			} else {
				require.NotNil(t, high, "high")
				assert.Equal(t, tc.high.slot, high.slot, "high.slot")
				assert.InDelta(t, tc.high.soc, high.soc, 1e-3, "high.soc")
				assert.Equal(t, tc.high.limit, high.limit, "high.limit")
			}
			if tc.low == nil {
				assert.Nil(t, low, "low")
			} else {
				require.NotNil(t, low, "low")
				assert.Equal(t, tc.low.slot, low.slot, "low.slot")
				assert.InDelta(t, tc.low.soc, low.soc, 1e-3, "low.soc")
				assert.Equal(t, tc.low.limit, low.limit, "low.limit")
			}
		})
	}
}

// TestBatteryRequestSocLimitsClamp ensures the reported soc is always clamped into
// the resulting [SMin, SMax] range, even when it lies outside the configured soc
// limits (e.g. right after a firmware update changed the reported soc or the min/max
// soc settings) - otherwise the optimizer is infeasible from the first slot.
func TestBatteryRequestSocLimitsClamp(t *testing.T) {
	newBatteryDevice := func(t *testing.T, minSoc, maxSoc float64) config.Device[api.Meter] {
		ctrl := gomock.NewController(t)

		var meter api.Meter
		batSocLimit := api.NewMockBatterySocLimiter(ctrl)
		batSocLimit.EXPECT().GetSocLimits().Return(minSoc, maxSoc).AnyTimes()

		bat := &struct {
			api.Meter
			api.BatterySocLimiter
		}{
			Meter:             meter,
			BatterySocLimiter: batSocLimit,
		}

		return config.NewStaticDevice(config.Named{}, api.Meter(bat))
	}

	site := &Site{log: util.NewLogger("foo")}
	capacity := 10.0 // kWh

	t.Run("soc below minSoc", func(t *testing.T) {
		soc := 15.0
		dev := newBatteryDevice(t, 20, 100)
		m := types.Measurement{Capacity: &capacity, Soc: &soc}

		req, _ := site.batteryRequest(dev, m, nil, 8, 15*time.Minute)

		assert.Equal(t, float32(1500), req.SMin)
		assert.Equal(t, float32(10000), req.SMax)
		assert.LessOrEqual(t, req.SMin, req.SInitial)
	})

	t.Run("soc above maxSoc", func(t *testing.T) {
		soc := 95.0
		dev := newBatteryDevice(t, 0, 80)
		m := types.Measurement{Capacity: &capacity, Soc: &soc}

		req, _ := site.batteryRequest(dev, m, nil, 8, 15*time.Minute)

		assert.Equal(t, float32(0), req.SMin)
		assert.Equal(t, float32(9500), req.SMax)
		assert.GreaterOrEqual(t, req.SMax, req.SInitial)
	})

	t.Run("soc within limits", func(t *testing.T) {
		soc := 50.0
		dev := newBatteryDevice(t, 20, 80)
		m := types.Measurement{Capacity: &capacity, Soc: &soc}

		req, _ := site.batteryRequest(dev, m, nil, 8, 15*time.Minute)

		assert.Equal(t, float32(2000), req.SMin)
		assert.Equal(t, float32(8000), req.SMax)
	})

	t.Run("empty maxSoc defaults to 100%", func(t *testing.T) {
		soc := 50.0
		dev := newBatteryDevice(t, 20, 0)
		m := types.Measurement{Capacity: &capacity, Soc: &soc}

		req, _ := site.batteryRequest(dev, m, nil, 8, 15*time.Minute)

		assert.Equal(t, float32(2000), req.SMin)
		assert.Equal(t, float32(10000), req.SMax)
	})
}

// charge goal for vehicles with and without known capacity/soc, see #32890
func TestLoadpointRequestChargeGoal(t *testing.T) {
	site := &Site{log: util.NewLogger("foo")}

	for _, tc := range []struct {
		name                  string
		capacity, soc         float64 // kWh, percent
		limitSoc              int     // percent
		limitEnergy, charged  float64 // kWh, Wh
		wantInitial, wantSMax float32 // Wh
	}{
		{"soc limit", 50, 20, 80, 0, 0, 10000, 40000},
		{"no capacity, energy limit", 0, 0, 100, 10, 0, 0, 10000},
		{"no capacity, energy limit partially charged", 0, 0, 100, 10, 4000, 4000, 10000},
		{"no capacity, limit exceeded", 0, 0, 100, 10, 11000, 11000, 11000},
		{"capacity but no soc, energy limit", 50, 0, 100, 10, 0, 0, 10000},
		{"capacity but no soc, no energy limit", 50, 0, 100, 0, 0, 0, 50000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			v := api.NewMockVehicle(ctrl)
			v.EXPECT().Capacity().Return(tc.capacity).AnyTimes()
			v.EXPECT().GetTitle().Return("").AnyTimes()

			lp := loadpoint.NewMockAPI(ctrl)
			lp.EXPECT().GetVehicle().Return(v).AnyTimes()
			lp.EXPECT().GetSoc().Return(tc.soc).AnyTimes()
			lp.EXPECT().EffectiveLimitSoc().Return(tc.limitSoc).AnyTimes()
			lp.EXPECT().GetLimitEnergy().Return(tc.limitEnergy).AnyTimes()
			lp.EXPECT().GetChargedEnergy().Return(tc.charged).AnyTimes()
			lp.EXPECT().GetTitle().Return("lp").AnyTimes()
			lp.EXPECT().EffectiveMinPower().Return(1380.0).AnyTimes()
			lp.EXPECT().EffectiveMaxPower().Return(11000.0).AnyTimes()
			lp.EXPECT().GetMode().Return(api.ModePV).AnyTimes()
			lp.EXPECT().GetStatus().Return(api.StatusB).AnyTimes()
			lp.EXPECT().GetSmartCostLimit().Return(nil).AnyTimes()
			lp.EXPECT().EffectivePlanStrategy().Return(api.PlanStrategy{}).AnyTimes()
			lp.EXPECT().GetPlanGoal().Return(0.0, false).AnyTimes()

			req, _ := site.loadpointRequest(lp, 8, 15*time.Minute, nil)

			assert.Equal(t, tc.wantInitial, req.SInitial)
			assert.Equal(t, tc.wantSMax, req.SMax)
		})
	}
}

func TestOptimizerChargingStrategy(t *testing.T) {
	site := &Site{log: util.NewLogger("foo")}

	// default when unset
	assert.Equal(t, defaultOptimizerChargingStrategy, site.GetOptimizerChargingStrategy())

	// invalid value rejected, strategy unchanged
	require.Error(t, site.SetOptimizerChargingStrategy("bogus"))
	assert.Equal(t, defaultOptimizerChargingStrategy, site.GetOptimizerChargingStrategy())

	// valid change is applied (re-trigger is gated on sponsor/enabled, not unit-tested here)
	require.NoError(t, site.SetOptimizerChargingStrategy(string(optimizer.OptimizerStrategyChargingStrategyAttenuateGridPeaks)))
	assert.Equal(t, "attenuate_grid_peaks", site.GetOptimizerChargingStrategy())
}

func TestGridExportLimit(t *testing.T) {
	site := &Site{log: util.NewLogger("foo")}

	// disabled by default
	assert.Equal(t, 0.0, site.GetGridExportLimit())

	// negative value rejected, limit unchanged
	require.Error(t, site.SetGridExportLimit(-1))
	assert.Equal(t, 0.0, site.GetGridExportLimit())

	require.NoError(t, site.SetGridExportLimit(7000))
	assert.Equal(t, 7000.0, site.GetGridExportLimit())
}

func TestBlendMeasured(t *testing.T) {
	slots := []float64{100, 100, 100, 100, 100, 100}
	blendMeasured(slots, 200, 4)
	assert.Equal(t, []float64{200, 175, 150, 125, 100, 100}, slots)

	// fewer slots than decay length
	short := []float32{100, 100}
	blendMeasured(short, 200, 4)
	assert.Equal(t, []float32{200, 175}, short)
}

func TestBlendScale(t *testing.T) {
	slots := []float32{100, 100, 100, 100, 100, 100}
	blendScale(slots, 2, 4)
	assert.Equal(t, []float32{200, 175, 150, 125, 100, 100}, slots)

	// fewer slots than decay length
	short := []float64{100, 100}
	blendScale(short, 0.5, 4)
	assert.Equal(t, []float64{50, 62.5}, short)
}

func TestCurrentSlotSuggestion(t *testing.T) {
	// slotHours 1 makes the per-slot Wh values map 1:1 to W
	for _, tc := range []struct {
		name              string
		typ               batteryType
		charge, disch     float32
		importing, export bool
		want              string
	}{
		{"battery grid charge", batteryTypeBattery, 3000, 0, true, false, "charge"},
		{"battery pv charge (no import)", batteryTypeBattery, 3000, 0, false, true, "normal"},
		{"battery hold (idle while importing)", batteryTypeBattery, 0, 0, true, false, "hold"},
		{"battery holdcharge (idle while exporting)", batteryTypeBattery, 0, 0, false, true, "holdcharge"},
		{"battery discharge (self-consumption while importing)", batteryTypeBattery, 0, 2000, true, false, "normal"},
		{"battery grid discharge (discharge while exporting)", batteryTypeBattery, 0, 2000, false, true, "discharge"},
		{"battery idle balanced", batteryTypeBattery, 0, 0, false, false, "normal"},
		{"loadpoint charge", batteryTypeLoadpoint, 11000, 0, false, false, "charge"},
		{"loadpoint stop", batteryTypeLoadpoint, 0, 0, false, false, "stop"},
		{"vehicle below threshold is stop", batteryTypeVehicle, 40, 0, false, false, "stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := optimizer.BatteryResult{
				ChargingPower:    []float32{tc.charge},
				DischargingPower: []float32{tc.disch},
			}
			s := currentSlotSuggestion(batteryDetail{Type: tc.typ}, res, tc.importing, tc.export, 1)
			assert.Equal(t, tc.want, s.Action)
			assert.InDelta(t, tc.charge, s.Charge, 1e-3)
			assert.InDelta(t, tc.disch, s.Discharge, 1e-3)
		})
	}

	// no result yields an empty suggestion
	assert.Empty(t, currentSlotSuggestion(batteryDetail{Type: batteryTypeBattery}, optimizer.BatteryResult{}, true, false, 1))
}

// TestSuggestionActionable ensures the actionable flag follows the current state
// instead of the state at optimizer run time
func TestSuggestionActionable(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), nil)

	site := &Site{
		batteryMode: api.BatteryNormal,
		loadpoints:  []*Loadpoint{lp},
	}
	site.setSuggestions(map[string]types.Suggestion{
		batteryKey("bat"): {Action: api.BatteryCharge.String()},
		loadpointKey(0):   {Action: actionCharge},
	})

	batterySuggestion := func(name string) *types.Suggestion {
		return site.suggestion(batteryKey(name), site.GetBatteryMode().String())
	}
	loadpointSuggestion := func(id int) *types.Suggestion {
		return site.suggestion(loadpointKey(id), loadpointCurrentAction(lp))
	}

	// battery mode differs from suggestion
	s := batterySuggestion("bat")
	require.NotNil(t, s)
	assert.True(t, s.Actionable)

	site.batteryMode = api.BatteryCharge
	assert.False(t, batterySuggestion("bat").Actionable)

	assert.Nil(t, batterySuggestion("unknown"))

	// loadpoint stopped, suggestion is to charge
	s = loadpointSuggestion(0)
	require.NotNil(t, s)
	assert.True(t, s.Actionable)

	// loadpoint charging matches the suggestion
	lp.enabled = true
	lp.status = api.StatusC
	assert.False(t, loadpointSuggestion(0).Actionable)

	assert.Nil(t, loadpointSuggestion(1))
}

func TestSuggestionEvent(t *testing.T) {
	id := 2

	// battery: no loadpoint id, carries name
	detail := batteryDetail{Type: batteryTypeBattery, Name: "home", Title: "Home"}
	assert.Equal(t, "battery:home", detail.key())

	ev := suggestionEvent(detail, types.Suggestion{Action: api.BatteryCharge.String()})
	assert.Nil(t, ev.Loadpoint)
	assert.Equal(t, evSuggestion, ev.Event)
	assert.Equal(t, api.BatteryCharge.String(), ev.Attributes["suggestionAction"])
	assert.Equal(t, "home", ev.Attributes["suggestionName"])
	assert.Equal(t, "Home", ev.Attributes["suggestionTitle"])

	// loadpoint: carries id, no name
	detail = batteryDetail{Type: batteryTypeVehicle, loadpoint: &id, Title: "Garage"}
	assert.Equal(t, "loadpoint:2", detail.key())

	ev = suggestionEvent(detail, types.Suggestion{Action: actionCharge})
	require.NotNil(t, ev.Loadpoint)
	assert.Equal(t, id, *ev.Loadpoint)
	assert.NotContains(t, ev.Attributes, "suggestionName")

	// vehicle without loadpoint can't act on a suggestion
	assert.Empty(t, batteryDetail{Type: batteryTypeVehicle}.key())
}

func TestDiffSuggestions(t *testing.T) {
	site := &Site{}

	pending := func(s types.Suggestion) map[string]pendingSuggestion {
		ev := suggestionEvent(batteryDetail{loadpoint: new(int)}, s)
		return map[string]pendingSuggestion{"loadpoint:0": {suggestion: s, event: ev}}
	}

	charge := types.Suggestion{Action: actionCharge, Actionable: true}
	stop := types.Suggestion{Action: actionStop, Actionable: true}
	notActionable := types.Suggestion{Action: actionCharge, Actionable: false}

	// first actionable suggestion fires
	assert.Len(t, site.diffSuggestions(pending(charge)), 1)

	// unchanged action does not fire again
	assert.Empty(t, site.diffSuggestions(pending(charge)))

	// changed action fires
	assert.Len(t, site.diffSuggestions(pending(stop)), 1)

	// non-actionable suggestion does not fire and clears tracking so the same
	// action re-notifies when it becomes actionable again
	assert.Empty(t, site.diffSuggestions(pending(notActionable)))
	assert.Len(t, site.diffSuggestions(pending(stop)), 1)

	// vanished device is pruned and re-notifies on return
	assert.Empty(t, site.diffSuggestions(map[string]pendingSuggestion{}))
	assert.Len(t, site.diffSuggestions(pending(stop)), 1)
}
