package core

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeterEnergy(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &meterEnergy{clock: clock}

	me.AddMeterTotal(10)
	assert.Equal(t, 0.0, me.AccumulatedEnergy())
	me.AddMeterTotal(11)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())
	me.AddMeterTotal(11)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())

	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())

	clock.Add(30 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.5, me.AccumulatedEnergy())
}

func TestMeterEnergyJSON(t *testing.T) {
	type storage map[string]*meterEnergy
	pvEnergy := storage{
		"pv1": {Accumulated: 10.0},
	}

	j, err := json.Marshal(pvEnergy)
	require.NoError(t, err)

	var res storage
	require.NoError(t, json.Unmarshal(j, &res))

	require.Equal(t, pvEnergy["pv1"].Accumulated, res["pv1"].Accumulated)
}
