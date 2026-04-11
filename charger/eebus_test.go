package charger

import (
	"errors"
	"testing"
	"time"

	ucapi "github.com/enbility/eebus-go/usecases/api"
	evcemuc "github.com/enbility/eebus-go/usecases/cem/evcem"
	"github.com/enbility/eebus-go/usecases/mocks"
	spinemocks "github.com/enbility/spine-go/mocks"
	"github.com/enbility/spine-go/model"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// Test measurements updated after writing limits detction works
func TestEEBusNoCurrents(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	evcem := mocks.NewCemEVCEMInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC:  evcc,
			EvCem: evcem,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)

	// limit set 15:04:45, measurement receviced afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 4, 45, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.5, 10.5, 10.5}, nil).Once()

	l1, l2, l3, err := eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.5, l1)
	assert.Equal(t, 10.5, l2)
	assert.Equal(t, 10.5, l3)

	// limit set 15:05:09, measurement receviced afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 5, 9, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{6.6, 6.6, 6.6}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 6.6, l1)
	assert.Equal(t, 6.6, l2)
	assert.Equal(t, 6.6, l3)

	// limit set 15:05:39, measurement received afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 5, 39, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.4, 10.5, 10.4}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.4, l1)
	assert.Equal(t, 10.5, l2)
	assert.Equal(t, 10.4, l3)

	// limit set 15:06:09, measurement received afterwards before calling currents
	eebus.limitUpdated = time.Date(2024, 9, 16, 15, 6, 9, 0, time.UTC)
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.4, 10.4, 10.4}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.4, l1)
	assert.Equal(t, 10.4, l2)
	assert.Equal(t, 10.4, l3)

	// limit set 20 seconds ago, no measurement received yet
	eebus.limitUpdated = time.Now().Add(-20 * time.Second)

	l1, l2, l3, err = eebus.currents()
	require.Error(t, err)
	assert.Equal(t, 0.0, l1)
	assert.Equal(t, 0.0, l2)
	assert.Equal(t, 0.0, l3)

	// now we got a measurement again
	eebus.UseCaseEvent(nil, evEntity, evcemuc.DataUpdateCurrentPerPhase)

	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{10.4, 10.4, 10.4}, nil).Once()

	l1, l2, l3, err = eebus.currents()
	require.NoError(t, err)
	assert.Equal(t, 10.4, l1)
	assert.Equal(t, 10.4, l2)
	assert.Equal(t, 10.4, l3)
}

// 3-phase limits helper
func limits3p(v float64) []float64 {
	return []float64{v, v, v}
}

// TestComputeOpevLimits verifies the per-phase OpEV obligation-max tuples that
// writeCurrentLimitData ships through the combined-write path. OpEV is active
// for the given current unless it meets/exceeds the phase max (in which case
// the obligation becomes inactive, meaning "no cap").
func TestComputeOpevLimits(t *testing.T) {
	tests := []struct {
		name       string
		current    float64
		maxLimits  []float64
		wantActive bool
		wantValue  float64
	}{
		{"mid-range", 10, limits3p(16), true, 10},
		{"at max", 16, limits3p(16), false, 16},
		{"above max", 20, limits3p(16), false, 20},
		{"disable", 0, limits3p(16), true, 0},
		{"no max limits", 10, nil, true, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeOpevLimits(tc.current, tc.maxLimits)
			require.Len(t, got, 3)
			for i, limit := range got {
				assert.Equalf(t, tc.wantActive, limit.IsActive, "phase %d IsActive", i)
				assert.Equalf(t, tc.wantValue, limit.Value, "phase %d Value", i)
			}
		})
	}
}

// TestComputeOscevLimits verifies the per-phase OSCEV recommendation tuples.
// Unlike OpEV, OSCEV is only active when the current is at least the phase
// min — a recommendation below min cannot drive charging and must be inactive.
func TestComputeOscevLimits(t *testing.T) {
	tests := []struct {
		name       string
		current    float64
		minLimits  []float64
		wantActive bool
		wantValue  float64
	}{
		{"at min", 6, limits3p(6), true, 6},
		{"above min", 10, limits3p(6), true, 10},
		{"below min", 4, limits3p(6), false, 4},
		{"disable", 0, limits3p(6), false, 0},
		{"no min limits", 10, nil, false, 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeOscevLimits(tc.current, tc.minLimits)
			require.Len(t, got, 3)
			for i, limit := range got {
				assert.Equalf(t, tc.wantActive, limit.IsActive, "phase %d IsActive", i)
				assert.Equalf(t, tc.wantValue, limit.Value, "phase %d Value", i)
			}
		})
	}
}

