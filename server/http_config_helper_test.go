package server

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigReqUnmarshal(t *testing.T) {
	var req configReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"type": "template",
		"deviceTitle": "bar",
		"template": "foo",
		"property": 1}
	`), &req))
	assert.Equal(t, config.Commons{
		Type:  "template",
		Title: "bar",
	}, req.Commons)
	assert.Equal(t, map[string]any{
		"template": "foo",
		"property": 1.0,
	}, req.Other)
}
