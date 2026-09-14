package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresets(t *testing.T) {
	ConfigDefaults.Presets = map[string][]Param{
		"preset": {
			{Name: "preset 1"},
			{Name: "preset 2"},
		},
	}

	tmpl := &Template{
		Params: []Param{
			{Name: "first"},
			{Preset: "preset"},
			{Name: "last"},
		},
	}

	require.NoError(t, tmpl.ResolvePresets())
	require.Equal(t, []Param{
		{Name: "first"},
		{Name: "preset 1"},
		{Name: "preset 2"},
		{Name: "last"},
	}, tmpl.Params)
}

func TestRequiredString(t *testing.T) {
	tmpl := &Template{
		Params: []Param{
			{
				Name:     "param",
				Required: true,
			},
		},
	}

	_, _, err := tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "foo",
	})
	assert.NoError(t, err, "test: required present")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "",
	})
	assert.Error(t, err, "test: required present but empty")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": nil,
	})
	assert.Error(t, err, "test: required present but nil")

	_, _, err = tmpl.RenderResult(Meter, RenderModeDocs, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "docs: required present but nil")
}

func TestRequiredNumber(t *testing.T) {
	tmpl := &Template{
		Params: []Param{
			{
				Name:     "param",
				Type:     TypeInt,
				Required: true,
			},
		},
	}

	_, _, err := tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "1",
	})
	assert.NoError(t, err, "test: required present")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "",
	})
	assert.Error(t, err, "test: required present but empty")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "0",
	})
	assert.Error(t, err, "test: required present but zero value")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": nil,
	})
	assert.Error(t, err, "test: required present but nil")

	_, _, err = tmpl.RenderResult(Meter, RenderModeDocs, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "docs: required present but nil")
}

func TestRequiredDeprecated(t *testing.T) {
	tmpl := &Template{
		Params: []Param{
			{
				Name:       "param",
				Required:   true,
				Deprecated: true,
			},
		},
	}

	_, _, err := tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "foo",
	})
	assert.NoError(t, err, "test: required present")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "",
	})
	assert.NoError(t, err, "test: required present but empty")

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "test: required present but nil")

	_, _, err = tmpl.RenderResult(Meter, RenderModeDocs, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "docs: required present but nil")
}

func TestRequiredPerUsage(t *testing.T) {
	tmpl := &Template{
		Params: []Param{
			{
				Name: "usage",
			},
			{
				Name:     "param",
				Required: true,
				Usages:   []string{"battery"},
			},
		},
	}

	_, _, err := tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": nil,
		"Usage": nil,
	})
	require.NoError(t, err)

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": nil,
		"Usage": "pv",
	})
	require.NoError(t, err)

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": nil,
		"Usage": "battery",
	})
	require.Error(t, err)

	_, _, err = tmpl.RenderResult(Meter, RenderModeUnitTest, map[string]any{
		"Param": "foo",
		"Usage": "battery",
	})
	require.NoError(t, err)
}

func TestValidatePattern(t *testing.T) {
	tmpl := &Template{
		Params: []Param{{Name: "host", Pattern: &Pattern{Regex: `^[^\\/\s]+(:[0-9]{1,5})?$`}}},
	}

	tests := []struct {
		host  string
		valid bool
	}{
		{"192.168.1.100", true},
		{"192.168.1.100:8080", true},
		{"example.com", true},
		{"http://192.168.1.100", false},
		{"192.168.1.100/admin", false},
		{"192.168.1.100 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			_, _, err := tmpl.RenderResult(Meter, RenderModeInstance, map[string]any{"host": tt.host})
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not match required pattern")
			}
		})
	}
}

// bool params are passed to the template as real booleans so templates can use
// `{{ if .param }}` instead of comparing against the string "true"
func TestBoolParamRendersAsBool(t *testing.T) {
	tmpl := &Template{
		Params: []Param{{Name: "flag", Type: TypeBool}},
		Render: "flag: {{ if .flag }}on{{ else }}off{{ end }}\nvalue: {{ .flag }}",
	}

	for _, tc := range []struct {
		value any
		want  string
	}{
		{nil, "flag: off\nvalue: false"},     // unset
		{"true", "flag: on\nvalue: true"},    // string, as sent by the config ui
		{"false", "flag: off\nvalue: false"}, // string
		{true, "flag: on\nvalue: true"},      // yaml bool
		{false, "flag: off\nvalue: false"},   // yaml bool
	} {
		values := map[string]any{}
		if tc.value != nil {
			values["flag"] = tc.value
		}

		b, _, err := tmpl.RenderResult(Charger, RenderModeInstance, values)
		require.NoError(t, err, tc.value)
		assert.Equal(t, tc.want, string(b), "value: %v", tc.value)
	}
}

// a bool param default must be a valid bool, otherwise ui and rendering
// disagree. the example is the default in docs and unit test mode.
func TestValidateBoolParam(t *testing.T) {
	tmpl := func(def, example string) *Template {
		return &Template{
			Params: []Param{{
				Name: "flag", Type: TypeBool, Default: def, Example: example,
				Description: TextLanguage{DE: "Schalter", EN: "Flag"},
			}},
		}
	}

	for _, val := range []string{"", "true", "false"} {
		require.NoError(t, tmpl(val, "").Validate(), val)
		require.NoError(t, tmpl("", val).Validate(), val)
	}

	for _, val := range []string{"yes", "1", "True"} {
		require.Error(t, tmpl(val, "").Validate(), val)
		require.Error(t, tmpl("", val).Validate(), val)
	}
}

// every template must render with each of its bool params set to true and
// false in every usage and for every value of its other choice params -
// comparing a bool param against a string (e.g. `eq .flag "true"`) fails at
// execution time and would otherwise only show up at runtime
func TestAllTemplatesRenderWithBoolParams(t *testing.T) {
	for _, class := range ClassValues() {
		for _, tmpl := range ByClass(class, WithDeprecated()) {
			usages := []string{""}
			if i, p := tmpl.ParamByName(ParamUsage); i >= 0 && len(p.Choice) > 0 {
				usages = p.Choice
			}

			// `and`/`or` short-circuit, so a comparison behind another choice param
			// is only reached for the right value of it. Vary those one at a time,
			// usage stays crossed with them as it selects whole render sections.
			variants := []map[string]string{nil}
			for _, p := range tmpl.Params {
				if p.Name == ParamUsage {
					continue
				}
				for _, c := range p.Choice {
					variants = append(variants, map[string]string{p.Name: c})
				}
			}

			for _, p := range tmpl.Params {
				if p.Type != TypeBool {
					continue
				}

				for _, val := range []bool{true, false} {
					for _, usage := range usages {
						for _, variant := range variants {
							values := tmpl.Defaults(RenderModeUnitTest)
							values["template"] = tmpl.Template
							values[p.Name] = val
							if usage != "" {
								values[ParamUsage] = usage
							}
							for k, v := range variant {
								values[k] = v
							}

							_, _, err := tmpl.RenderResult(class, RenderModeInstance, values)
							assert.NoError(t, err, "%s/%s %s=%v usage=%q %v", class, tmpl.Template, p.Name, val, usage, variant)
						}
					}
				}
			}
		}
	}
}
