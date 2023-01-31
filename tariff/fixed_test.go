package tariff

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/fixed"
	"github.com/golang-module/carbon/v2"
	"github.com/stretchr/testify/assert"
)

func TestFixed(t *testing.T) {
	tf := &Fixed{
		clock: clock.NewMock(),
		zones: []fixed.Zone{
			{Price: 0.3},
		},
	}

	var expect api.Rates
	for i := 0; i < 7; i++ {
		dayStart := carbon.FromStdTime(tf.clock.Now()).StartOfDay().AddDays(i)

		expect = append(expect, api.Rate{
			Price: 0.3,
			Start: dayStart.ToStdTime(),
			End:   dayStart.AddDay().ToStdTime(),
		})
	}

	rates, err := tf.Rates()
	assert.NoError(t, err)
	assert.Equal(t, expect, rates)
}

func TestFixedSplitZones(t *testing.T) {
	at, err := NewFixedFromConfig(map[string]interface{}{
		"price": 0.5,
		"zones": []struct {
			Price float64
			Hours string
		}{
			{0.1, "0-5:30,21-0"},
		},
	})
	assert.NoError(t, err)

	tf := at.(*Fixed)
	tf.clock = clock.NewMock()

	var expect api.Rates
	for i := 0; i < 7; i++ {
		dayStart := carbon.FromStdTime(tf.clock.Now()).StartOfDay().AddDays(i)

		expect = append(expect,
			api.Rate{
				Price: 0.1,
				Start: dayStart.ToStdTime(),
				End:   dayStart.AddHours(5).AddMinutes(30).ToStdTime(),
			},
			api.Rate{
				Price: 0.5,
				Start: dayStart.AddHours(5).AddMinutes(30).ToStdTime(),
				End:   dayStart.AddHours(21).ToStdTime(),
			},
			api.Rate{
				Price: 0.1,
				Start: dayStart.AddHours(21).ToStdTime(),
				End:   dayStart.AddDay().ToStdTime(),
			},
		)
	}

	rates, err := tf.Rates()
	assert.NoError(t, err)
	assert.Equal(t, expect, rates)
}
