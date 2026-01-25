package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/spf13/cast"
)

// Template describes is a proxy device for use with cli and automated testing
type Template struct {
	Template     string
	Deprecated   bool           `json:"-"`
	Auth         map[string]any `json:",omitempty"` // OAuth parameters (if required)
	Group        string         `json:",omitempty"` // the group this template belongs to, references groupList entries
	Covers       []string       `json:",omitempty"` // list of covered outdated template names
	Products     []Product      `json:",omitempty"` // list of products this template is compatible with
	Capabilities []string       `json:",omitempty"`
	Countries    []CountryCode  `json:",omitempty"` // list of countries supported by this template
	Requirements Requirements   `json:",omitempty"`
	Params       []Param        `json:",omitempty"`
	Render       string         `json:"-"` // rendering template
}

// UpdateParamWithDefaults adds default values to specific param name entries
func (t *Template) UpdateParamsWithDefaults() error {
	for i, p := range t.Params {
		if index, resultMapItem := ConfigDefaults.ParamByName(strings.ToLower(p.Name)); index > -1 {
			t.Params[i].OverwriteProperties(resultMapItem)
		}
	}

	return nil
}

// UpdateModbusParamsWithDefaults populates modbus param fields with global defaults
// when device-specific values are not set (zero/empty).
func (t *Template) UpdateModbusParamsWithDefaults() error {
	idx, modbusParam := t.ParamByName(ParamModbus)
	if idx == -1 || len(modbusParam.Choice) == 0 {
		return nil
	}

	if modbusParam.ID == 0 {
		modbusParam.ID = cast.ToInt(ConfigDefaults.ModbusDefault(ModbusParamId))
	}
	if modbusParam.Baudrate == 0 {
		modbusParam.Baudrate = cast.ToInt(ConfigDefaults.ModbusDefault(ModbusParamBaudrate))
	}
	if modbusParam.Comset == "" {
		modbusParam.Comset = cast.ToString(ConfigDefaults.ModbusDefault(ModbusParamComset))
	}
	if modbusParam.Port == 0 {
		modbusParam.Port = cast.ToInt(ConfigDefaults.ModbusDefault(ModbusParamPort))
	}

	t.Params[idx] = modbusParam
	return nil
}

func (t *Template) SortRequiredParamsFirst() error {
	slices.SortStableFunc(t.Params, func(a, b Param) int {
		if a.Required && !b.Required {
			return -1
		}
		if b.Required && !a.Required {
			return +1
		}
		return 0
	})

	return nil
}

// validate the template (only rudimentary for now)
func (t *Template) Validate() error {
	for _, c := range t.Capabilities {
		if !slices.Contains(ValidCapabilities, c) {
			return fmt.Errorf("invalid capability: '%s'", c)
		}
	}

	for _, c := range t.Countries {
		if !c.IsValid() {
			return fmt.Errorf("invalid country code: '%s'", c)
		}
	}

	for _, r := range t.Requirements.EVCC {
		if !slices.Contains(ValidRequirements, r) {
			return fmt.Errorf("invalid requirement: '%s'", r)
		}
	}

	for _, p := range t.Params {
		if p.IsDeprecated() {
			continue
		}

		// Validate that a param cannot be both masked and private
		if p.Mask && p.Private {
			return fmt.Errorf("param %s: 'mask' and 'private' cannot be used together. Use 'mask' for sensitive data like passwords/tokens that should be hidden in UI. Use 'private' for personal data like emails/locations that should only be redacted from bug reports", p.Name)
		}

		if p.Description.String("en") == "" || p.Description.String("de") == "" {
			return fmt.Errorf("param %s: description can't be empty", p.Name)
		}

		maxLength := 50
		actualLength := max(len(p.Description.String("en")), len(p.Description.String("de")))
		if actualLength > maxLength {
			return fmt.Errorf("param %s: description too long (%d/%d allowed)- use help instead", p.Name, actualLength, maxLength)
		}

		switch p.Name {
		case ParamUsage:
			for _, c := range p.Choice {
				if !slices.Contains(UsageStrings(), c) {
					return fmt.Errorf("invalid usage: '%s'", c)
				}
			}
		case ParamModbus:
			for _, c := range p.Choice {
				if !slices.Contains(ValidModbusChoices, c) {
					return fmt.Errorf("invalid modbus type: '%s'", c)
				}
			}
		}

		// validate pattern examples against pattern
		if p.Pattern != nil && p.Pattern.Regex != "" && len(p.Pattern.Examples) > 0 {
			for _, example := range p.Pattern.Examples {
				if err := p.Pattern.Validate(example); err != nil {
					return fmt.Errorf("param %s: pattern example %q is invalid: pattern=%q", p.Name, example, p.Pattern.Regex)
				}
			}
		}
	}

	return nil
}

