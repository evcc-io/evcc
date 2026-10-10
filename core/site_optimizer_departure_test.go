package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/session"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestDepartureProfileHalfLife(t *testing.T) {
	now := time.Date(2026, 10, 6, 22, 10, 0, 0, time.Local) // Tuesday

	res := departureProfile(now, nil, 2*departureSlotsPerDay)

	assert.Len(t, res, 2*departureSlotsPerDay)
	assert.InDelta(t, 0.5, lo.Sum(res[:departureSlotsPerDay]), 0.01, "half the mass within 24h")
	assert.InDelta(t, 0.75, lo.Sum(res), 0.01, "three quarters within 48h")
	assert.Less(t, res[0], res[1], "the running first slot carries 5 of 15 minutes")
	for i := 2; i < len(res); i++ {
		assert.Less(t, res[i], res[i-1], "constant hazard decays")
	}
}

func TestDepartureProfileHistory(t *testing.T) {
	now := time.Date(2026, 10, 6, 22, 0, 0, 0, time.Local) // Tuesday

	// plugged in at 18:00, gone at 07:35 the next morning, every day for three weeks
	var history session.Sessions
	for d := 1; d <= 21; d++ {
		created := now.AddDate(0, 0, -d).Add(-4 * time.Hour)
		history = append(history, session.Session{Created: created, Finished: created.Add(13*time.Hour + 35*time.Minute)})
	}

	res := departureProfile(now, history, departureSlotsPerDay)

	// 07:30 is 38 slots after 22:00
	assert.Greater(t, res[38], float32(0.8), "the departure sits in the 07:30 slot")
	assert.Less(t, lo.Sum(res[:38]), float32(0.1), "little before it")
	assert.LessOrEqual(t, lo.Sum(res), float32(1))
}
