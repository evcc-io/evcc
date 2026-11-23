package fnn

import (
	"testing"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSite implements site.API for testing — only GetGridPower is exercised.
type stubSite struct {
	site.API
}

func (s *stubSite) GetGridPower() float64 { return 0 }

func boolG(v bool) func() (bool, error) {
	return func() (bool, error) { return v, nil }
}

// TestCurtailmentNotConfigured verifies that without W3 no curtailment
// statement is made, while dimming via W4 remains available.
func TestCurtailmentNotConfigured(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	fnn, err := NewFnn(&stubSite{}, 1e3, 1e3, nil, nil, nil, boolG(true), 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.CurtailedPercent())
	assert.Nil(t, fnn.MaxProductionPower())
	assert.Nil(t, hems.Curtailed(fnn))

	assert.NotNil(t, fnn.MaxConsumptionPower())
	assert.NotNil(t, hems.Dimmed(fnn))
}

// TestDimmingNotConfigured verifies that without W4 no dimming statement is
// made, while curtailment via W3 remains available.
func TestDimmingNotConfigured(t *testing.T) {
	fnn, err := NewFnn(&stubSite{}, 0, 1e3, boolG(false), nil, nil, nil, 0)
	require.NoError(t, err)

	assert.Nil(t, fnn.MaxConsumptionPower())
	assert.Nil(t, hems.Dimmed(fnn))

	assert.NotNil(t, fnn.CurtailedPercent())
	assert.NotNil(t, hems.Curtailed(fnn))
}

// TestProductionNominalMaxMissing verifies that partial curtailment without
// generator power yields no export limit instead of 0 W, while W3 stays at 0 W.
func TestProductionNominalMaxMissing(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	for _, tc := range []struct {
		name       string
		w3, s1, s2 bool
		percent    int
		limit      *float64
	}{
		{"uncurtailed", false, false, false, 100, new(0.0)},
		{"w3", true, false, false, 0, new(0.0)},
		{"s1", false, true, false, 60, nil},
		{"s2", false, false, true, 30, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fnn, err := NewFnn(&stubSite{}, 0, 0, boolG(tc.w3), boolG(tc.s1), boolG(tc.s2), nil, 0)
			require.NoError(t, err)

			assert.Equal(t, tc.percent, *fnn.CurtailedPercent())
			assert.Equal(t, tc.limit, fnn.MaxProductionPower())
		})
	}
}