// add the referenced base Params and overwrite existing ones
func (t *Template) ResolvePresets() error {
	currentParams := make([]Param, len(t.Params))
	copy(currentParams, t.Params)
	t.Params = []Param{}

	for _, p := range currentParams {
		if p.Preset != "" {
			preset, ok := ConfigDefaults.Presets[p.Preset]
			if !ok {
				return fmt.Errorf("could not find preset definition: %s", p.Preset)
			}

			for _, pp := range preset {
				if i, _ := t.ParamByName(pp.Name); i > -1 {
					return fmt.Errorf("parameter %s must not be defined before containing preset %s", pp.Name, p.Preset)
				}
			}

			t.Params = append(t.Params, preset...)
			continue
		}

		if i, _ := t.ParamByName(p.Name); i > -1 {
			// take the preset values as a base and overwrite it with param values
			p.OverwriteProperties(t.Params[i])
			t.Params[i] = p
		} else {
			t.Params = append(t.Params, p)
		}
	}

	return nil
}

// check if the provided group exists
func (t *Template) ResolveGroup() error {
	if t.Group == "" {
		return nil
	}

	_, ok := ConfigDefaults.DeviceGroups[t.Group]
	if !ok {
		return fmt.Errorf("could not find devicegroup definition: %s", t.Group)
	}

	return nil
}

// return the language specific group title
func (t *Template) GroupTitle(lang string) string {
	tl := ConfigDefaults.DeviceGroups[t.Group]
	return tl.String(lang)
}

// Defaults returns a map of default values for the template
func (t *Template) Defaults(renderMode int) map[string]any {
	values := make(map[string]any, len(t.Params))
	for _, p := range t.Params {
		values[p.Name] = p.DefaultValue(renderMode)
	}

	return values
}

// SetParamDefault updates the default value of a param
func (t *Template) SetParamDefault(name string, value string) {
	for i, p := range t.Params {
		if p.Name == name {
			t.Params[i].Default = value
			return
		}
	}
}

// return the param with the given name
func (t *Template) ParamByName(name string) (int, Param) {
	for i, p := range t.Params {
		if strings.EqualFold(p.Name, name) {
			return i, p
		}
	}
	return -1, Param{}
}

// Usages returns the list of supported usages
func (t *Template) Usages() []string {
	if i, p := t.ParamByName(ParamUsage); i > -1 {
		return p.Choice
	}

	return nil
}

// return all modbus choices defined in the template
func (t *Template) ModbusChoices() []string {
	if i, p := t.ParamByName(ParamModbus); i > -1 {
		return p.Choice
	}

	return nil
}

//go:embed proxy.tpl
var proxyTmpl string

