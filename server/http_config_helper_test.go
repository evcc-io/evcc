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
		"productbrand": "baz",
		"property": 1}
	`), &req))
	assert.Equal(t, config.Properties{
		Type:  "template",
		Title: "bar",
		Brand: "baz",
	}, req.Properties)
	assert.Equal(t, map[string]any{
		"template": "foo",
		"property": 1.0,
	}, req.Other)
}

func TestConfigReqMarshalToMap(t *testing.T) {
	props := config.Properties{
		Type:  "type",
		Title: "title",
		Brand: "brand",
	}

	res, err := propsToMap(props)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"deviceTitle":  "title",
		"productBrand": "brand",
	}, res)
}
