package core

import (
	"testing"

	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// simulatedOldLoadpoint without PlanPrecondition_ field
type simulatedOldLoadpoint struct {
	GuardDuration_ float64 `mapstructure:"guardduration"`
}

func TestDecodeOther_FailsOnUnknownField(t *testing.T) {
	config := map[string]any{"planPrecondition": int64(3600)}

	err := util.DecodeOther(config, &simulatedOldLoadpoint{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid keys: planPrecondition")
}

func TestNewLoadpointFromConfig_ToleratesUnknownFields(t *testing.T) {
	config := map[string]any{"unknownField": "test"}

	lp, err := NewLoadpointFromConfig(util.NewLogger("test"), settings.NewDatabaseSettingsAdapter("test"), config)

	// unknown fields are logged but tolerated
	require.NoError(t, err)
	assert.NotNil(t, lp)
}