// TestFindLimitDescriptionByMeasurementID pins the per-phase description
// lookup that buildPhaseLimitData uses to map a parameter's MeasurementId to
// the matching LoadControl limit description. Descriptions with a nil
// MeasurementId must be skipped; first match wins; no match returns nil.
func TestFindLimitDescriptionByMeasurementID(t *testing.T) {
	id := func(v model.MeasurementIdType) *model.MeasurementIdType { return &v }
	limitID := func(v model.LoadControlLimitIdType) *model.LoadControlLimitIdType { return &v }

	descs := []model.LoadControlLimitDescriptionDataType{
		{MeasurementId: nil, LimitId: limitID(1)},
		{MeasurementId: id(7), LimitId: limitID(2)},
		{MeasurementId: id(9), LimitId: limitID(3)},
		{MeasurementId: id(9), LimitId: limitID(4)},
	}

	t.Run("empty slice returns nil", func(t *testing.T) {
		assert.Nil(t, findLimitDescriptionByMeasurementID(nil, 9))
	})

	t.Run("no match returns nil", func(t *testing.T) {
		assert.Nil(t, findLimitDescriptionByMeasurementID(descs, 42))
	})

	t.Run("skips nil MeasurementId and returns first match", func(t *testing.T) {
		got := findLimitDescriptionByMeasurementID(descs, 7)
		require.NotNil(t, got)
		require.NotNil(t, got.LimitId)
		assert.Equal(t, model.LoadControlLimitIdType(2), *got.LimitId)
	})

	t.Run("first-match wins on duplicates", func(t *testing.T) {
		got := findLimitDescriptionByMeasurementID(descs, 9)
		require.NotNil(t, got)
		require.NotNil(t, got.LimitId)
		assert.Equal(t, model.LoadControlLimitIdType(3), *got.LimitId)
	})
}

// TestOscevMinLimits pins the three skip conditions and the happy path of
// the OSCEV data-availability check that writeCurrentLimitData performs
// before including recommendation limits in the combined write.
func TestOscevMinLimits(t *testing.T) {
	t.Run("scenario not available returns false", func(t *testing.T) {
		oscev := mocks.NewCemOSCEVInterface(t)
		evEntity := spinemocks.NewEntityRemoteInterface(t)
		oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(false)

		limits, ok := oscevMinLimits(oscev, evEntity)
		assert.False(t, ok)
		assert.Nil(t, limits)
	})

	t.Run("LoadControlLimits error returns false", func(t *testing.T) {
		oscev := mocks.NewCemOSCEVInterface(t)
		evEntity := spinemocks.NewEntityRemoteInterface(t)
		oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
		oscev.EXPECT().LoadControlLimits(evEntity).Return(nil, errors.New("no data"))

		limits, ok := oscevMinLimits(oscev, evEntity)
		assert.False(t, ok)
		assert.Nil(t, limits)
	})

	t.Run("CurrentLimits error returns false", func(t *testing.T) {
		oscev := mocks.NewCemOSCEVInterface(t)
		evEntity := spinemocks.NewEntityRemoteInterface(t)
		oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
		oscev.EXPECT().LoadControlLimits(evEntity).Return(nil, nil)
		oscev.EXPECT().CurrentLimits(evEntity).Return(nil, nil, nil, errors.New("no limits"))

		limits, ok := oscevMinLimits(oscev, evEntity)
		assert.False(t, ok)
		assert.Nil(t, limits)
	})

	t.Run("happy path returns min limits", func(t *testing.T) {
		oscev := mocks.NewCemOSCEVInterface(t)
		evEntity := spinemocks.NewEntityRemoteInterface(t)
		want := limits3p(6)
		oscev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(true)
		oscev.EXPECT().LoadControlLimits(evEntity).Return(nil, nil)
		oscev.EXPECT().CurrentLimits(evEntity).Return(want, limits3p(16), limits3p(10), nil)

		limits, ok := oscevMinLimits(oscev, evEntity)
		assert.True(t, ok)
		assert.Equal(t, want, limits)
	})
}

