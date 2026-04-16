package metrics

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMeterEnergyMeterTotal(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetImportMeterTotal(10)
	assert.Equal(t, 0.0, me.Imported())
	me.SetImportMeterTotal(11)
	assert.Equal(t, 1.0, me.Imported())
	me.SetImportMeterTotal(11)
	assert.Equal(t, 1.0, me.Imported())
}

func TestMeterEnergyAddPower(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 0.0, me.Imported())

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.Imported())

	clock.Add(30 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.5, me.Imported())
}

func TestAccumulatorJSONRoundTrip(t *testing.T) {
	ts := now.BeginningOfDay().Add(2 * time.Hour)

	me := NewAccumulator()
	me.Restore(1.5, 0.25, ts)

	data, err := json.Marshal(me)
	require.NoError(t, err)

	var restored Accumulator
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, 1.5, restored.Imported())
	assert.Equal(t, 0.25, restored.Exported())
	assert.True(t, ts.Equal(restored.Updated()))
}

func TestAccumulatorJSONLegacyCompatibility(t *testing.T) {
	ts := now.BeginningOfDay().Add(3 * time.Hour)
	data := []byte(`{"accumulated":2.75,"updated":"` + ts.Format(time.RFC3339Nano) + `"}`)

	var restored Accumulator
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, 2.75, restored.Imported())
	assert.Equal(t, 0.0, restored.Exported())
	assert.True(t, ts.Equal(restored.Updated()))
}
