package server

import (
	"encoding/json"
	"testing"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigReqUnmarshal(t *testing.T) {
	var req configReq
	require.NoError(t, json.Unmarshal([]byte(`{
		"type": "template",
		"deviceTitle": "bar",
		"template": "foo",
		"deviceProduct": "baz",
		"property": 1}
	`), &req))
	assert.Equal(t, config.Properties{
		Type:    "template",
		Title:   "bar",
		Product: "baz",
	}, req.Properties)
	assert.Equal(t, map[string]any{
		"template": "foo",
		"property": 1.0,
	}, req.Other)
}

func TestConfigReqMarshalToMap(t *testing.T) {
	props := config.Properties{
		Type:    "type",
		Title:   "title",
		Product: "product",
	}

	res, err := propsToMap(props)
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"deviceTitle":   "title",
		"deviceProduct": "product",
	}, res)
}

type testStruct struct {
	Field1 string
	Field2 int
}

type testStructWithBool struct {
	Field1 string
	Field2 int
	Field3 bool
}

func TestMergeMaskedAny(t *testing.T) {
	tests := []struct {
		old           any
		new, expected *testStruct
	}{
		{
			old:      &testStruct{"oldValue1", 24},
			new:      &testStruct{"newValue1", 42},
			expected: &testStruct{"newValue1", 42},
		},
		{
			old:      &testStruct{"oldValue1", 24},
			new:      &testStruct{masked, 42},
			expected: &testStruct{"oldValue1", 42},
		},
	}

	for _, tc := range tests {
		require.NoError(t, mergeMaskedAny(tc.old, tc.new))
		assert.Equal(t, tc.expected, tc.new)
	}

	// Test boolean field handling
	boolTests := []struct {
		old           any
		new, expected *testStructWithBool
	}{
		{
			// Boolean false should not be overwritten by true
			old:      &testStructWithBool{"oldValue", 24, true},
			new:      &testStructWithBool{"newValue", 42, false},
			expected: &testStructWithBool{"newValue", 42, false},
		},
		{
			// Boolean true should be preserved
			old:      &testStructWithBool{"oldValue", 24, false},
			new:      &testStructWithBool{"newValue", 42, true},
			expected: &testStructWithBool{"newValue", 42, true},
		},
		{
			// Masked string should be restored, boolean should not be merged
			old:      &testStructWithBool{"oldValue", 24, true},
			new:      &testStructWithBool{masked, 42, false},
			expected: &testStructWithBool{"oldValue", 42, false},
		},
	}

	for _, tc := range boolTests {
		require.NoError(t, mergeMaskedAny(tc.old, tc.new))
		assert.Equal(t, tc.expected, tc.new)
	}
}

func TestSquashedMergeMaskedAny(t *testing.T) {
	old := globalconfig.Mqtt{
		Config: mqtt.Config{
			Broker: "host",
			User:   "user",
		},
		Topic: "test",
	}
	{
		new := old
		new.User = masked

		require.NoError(t, mergeMaskedAny(old, &new))
		assert.Equal(t, "user", new.User)
	}
	{
		new := old
		new.User = "new"

		require.NoError(t, mergeMaskedAny(old, &new))
		assert.Equal(t, "new", new.User)
	}
}

func TestMergeMaskedFiltersBehavior(t *testing.T) {
	conf := map[string]any{
		"template": "demo-meter",
		"power":    200.0,
	}

	old := map[string]any{
		"template":      "demo-meter",
		"power":         100.0,
		"outdatedField": "old-value",
	}

	result, err := mergeMasked(templates.Meter, conf, old)
	require.NoError(t, err)

	assert.Equal(t, 200.0, result["power"])
	assert.Equal(t, "demo-meter", result["template"])
	assert.NotContains(t, result, "outdatedField")
}

func TestFilterValidTemplateParams(t *testing.T) {
	conf := map[string]any{
		"template":      "generic",
		"usage":         "grid",
		"capacity":      50.0,
		"power":         100.0,
		"outdatedField": "should-be-removed",
	}

	result := filterValidTemplateParams(&templates.Template{
		TemplateDefinition: templates.TemplateDefinition{
			Params: []templates.Param{
				{Name: "usage"},
				{Name: "power"},
				{Name: "capacity"},
			},
		},
	}, conf)

	assert.Equal(t, "generic", result["template"], "template")
	assert.Equal(t, "grid", result["usage"], "usage")
	assert.NotContains(t, result, "outdatedField")
}
