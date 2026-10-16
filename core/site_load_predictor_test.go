package core

import (
	"slices"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestApplyTemperatureCorrection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	now := time.Now().Truncate(15 * time.Minute)

	mockTariff := api.NewMockTariff(ctrl)

	rates := []api.Rate{}

	// Past 7 days - create slots for ALL hours to ensure historical data exists
	for day := -7; day < 0; day++ {
		for hour := range 24 {
			for slot := range 4 {
				baseTime := now.AddDate(0, 0, day).Truncate(24 * time.Hour)
				rates = append(rates, api.Rate{
					Start: baseTime.Add(time.Duration(hour)*time.Hour + time.Duration(slot)*15*time.Minute),
					End:   baseTime.Add(time.Duration(hour)*time.Hour + time.Duration(slot+1)*15*time.Minute),
					Value: 10.0,
				})
			}
		}
	}

	// Future forecast: 12 slots (3 hours)
	// First hour: 5°C, second hour: 15°C, third hour: 20°C (above heating stop threshold)
	for i := range 12 {
		temp := 5.0
		switch {
		case i >= 8:
			temp = 20.0
		case i >= 4:
			temp = 15.0
		}
		rates = append(rates, api.Rate{
			Start: now.Add(time.Duration(i) * 15 * time.Minute),
			End:   now.Add(time.Duration(i+1) * 15 * time.Minute),
			Value: temp,
		})
	}

	mockTariff.EXPECT().Rates().Return(rates, nil).AnyTimes()

	site := &Site{
		log:     util.NewLogger("test"),
		tariffs: &tariff.Tariffs{Temperature: mockTariff},
	}

	profile := slices.Repeat([]float64{2.0}, 12)

	result := site.applyTemperatureCorrection(profile)

	require.Len(t, result, 12)

	// Verify correction is applied: first hour should increase, second hour should decrease
	assert.Greater(t, result[0], 2.0, "first hour should increase (colder forecast)")
	assert.Less(t, result[4], 2.0, "second hour should decrease (warmer forecast)")
	assert.Greater(t, result[0], result[4], "first hour should be higher than second hour")
	// above the heating threshold the correction is skipped but the historical average is kept
	assert.Equal(t, 2.0, result[8], "third hour keeps historical average when above heating threshold")
}

func TestTileAndTrim(t *testing.T) {
	profile := make([]float64, 96)
	for i := range profile {
		profile[i] = float64(i)
	}

	res := tileAndTrim(profile, 200)
	require.Len(t, res, 200)

	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)
	assert.Equal(t, float64(firstSlot), res[0], "starts at the current slot")
	assert.Equal(t, res[0], res[96], "wraps after a full day")
}

func TestExcludeDays(t *testing.T) {
	bod := now.BeginningOfDay()
	key := func(i int) string { return bod.AddDate(0, 0, -i).Format(time.DateOnly) }

	t.Run("no absent days", func(t *testing.T) {
		classes := make(map[string]metrics.DayClass)
		for i := 1; i <= homeProfileLookback; i++ {
			classes[key(i)] = metrics.DayNormal
		}

		ex := excludeDays(classes, bod)
		assert.Empty(t, ex.exclude)
		assert.Equal(t, bod.AddDate(0, 0, -homeProfileDays), ex.from)
	})

	t.Run("vacation is replaced by older days", func(t *testing.T) {
		classes := make(map[string]metrics.DayClass)
		for i := 1; i <= homeProfileLookback; i++ {
			classes[key(i)] = metrics.DayNormal
		}

		// 14 days vacation ending 3 days ago, one outlier
		for i := 4; i < 18; i++ {
			classes[key(i)] = metrics.DayAbsent
		}
		classes[key(20)] = metrics.DayOutlier

		ex := excludeDays(classes, bod)
		assert.Len(t, ex.exclude, 15)
		assert.Equal(t, bod.AddDate(0, 0, -(homeProfileDays+15)), ex.from, "window extends by the excluded days")
	})

	t.Run("incomplete days are kept, days without data are skipped", func(t *testing.T) {
		classes := map[string]metrics.DayClass{key(1): metrics.DayAbsent}
		for i := 2; i <= 40; i++ {
			if i%2 == 0 {
				classes[key(i)] = metrics.DayIncomplete
			}
		}

		ex := excludeDays(classes, bod)
		assert.Equal(t, []string{key(1)}, ex.exclude)
		assert.Equal(t, bod.AddDate(0, 0, -homeProfileLookback), ex.from, "fewer than 28 days with data: full lookback")
	})

	t.Run("only days with data count towards the minimum", func(t *testing.T) {
		classes := map[string]metrics.DayClass{key(1): metrics.DayAbsent}
		for i := 2; i < 2+homeProfileMinDays-1; i++ {
			classes[key(i)] = metrics.DayNormal
		}

		ex := excludeDays(classes, bod)
		assert.Empty(t, ex.exclude)
	})

	t.Run("too few remaining days", func(t *testing.T) {
		classes := make(map[string]metrics.DayClass)
		for i := 1; i <= homeProfileLookback; i++ {
			classes[key(i)] = metrics.DayAbsent
		}
		for i := 1; i < homeProfileMinDays; i++ {
			classes[key(i)] = metrics.DayNormal
		}

		ex := excludeDays(classes, bod)
		assert.Empty(t, ex.exclude, "nothing excluded below minimum days")
	})
}

func TestHomeProfileExcludesVacation(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, metrics.SetupSchema())

	col, err := metrics.NewCollector(metrics.Home, metrics.Home, metrics.Home)
	require.NoError(t, err)

	var meter int
	require.NoError(t, db.Instance.Raw(`SELECT id FROM entities WHERE "group" = ? AND name = ?`, metrics.Home, metrics.Home).Scan(&meter).Error)

	bod := now.BeginningOfDay()

	// 6 weeks: 2 weeks vacation ending yesterday, normal days before with an evening peak
	for day := -42; day < 0; day++ {
		for slot := range 96 {
			energy := 0.1 // kWh per slot
			switch {
			case day >= -14:
				energy = 0.09
			case slot/4 == 18:
				energy = 0.5
			}

			ts := bod.AddDate(0, 0, day).Add(time.Duration(slot) * tariff.SlotDuration)
			require.NoError(t, db.Instance.Exec(`INSERT INTO meters (meter, ts, energy, return_energy) VALUES (?, ?, ?, 0)`, meter, ts.Unix(), energy).Error)
		}
	}

	site := &Site{
		log:        util.NewLogger("foo"),
		collectors: map[string]*metrics.Collector{metrics.Home: col},
	}

	res, err := site.homeProfile(96)
	require.NoError(t, err)

	// profile only contains the 28 normal days before the vacation
	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(bod) / tariff.SlotDuration)
	for i, v := range res {
		expected := 100.0 // Wh
		if (firstSlot+i)%96/4 == 18 {
			expected = 500
		}
		require.InDelta(t, expected, v, 1e-6, "slot %d", i)
	}

	// right after midnight the previous day may still be incomplete, so nothing is cached
	site.homeExclusion = nil
	ex, err := site.homeDayExclusion(col, bod.Add(tariff.SlotDuration))
	require.NoError(t, err)
	assert.Len(t, ex.exclude, 14)
	assert.Nil(t, site.homeExclusion)

	_, err = site.homeDayExclusion(col, bod.Add(time.Hour))
	require.NoError(t, err)
	require.NotNil(t, site.homeExclusion)
	assert.Len(t, site.homeExclusion.exclude, 14)
}
