package charger

import (
	"context"
	"math"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/adax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaxRoom(t *testing.T) {
	rooms := []adax.Room{{Id: 1, Name: "Office"}, {Id: 2, Name: "Guest room"}}

	for _, tc := range []struct {
		rooms []adax.Room
		id    string
		want  int64
	}{
		{rooms, "2", 2},
		{rooms, "office", 1},
		{rooms[:1], "", 1},
	} {
		r, err := adaxRoom(tc.rooms, tc.id)
		require.NoError(t, err, tc.id)
		assert.Equal(t, tc.want, r.Id, tc.id)
	}

	_, err := adaxRoom(rooms, "")
	assert.EqualError(t, err, "cannot find room, got: 1 (Office), 2 (Guest room)")

	_, err = adaxRoom(rooms, "attic")
	assert.Error(t, err)

	// exact id wins over a name that looks like an id, duplicate names are rejected
	tricky := []adax.Room{{Id: 1, Name: "2"}, {Id: 2, Name: "Office"}, {Id: 3, Name: "office"}}

	r, err := adaxRoom(tricky, "2")
	require.NoError(t, err)
	assert.Equal(t, int64(2), r.Id)

	_, err = adaxRoom(tricky, "Office")
	assert.ErrorContains(t, err, "ambiguous")
}

func TestAdaxMode(t *testing.T) {
	on, off := true, false
	t20, t22 := 2000, 2200

	assert.Equal(t, Dim, adaxMode(adax.Room{HeatingEnabled: &off, TargetTemperature: &t22}, 22))
	assert.Equal(t, Boost, adaxMode(adax.Room{HeatingEnabled: &on, TargetTemperature: &t22}, 22))
	assert.Equal(t, Normal, adaxMode(adax.Room{HeatingEnabled: &on, TargetTemperature: &t20}, 22))
	assert.Equal(t, Normal, adaxMode(adax.Room{}, 22))
}

func TestAdaxPower(t *testing.T) {
	on, off := true, false
	temp, target := 1500, 1400
	room := adax.Room{HeatingEnabled: &on, Temperature: &temp, TargetTemperature: &target}

	ptr := func(f float64) *float64 { return &f }

	for _, tc := range []struct {
		name      string
		room      adax.Room
		setTarget *float64
		want      float64
	}{
		{"reported target reached", room, nil, 0},
		{"reported heating off", adax.Room{HeatingEnabled: &off, Temperature: &temp, TargetTemperature: &target}, nil, 0},
		{"boost set, report lagging", room, ptr(22), 1000},
		{"set off", room, ptr(0), 0},
		{"set target reached", room, ptr(15), 0},
	} {
		p, err := adaxPower(tc.room, tc.setTarget, 1000)
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.want, p, tc.name)
	}

	_, err := adaxPower(adax.Room{}, nil, 1000)
	assert.ErrorIs(t, err, api.ErrNotAvailable)

	// heating on but no reported target
	_, err = adaxPower(adax.Room{HeatingEnabled: &on, Temperature: &temp}, nil, 1000)
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestAdaxInvalidTemperatures(t *testing.T) {
	for _, tc := range []map[string]any{
		{"normal": math.NaN()},
		{"setpoint": math.NaN()},
		{"normal": 22, "setpoint": 20},
		{"setpoint": 40},
	} {
		tc["user"], tc["password"] = "user", "password"
		_, err := NewAdaxFromConfig(context.Background(), tc)
		assert.ErrorContains(t, err, "invalid temperatures", tc)
	}
}
