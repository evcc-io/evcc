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
