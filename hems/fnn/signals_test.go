package fnn

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/hems/smartgrid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalsValidation(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `[]`, `{"w4":null}`, `{"w4":1}`, `{"w4":"true"}`, `{"w5":true}`, `{"w4":false,"s1":true}`, `{"s2":true}`} {
		_, err := decodeSignals(raw)
		require.Error(t, err, raw)
	}
	for _, raw := range []string{`{"w4":false}`, `{"w3":true,"s1":false,"s2":true,"w4":true}`} {
		_, err := decodeSignals(raw)
		require.NoError(t, err)
	}
}

func TestSignalsOneSnapshotPerCycle(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	calls := 0
	getter := func() (string, error) {
		calls++
		if calls%2 == 1 {
			return `{"w3":false,"s1":true,"s2":false,"w4":true}`, nil
		}
		return `{"w3":false,"s1":false,"s2":true,"w4":false}`, nil
	}
	c, err := newSnapshotFnn(&stubSite{}, 4200, 10000, getter, 0)
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, 60, *c.CurtailedPercent())
	assert.Equal(t, 4200.0, *c.MaxConsumptionPower())
	require.NoError(t, c.runCycle())
	assert.Equal(t, 2, calls)
	assert.Equal(t, 30, *c.CurtailedPercent())
	assert.Zero(t, *c.MaxConsumptionPower())
}

func TestSignalsErrorsKeepLimits(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	raw := `{"w3":true,"w4":true}`
	var readErr error
	c, err := newSnapshotFnn(&stubSite{}, 4200, 10000, func() (string, error) { return raw, readErr }, 0)
	require.NoError(t, err)
	for _, bad := range []string{`{"w3":false}`, `{"w3":false,"w4":null}`, `{"w4":false,"s1":false}`, `{"w3":false,"w4":false,"s2":true}`} {
		raw = bad
		require.Error(t, c.runCycle())
		assert.Equal(t, 0, *c.CurtailedPercent())
		assert.Equal(t, 4200.0, *c.MaxConsumptionPower())
	}
	raw = `{"w3":false,"w4":false}`
	readErr = errors.New("offline")
	require.Error(t, c.runCycle())
	assert.Equal(t, 0, *c.CurtailedPercent())
	assert.Equal(t, 4200.0, *c.MaxConsumptionPower())
	readErr = nil
	require.NoError(t, c.runCycle())
	assert.Equal(t, 100, *c.CurtailedPercent())
	assert.Zero(t, *c.MaxConsumptionPower())
}

func TestFailedStartupCreatesNoSessions(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	c, err := NewFnn(&stubSite{}, 4200, 10000, boolG(true), nil, nil, func() (bool, error) { return false, errors.New("W4 unavailable") }, 0)
	require.Error(t, err)
	assert.Nil(t, c)
	var count int64
	require.NoError(t, db.Instance.Model(&smartgrid.GridSession{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestLegacyReadFailureKeepsBothLimits(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	active := true
	var readErr error
	c, err := NewFnn(&stubSite{}, 4200, 10000, func() (bool, error) { return active, nil }, nil, nil, func() (bool, error) { return active, readErr }, 0)
	require.NoError(t, err)
	active = false
	readErr = errors.New("W4 unavailable")
	require.Error(t, c.runCycle())
	assert.Equal(t, 0, *c.CurtailedPercent())
	assert.Equal(t, 4200.0, *c.MaxConsumptionPower())
	var sessions []smartgrid.GridSession
	require.NoError(t, db.Instance.Find(&sessions).Error)
	require.Len(t, sessions, 2)
	for _, session := range sessions {
		assert.True(t, session.Finished.IsZero())
	}
}

func TestSignalsCannotMixContacts(t *testing.T) {
	_, err := NewFromConfig(t.Context(), map[string]any{
		"signals": map[string]any{"source": "const", "value": `{"w4":false}`},
		"w4":      map[string]any{"source": "const", "value": false},
	}, &stubSite{})
	require.ErrorContains(t, err, "cannot be combined")
}
