package mercedes

import (
	"errors"
	"github.com/evcc-io/evcc/util"
	"math"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snapshotProvider(state *VehicleStatusUpdate) *Provider {
	return &Provider{stateG: func() (*VehicleStatusUpdate, error) { return state, nil }}
}

func TestOdometer(t *testing.T) {
	for _, tc := range []struct {
		name string
		attr *IntegerAttribute
		want float64
		fail bool
	}{
		{"kilometers", &IntegerAttribute{Value: 12408, Unit: 1, Metadata: &AttributeMetadata{}}, 12408, false},
		{"miles", &IntegerAttribute{Value: 100, Unit: 2, Metadata: &AttributeMetadata{}}, 160.9344, false},
		{"unknown unit", &IntegerAttribute{Value: 100, Metadata: &AttributeMetadata{}}, 0, true},
		{"negative", &IntegerAttribute{Value: -1, Unit: 1, Metadata: &AttributeMetadata{}}, 0, true},
		{"invalid", &IntegerAttribute{Value: 100, Unit: 1, Metadata: &AttributeMetadata{Status: 3}}, 0, true},
		{"missing", nil, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := snapshotProvider(&VehicleStatusUpdate{Odometer: tc.attr}).Odometer()
			assert.InDelta(t, tc.want, got, 0.000001)
			assert.Equal(t, tc.fail, err != nil)
		})
	}
}

func TestLimitSoc(t *testing.T) {
	for _, value := range []int64{-1, 0, 1, 80, 100, 101} {
		got, err := snapshotProvider(&VehicleStatusUpdate{MaxSoc: &IntegerAttribute{Value: value, Metadata: &AttributeMetadata{}}}).GetLimitSoc()
		if value == 0 {
			assert.ErrorIs(t, err, api.ErrNotAvailable)
			assert.Zero(t, got)
		} else if value < 0 || value > 100 {
			require.Error(t, err)
			assert.NotErrorIs(t, err, api.ErrNotAvailable)
			assert.Zero(t, got)
		} else {
			require.NoError(t, err)
			assert.Equal(t, value, got)
		}
	}
	_, err := snapshotProvider(&VehicleStatusUpdate{}).GetLimitSoc()
	assert.ErrorIs(t, err, api.ErrMustRetry)
}

func TestClimater(t *testing.T) {
	for _, value := range []bool{false, true} {
		got, err := snapshotProvider(&VehicleStatusUpdate{PreconditioningActive: &BooleanAttribute{Value: value, Metadata: &AttributeMetadata{}}}).Climater()
		require.NoError(t, err)
		assert.Equal(t, value, got)
	}
	for _, attr := range []*BooleanAttribute{nil, {}, {Value: true, Metadata: &AttributeMetadata{Status: 4}}} {
		_, err := snapshotProvider(&VehicleStatusUpdate{PreconditioningActive: attr}).Climater()
		assert.ErrorIs(t, err, api.ErrMustRetry)
	}
}

func TestPosition(t *testing.T) {
	for _, tc := range []struct {
		lat, lon float64
		fail     bool
	}{
		{0, 0, false}, {52, 13, false}, {-90, -180, false},
		{91, 13, true}, {52, 181, true}, {math.NaN(), 13, true}, {52, math.Inf(1), true},
	} {
		gotLat, gotLon, err := snapshotProvider(&VehicleStatusUpdate{
			Latitude:  &DoubleAttribute{Value: tc.lat, Metadata: &AttributeMetadata{}},
			Longitude: &DoubleAttribute{Value: tc.lon, Metadata: &AttributeMetadata{}},
		}).Position()
		assert.Equal(t, tc.fail, err != nil)
		if !tc.fail {
			assert.Equal(t, tc.lat, gotLat)
			assert.Equal(t, tc.lon, gotLon)
		}
	}
	for _, state := range []*VehicleStatusUpdate{
		{}, {Latitude: &DoubleAttribute{Metadata: &AttributeMetadata{}}},
		{Latitude: &DoubleAttribute{Metadata: &AttributeMetadata{}}, Longitude: &DoubleAttribute{Metadata: &AttributeMetadata{Status: 1}}},
	} {
		_, _, err := snapshotProvider(state).Position()
		assert.ErrorIs(t, err, api.ErrMustRetry)
	}
}

func TestSnapshotFailure(t *testing.T) {
	failure := errors.New("snapshot failed")
	v := &Provider{stateG: func() (*VehicleStatusUpdate, error) { return nil, failure }}
	_, err := v.Status()
	assert.ErrorIs(t, err, failure)
	_, err = v.Odometer()
	assert.ErrorIs(t, err, failure)
	_, err = v.GetLimitSoc()
	assert.ErrorIs(t, err, failure)
	_, err = v.Climater()
	assert.ErrorIs(t, err, failure)
	_, _, err = v.Position()
	assert.ErrorIs(t, err, failure)
}

func TestInvalidMetadataRefreshesAfterAccountInterval(t *testing.T) {
	for _, tc := range []struct {
		name string
		get  func(*Provider) error
	}{
		{"status", func(v *Provider) error { _, err := v.Status(); return err }},
		{"odometer", func(v *Provider) error { _, err := v.Odometer(); return err }},
		{"limit", func(v *Provider) error { _, err := v.GetLimitSoc(); return err }},
		{"climate", func(v *Provider) error { _, err := v.Climater(); return err }},
		{"position", func(v *Provider) error { _, _, err := v.Position(); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var account vehicleStateCache
			calls := 0
			state := &VehicleStatusUpdate{}
			fetch := func([]string) (map[string]*VehicleStatusUpdate, error) {
				calls++
				return map[string]*VehicleStatusUpdate{"car": state}, nil
			}
			cache := util.ResettableCached(func() (*VehicleStatusUpdate, error) { return account.get("car", fetch) }, time.Hour)
			v := &Provider{stateG: cache.Get, resetState: cache.Reset}
			for range 2 {
				assert.ErrorIs(t, tc.get(v), api.ErrMustRetry)
			}
			assert.Equal(t, 1, calls, "metadata errors must not bypass the account rate limit")
			state = &VehicleStatusUpdate{
				ChargingStatus:        &IntegerAttribute{Value: 3, Metadata: &AttributeMetadata{}},
				Odometer:              &IntegerAttribute{Value: 123, Unit: 1, Metadata: &AttributeMetadata{}},
				MaxSoc:                &IntegerAttribute{Value: 80, Metadata: &AttributeMetadata{}},
				PreconditioningActive: &BooleanAttribute{Metadata: &AttributeMetadata{}},
				Latitude:              &DoubleAttribute{Metadata: &AttributeMetadata{}},
				Longitude:             &DoubleAttribute{Metadata: &AttributeMetadata{}},
			}
			account.updated = time.Now().Add(-vehicleStatusInterval)
			require.NoError(t, tc.get(v), "recover without waiting for the hour-long provider cache")
			assert.Equal(t, 2, calls)
		})
	}
}
