package core

import (
	"testing"

	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadpointDecodeWithDeprecatedPlanPrecondition(t *testing.T) {
	// Old config format with planPrecondition (removed in PR #24423)
	// Without the PlanPrecondition_ compatibility field, this would fail with:
	// "invalid keys: planPrecondition"
	config := map[string]any{
		"planPrecondition": int64(3600),
	}

	lp := &Loadpoint{}
	err := util.DecodeOther(config, lp)

	// Should NOT error - field should be accepted into PlanPrecondition_
	assert.NoError(t, err, "DecodeOther should accept deprecated planPrecondition field")
}

func TestLoadpointFromConfigWithUnknownField(t *testing.T) {
	// Simulate old config with unknown field (like planPrecondition before fix)
	// This proves: unknown field -> DecodeOther fails -> lp is nil
	config := map[string]any{
		"unknownField": "test", // This will cause DecodeOther to fail
	}

	lp, err := NewLoadpointFromConfig(util.NewLogger("test"), settings.NewDatabaseSettingsAdapter("test"), config)

	// Unknown fields cause nil loadpoint - this is the root cause of the panic
	require.Error(t, err)
	assert.Nil(t, lp, "Loadpoint should be nil when DecodeOther fails")
	assert.Contains(t, err.Error(), "invalid keys")
}
