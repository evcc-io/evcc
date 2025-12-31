package loadpoint

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeOther_RejectsUnknownFields(t *testing.T) {
	config := map[string]any{
		"title":        "Test",
		"unknownField": "value",
	}

	var dc DynamicConfig
	err := util.DecodeOther(config, &dc)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid keys")
}

func TestSplitConfig_PassesUnknownFieldsToOther(t *testing.T) {
	config := map[string]any{
		"title":            "Garage",
		"planPrecondition": int64(3600),
	}

	dc, other, err := SplitConfig(config)

	require.NoError(t, err)
	assert.Equal(t, "Garage", dc.Title)
	assert.Contains(t, other, "planPrecondition")
}
