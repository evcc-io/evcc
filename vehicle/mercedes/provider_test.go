package mercedes

import (
	"errors"
	"math"
	"testing"

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
		if value <= 0 || value > 100 {
			assert.ErrorIs(t, err, api.ErrNotAvailable)
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
