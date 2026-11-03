package server

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestFullChargeDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := vehicle.NewMockAPI(ctrl)

	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	v.EXPECT().SetFullChargeTime(past)
	require.NoError(t, fullChargeDone(v, past))

	assert.Error(t, fullChargeDone(v, time.Now().Add(time.Hour)), "future")
}

func TestFullChargeDoneSetter(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := vehicle.NewMockAPI(ctrl)
	set := fullChargeDoneSetter(v)

	ts := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	v.EXPECT().SetFullChargeTime(ts)
	require.NoError(t, set("2026-11-01T12:00:00Z"))

	v.EXPECT().SetFullChargeTime(gomock.Any()).Times(2)
	require.NoError(t, set("now"))
	require.NoError(t, set(""))

	assert.Error(t, set("yesterday"))
}

func TestUpdateFullCharge(t *testing.T) {
	ctrl := gomock.NewController(t)
	v := vehicle.NewMockAPI(ctrl)

	v.EXPECT().GetFullCharge().Return(api.FullCharge{Days: 14, Energy: 190}).AnyTimes()

	v.EXPECT().SetFullCharge(api.FullCharge{Days: 14, Energy: 0})
	require.NoError(t, updateFullCharge(v, []byte(`{"energy":0}`)))

	v.EXPECT().SetFullCharge(api.FullCharge{Days: 28, Energy: 190})
	require.NoError(t, updateFullCharge(v, []byte(`{"days":28}`)))

	assert.Error(t, updateFullCharge(v, []byte(`{`)))
}
