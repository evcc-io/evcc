package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriceProfile(t *testing.T) {
	tue := time.Date(2026, 9, 15, 0, 0, 0, 0, romeLocation)
	require.Equal(t, time.Tuesday, tue.Weekday())
	sat := time.Date(2026, 9, 19, 0, 0, 0, 0, romeLocation)
	require.Equal(t, time.Saturday, sat.Weekday())

	p := newPriceProfile(api.Rates{
		{Start: tue, End: tue.Add(time.Hour), Value: 1},
		{Start: tue.Add(time.Hour), End: tue.Add(2 * time.Hour), Value: 2},
		{Start: sat, End: sat.Add(time.Hour), Value: 10},
		{Start: sat.Add(2 * time.Hour), End: sat.Add(3 * time.Hour), Value: 20},
	})

	weekday := tue.AddDate(0, 0, 7)
	for hour, want := range []float64{1, 2} {
		v, ok := p.price(weekday.Add(time.Duration(hour) * time.Hour))
		require.True(t, ok)
		assert.Equal(t, want, v)
	}

	v, ok := p.price(sat.AddDate(0, 0, 7))
	require.True(t, ok)
	assert.Equal(t, 10.0, v)

	// fall back to the other day class at the same hour
	v, ok = p.price(weekday.Add(2 * time.Hour))
	require.True(t, ok)
	assert.Equal(t, 20.0, v)

	v, ok = p.price(sat.AddDate(0, 0, 7).Add(time.Hour))
	require.True(t, ok)
	assert.Equal(t, 2.0, v)

	// both day classes unsampled at this hour
	_, ok = p.price(weekday.Add(3 * time.Hour))
	assert.False(t, ok)
}

func TestExtendForecast(t *testing.T) {
	loc := romeLocation
	today := time.Date(2026, 9, 15, 0, 0, 0, 0, loc)

	var rates api.Rates
	for _, day := range []time.Time{today.AddDate(0, 0, -1), today} {
		for hour := range 24 {
			rates = append(rates, api.Rate{
				Start: day.Add(time.Duration(hour) * time.Hour),
				End:   day.Add(time.Duration(hour+1) * time.Hour),
				Value: float64(hour),
			})
		}
	}

	tf := &Pun{forecast: 2}
	out := tf.extendForecast(rates, today)

	require.Len(t, out, 24+2*24)
	for _, r := range out {
		assert.False(t, r.Start.Before(today), "history must not be published")
	}

	// first forecast rate starts where the published data ends
	assert.Equal(t, today.AddDate(0, 0, 1), out[24].Start)
	assert.Equal(t, today.AddDate(0, 0, 1+2), out[len(out)-1].End)

	for i := 24; i < len(out); i++ {
		assert.Equal(t, float64((i-24)%24), out[i].Value)
	}

	for i := 1; i < len(out); i++ {
		assert.Equal(t, out[i-1].End, out[i].Start)
	}
}

func TestExtendForecastShortHistory(t *testing.T) {
	loc := romeLocation
	today := time.Date(2026, 9, 17, 0, 0, 0, 0, loc)
	require.Equal(t, time.Thursday, today.Weekday())

	// one day of history, weekday only: no weekend samples exist
	var rates api.Rates
	for _, day := range []time.Time{today.AddDate(0, 0, -1), today} {
		for hour := range 24 {
			rates = append(rates, api.Rate{
				Start: day.Add(time.Duration(hour) * time.Hour),
				End:   day.Add(time.Duration(hour+1) * time.Hour),
				Value: float64(hour),
			})
		}
	}

	tf := &Pun{forecast: 3}
	out := tf.extendForecast(rates, today)

	// weekend hours fall back to the weekday average instead of being skipped
	require.Len(t, out, 24+3*24)
	assert.Equal(t, today.AddDate(0, 0, 4), out[len(out)-1].End)

	for i := 24; i < len(out); i++ {
		assert.Equal(t, float64(i%24), out[i].Value)
	}

	for i := 1; i < len(out); i++ {
		assert.Equal(t, out[i-1].End, out[i].Start)
	}
}

func TestExtendForecastWeekend(t *testing.T) {
	loc := romeLocation
	fri := time.Date(2026, 9, 18, 0, 0, 0, 0, loc)
	require.Equal(t, time.Friday, fri.Weekday())

	var rates api.Rates
	for _, day := range []time.Time{fri, fri.AddDate(0, 0, 1)} {
		for hour := range 24 {
			val := float64(hour)
			if day.Equal(fri.AddDate(0, 0, 1)) {
				val = 100 + float64(hour)
			}
			rates = append(rates, api.Rate{
				Start: day.Add(time.Duration(hour) * time.Hour),
				End:   day.Add(time.Duration(hour+1) * time.Hour),
				Value: val,
			})
		}
	}

	tf := &Pun{forecast: 2}
	out := tf.extendForecast(rates, fri)

	require.Len(t, out, 2*24+2*24)

	// weekend forecast uses the weekend profile, weekday forecast the weekday profile
	for i := 48; i < len(out); i++ {
		want := 100 + float64((i-48)%24)
		if i >= 48+24 {
			want = float64((i - 48 - 24) % 24)
		}
		assert.Equal(t, want, out[i].Value)
	}
}