// TestWriteCurrentLimitData_OpevNotAvailable verifies that writeCurrentLimitData
// returns ErrNotAvailable without touching any clients when OpEV scenario 1 is
// not supported on the target entity.
func TestWriteCurrentLimitData_OpevNotAvailable(t *testing.T) {
	opev := mocks.NewCemOPEVInterface(t)
	evEntity := spinemocks.NewEntityRemoteInterface(t)

	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			OpEV: opev,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	opev.EXPECT().IsScenarioAvailableAtEntity(evEntity, uint(1)).Return(false)

	err := eebus.writeCurrentLimitData(evEntity, 10)
	require.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestEnabledAlwaysReadsOpev(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	opev := mocks.NewCemOPEVInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC: evcc,
			OpEV: opev,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	opev.EXPECT().LoadControlLimits(evEntity).Return([]ucapi.LoadLimitsPhase{
		{Phase: model.ElectricalConnectionPhaseNameTypeA, IsActive: true, Value: 10},
		{Phase: model.ElectricalConnectionPhaseNameTypeB, IsActive: true, Value: 10},
		{Phase: model.ElectricalConnectionPhaseNameTypeC, IsActive: true, Value: 10},
	}, nil)

	enabled, err := eebus.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestEEBusIsCharging(t *testing.T) {
	type limitStruct struct {
		min, max, pause float64
	}

	type testMeasurementStruct struct {
		charging bool
		currents []float64
		powers   []float64
	}

	tests := []struct {
		name         string
		limits       []limitStruct
		measurements []testMeasurementStruct
	}{
		{
			"3 phase IEC",
			[]limitStruct{
				{6, 16, 0},
				{6, 16, 0},
				{6, 16, 0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]float64{0, 3, 0},
					[]float64{0, 690, 0},
				},
				{
					true,
					[]float64{6, 0, 1},
					[]float64{1380, 0, 230},
				},
			},
		},
		{
			"1 phase IEC",
			[]limitStruct{
				{6, 16, 0},
			},
			[]testMeasurementStruct{
				{
					false,
					[]float64{2},
					[]float64{460},
				},
				{
					true,
					[]float64{6},
					[]float64{1380},
				},
			},
		},
		{
			"3 phase ISO",
			[]limitStruct{
				{2.2, 16, 0.1},
				{2.2, 16, 0.1},
				{2.2, 16, 0.1},
			},
			[]testMeasurementStruct{
				{
					false,
					[]float64{1, 0, 0},
					[]float64{230, 0, 0},
				},
				{
					true,
					[]float64{1.8, 1, 3},
					[]float64{414, 230, 690},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var limitsMin, limitsMax, limitsDefault []float64

			for _, limit := range tc.limits {
				limitsMin = append(limitsMin, limit.min)
				limitsMax = append(limitsMax, limit.max)
				limitsDefault = append(limitsDefault, limit.pause)
			}

			for _, m := range tc.measurements {
				ctrl := gomock.NewController(t)

				evcc := mocks.NewCemEVCCInterface(t)
				evcem := mocks.NewCemEVCEMInterface(t)
				opev := mocks.NewCemOPEVInterface(t)

				evEntity := spinemocks.NewEntityRemoteInterface(t)
				eebus := &EEBus{
					cem: &eebus.CustomerEnergyManagement{
						EvCC:  evcc,
						EvCem: evcem,
						OpEV:  opev,
					},
					ev: evEntity,
				}

				evcc.EXPECT().EVConnected(evEntity).Return(true)
				evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)
				evcem.EXPECT().PowerPerPhase(evEntity).Return(m.powers, nil)
				opev.EXPECT().CurrentLimits(evEntity).Return(limitsMin, limitsMax, limitsDefault, nil)

				require.Equal(t, m.charging, eebus.isCharging(evEntity))

				ctrl.Finish()
			}
		})
	}
}

func TestEEBusCurrentPower(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	evcem := mocks.NewCemEVCEMInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC:  evcc,
			EvCem: evcem,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)
	evcem.EXPECT().PowerPerPhase(evEntity).Return([]float64{600, 600, 600}, nil)

	power, err := eebus.currentPower()
	require.NoError(t, err)
	assert.Equal(t, 1800.0, power)
}

func TestEEBusCurrentPower_Elli(t *testing.T) {
	evcc := mocks.NewCemEVCCInterface(t)
	evcem := mocks.NewCemEVCEMInterface(t)

	evEntity := spinemocks.NewEntityRemoteInterface(t)
	eebus := &EEBus{
		cem: &eebus.CustomerEnergyManagement{
			EvCC:  evcc,
			EvCem: evcem,
		},
		ev:  evEntity,
		log: util.NewLogger("test"),
	}

	evcc.EXPECT().EVConnected(evEntity).Return(true)
	evcem.EXPECT().IsScenarioAvailableAtEntity(evEntity, mock.Anything).Return(true)
	evcem.EXPECT().PowerPerPhase(evEntity).Return(nil, errors.New("error"))
	evcem.EXPECT().CurrentPerPhase(evEntity).Return([]float64{5.8, 5.8, 5.8}, nil)

	power, err := eebus.currentPower()
	require.NoError(t, err)
	assert.Equal(t, 4002.0, power)
}
