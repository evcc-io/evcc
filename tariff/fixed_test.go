package tariff

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/fixed"
	"github.com/golang-module/carbon"
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
		dayStart := carbon.Time2Carbon(tf.clock.Now()).StartOfDay().AddDays(i)

		expect = append(expect, api.Rate{
			Price: 0.3,
			Start: dayStart.Carbon2Time(),
			End:   dayStart.AddDay().Carbon2Time(),
		})
	}

	rates, err := tf.Rates()
	assert.NoError(t, err)
	assert.Equal(t, expect, rates)
}
