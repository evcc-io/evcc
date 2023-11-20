package tariff

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/fixed"
	"github.com/jinzhu/now"
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
	for dow := 0; dow < 7; dow++ {
		dayStart := now.With(tf.clock.Now()).BeginningOfDay().AddDate(0, 0, dow)

		for hour := 0; hour < 24; hour++ {
			expect = append(expect, api.Rate{
				Price: 0.3,
				Start: dayStart.Add(time.Hour * time.Duration(hour)),
				End:   dayStart.Add(time.Hour * time.Duration(hour+1)),
			})
		}
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
		dayStart := now.With(tf.clock.Now()).BeginningOfDay().AddDate(0, 0, i)

		// 00:00-05:00 0.1
		for hour := 0; hour < 5; hour++ {
			expect = append(expect, api.Rate{
				Price: 0.1,
				Start: dayStart.Add(time.Hour * time.Duration(hour)),
				End:   dayStart.Add(time.Hour * time.Duration(hour+1)),
			})
		}

		// 05:00-05:30 0.1
		expect = append(expect, api.Rate{
			Price: 0.1,
			Start: dayStart.Add(5 * time.Hour),
			End:   dayStart.Add(5*time.Hour + 30*time.Minute),
		})

		// 05:30-06:00 0.5
		expect = append(expect, api.Rate{
			Price: 0.5,
			Start: dayStart.Add(5*time.Hour + 30*time.Minute),
			End:   dayStart.Add(6 * time.Hour),
		})

		// 06:00-21:00 0.5
		for hour := 6; hour < 21; hour++ {
			expect = append(expect, api.Rate{
				Price: 0.5,
				Start: dayStart.Add(time.Hour * time.Duration(hour)),
				End:   dayStart.Add(time.Hour * time.Duration(hour+1)),
			})
		}

		// 21:00-00:00 0.1
		for hour := 21; hour < 24; hour++ {
			expect = append(expect, api.Rate{
				Price: 0.1,
				Start: dayStart.Add(time.Hour * time.Duration(hour)),
				End:   dayStart.Add(time.Hour * time.Duration(hour+1)),
			})
		}
	}

	rates, err := tf.Rates()
	assert.NoError(t, err)
	assert.Equal(t, expect, rates)
}
