package charger

import (
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
}