// RenderProxyWithValues renders the proxy template
func (t *Template) RenderProxyWithValues(values map[string]any, lang string) ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(sprig.FuncMap()).Parse(proxyTmpl)
	if err != nil {
		panic(err)
	}

	t.ModbusParams("", values)

	for index, p := range t.Params {
		v, ok := values[p.Name]
		if !ok {
			continue
		}

		switch p.Type {
		case TypeList:
			for _, e := range v.([]string) {
				t.Params[index].Values = append(p.Values, p.yamlQuote(e))
			}
		default:
			switch v := v.(type) {
			case string:
				t.Params[index].Value = p.yamlQuote(v)
			case int:
				t.Params[index].Value = strconv.Itoa(v)
			}
		}
	}

	// remove params with no values
	var newParams []Param
	for _, param := range t.Params {
		if !param.IsRequired() {
			switch param.Type {
			case TypeList:
				if len(param.Values) == 0 {
					continue
				}
			default:
				if param.Value == "" {
					continue
				}
			}
		}
		newParams = append(newParams, param)
	}

	t.Params = newParams

	out := new(bytes.Buffer)
	data := map[string]any{
		"Template": t.Template,
		"Params":   t.Params,
	}
	err = tmpl.Execute(out, data)

	return bytes.TrimSpace(out.Bytes()), err
}

// RenderResult renders the result template to instantiate the proxy
func (t *Template) RenderResult(renderMode int, other map[string]any) ([]byte, map[string]any, error) {
	values := t.Defaults(renderMode)
	if err := mergeMaps(other, values); err != nil {
		return nil, values, err
	}

	t.ModbusValues(renderMode, values)

	res := make(map[string]any)

	var usage string
	for k, v := range values {
		if strings.ToLower(k) == "usage" {
			usage = strings.ToLower(cast.ToString(v))
			break
		}
	}

	// TODO this is an utterly horrible hack
	//
	// When decoding the actual values ("other" parameter) into the
	// defaults-populated map, mismatching key case will create multiple
	// map entries for the same parameter.
	// The code below tries to select the best, i.e. non-empty, value for the
	// parameter and assigns it to the result key.
	// The actual key name is taken from the parameter to make it unique.
	// Since predefined properties are not matched by actual parameters using
	// ParamByName(), the lower case key name is used instead.
	// All keys *must* be assigned or rendering will create "<no value>" artifacts. For this reason,
	// deprecated parameters (that may still be rendered) must be evaluated, too.

	for key, val := range values {
		out := strings.ToLower(key)

		i, p := t.ParamByName(key)
		if i == -1 {
			if !slices.Contains(predefinedTemplateProperties, out) {
				return nil, values, fmt.Errorf("invalid key: %s", key)
			}
		} else {
			out = p.Name
		}

		// TODO move yamlQuote to explicit quoting in templates, see https://github.com/evcc-io/evcc/issues/10742

		switch typed := val.(type) {
		case []any:
			var list []string
			for _, v := range typed {
				list = append(list, p.yamlQuote(fmt.Sprintf("%v", v)))
			}
			if res[out] == nil || len(res[out].([]any)) == 0 {
				res[out] = list
			}

		case []string:
			var list []string
			for _, v := range typed {
				list = append(list, p.yamlQuote(v))
			}
			if res[out] == nil || len(res[out].([]string)) == 0 {
				res[out] = list
			}

		default:
			if res[out] == nil || res[out].(string) == "" {
				// prevent rendering nil interfaces as "<nil>" string
				var s string
				if val != nil {
					s = p.yamlQuote(fmt.Sprintf("%v", val))
				}

				// validate required fields from yaml
				if p.IsRequired() && p.IsZero(s) && (renderMode == RenderModeUnitTest || renderMode == RenderModeInstance && !testing.Testing()) {
					// validate required per usage
					if len(p.Usages) == 0 || slices.Contains(p.Usages, usage) {
						return nil, nil, fmt.Errorf("missing required `%s`", p.Name)
					}
				}

				// validate pattern if defined
				if s != "" && p.Pattern != nil && p.Pattern.Regex != "" {
					if err := p.Pattern.Validate(s); err != nil {
						return nil, nil, fmt.Errorf("%s: %w", p.Name, err)
					}
				}

				res[out] = s
			}
		}
	}

	tmpl, err := FuncMap(template.Must(baseTmpl.Clone())).Parse(t.Render)
	if err != nil {
		return nil, res, err
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, res)

	return bytes.TrimSpace(out.Bytes()), res, err
}
