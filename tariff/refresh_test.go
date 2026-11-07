package tariff

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshDelay(t *testing.T) {
	day := time.Date(2026, 10, 27, 0, 0, 0, 0, time.Local)
	at := func(h, m int) time.Time {
		return day.Add(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute)
	}

	// rates from midnight until the given number of hours later
	until := func(h int) api.Rates {
		return api.Rates{{Start: day, End: day.Add(time.Duration(h) * time.Hour)}}
	}

	today, tomorrow := until(24), until(48)

	for _, tc := range []struct {
		name     string
		now      time.Time
		interval time.Duration
		early    time.Duration
		rates    api.Rates
		expected time.Duration
	}{
		{"disabled", at(13, 5), time.Hour, 0, today, time.Hour},
		{"morning", at(9, 0), time.Hour, 5 * time.Minute, today, time.Hour},
		{"wait for publication", at(12, 40), time.Hour, 5 * time.Minute, today, 20 * time.Minute},
		{"publication window", at(13, 5), time.Hour, 5 * time.Minute, today, 5 * time.Minute},
		{"late publication", at(15, 0), time.Hour, 5 * time.Minute, today, 30 * time.Minute},
		{"tomorrow available", at(13, 20), time.Hour, 5 * time.Minute, tomorrow, time.Hour},
		{"shorter interval wins", at(13, 5), time.Minute, 5 * time.Minute, today, time.Minute},
		{"no rates", at(13, 5), time.Hour, 5 * time.Minute, nil, 5 * time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, refreshDelay(tc.now, tc.interval, tc.early, tc.rates))
		})
	}
}

func TestEarlyRefreshConfig(t *testing.T) {
	var cc struct {
		embed `mapstructure:",squash"`
	}

	require.NoError(t, util.DecodeOther(map[string]any{"earlyRefresh": "5m"}, &cc))
	assert.Equal(t, 5*time.Minute, cc.EarlyRefresh)
}
