package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/util"
	"golang.org/x/exp/slices"
)

// Template describes is a proxy device for use with cli and automated testing
type Template struct {
	TemplateDefinition

	ConfigDefaults ConfigDefaults

	Lang string

	title  string
	titles []string
}

// UpdateParamWithDefaults adds default values to specific param name entries
func (t *Template) UpdateParamsWithDefaults() error {
	for i, p := range t.Params {
		if p.ValueType == "" || (p.ValueType != "" && !slices.Contains(ValidParamValueTypes, p.ValueType)) {
			t.Params[i].ValueType = ParamValueTypeString
		}

		if index, resultMapItem := t.ConfigDefaults.ParamByName(strings.ToLower(p.Name)); index > -1 {
			t.Params[i].OverwriteProperties(resultMapItem)
		}
	}

	return nil
}

// validate the template (only rudimentary for now)
func (t *Template) Validate() error {
	for _, c := range t.Capabilities {
		if !slices.Contains(ValidCapabilities, c) {
			return fmt.Errorf("invalid capability '%s' in template %s", c, t.Template)
		}
	}

	for _, r := range t.Requirements.EVCC {
		if !slices.Contains(ValidRequirements, r) {
			return fmt.Errorf("invalid requirement '%s' in template %s", r, t.Template)
		}
	}

	for _, p := range t.Params {
		switch p.Name {
		case ParamUsage:
			for _, c := range p.Choice {
				if !slices.Contains(ValidUsageChoices, c) {
					return fmt.Errorf("invalid usage choice '%s' in template %s", c, t.Template)
				}
			}
		case ParamModbus:
			for _, c := range p.Choice {
				if !slices.Contains(ValidModbusChoices, c) {
					return fmt.Errorf("invalid modbus choice '%s' in template %s", c, t.Template)
				}
			}
		}

		if p.ValueType != "" && !slices.Contains(ValidParamValueTypes, p.ValueType) {
			return fmt.Errorf("invalid value type '%s' in template %s", p.ValueType, t.Template)
		}

		for _, d := range p.Dependencies {
			if !slices.Contains(ValidDependencies, d.Check) {
				return fmt.Errorf("invalid dependency check '%s' in template %s", d.Check, t.Template)
			}
		}
	}

	return nil
}

// set the language title by combining all product titles
func (t *Template) SetCombinedTitle() {
	if len(t.titles) == 0 {
		t.resolveTitles()
	}

	t.title = strings.Join(t.titles, "/")
}

// set the title for this templates
func (t *Template) SetTitle(title string) {
	t.title = title
}

// return the title for this template
func (t *Template) Title() string {
	return t.title
}

// return a language specific product title
func (t *Template) ProductTitle(p Product) string {
	return strings.TrimSpace(fmt.Sprintf("%s %s", p.Brand, p.Description.String(t.Lang)))
}

// return the language specific product titles
func (t *Template) Titles(lang string) []string {
	t.Lang = lang

	if len(t.titles) == 0 {
		t.resolveTitles()
	}

	return t.titles
}

// set the language specific product titles
func (t *Template) resolveTitles() {
	for _, p := range t.Products {
		t.titles = append(t.titles, t.ProductTitle(p))
	}
}

// add the referenced base Params and overwrite existing ones
func (t *Template) ResolvePresets() error {
	currentParams := make([]Param, len(t.Params))
	copy(currentParams, t.Params)
	t.Params = []Param{}
	for _, p := range currentParams {
		if p.Preset != "" {
			base, ok := t.ConfigDefaults.Presets[p.Preset]
			if !ok {
				return fmt.Errorf("could not find preset definition: %s", p.Preset)
			}

			t.Params = append(t.Params, base.Params...)
			continue
		}

		if i, _ := t.ParamByName(p.Name); i > -1 {
			t.Params[i].OverwriteProperties(p)
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

	_, ok := t.ConfigDefaults.DeviceGroups[t.Group]
	if !ok {
		return fmt.Errorf("could not find devicegroup definition: %s", t.Group)
	}

	return nil
}

// return the language specific group title
func (t *Template) GroupTitle() string {
	tl := t.ConfigDefaults.DeviceGroups[t.Group]
	return tl.String(t.Lang)
}

// Defaults returns a map of default values for the template
func (t *Template) Defaults(renderMode string) map[string]interface{} {
	values := make(map[string]interface{})
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

// RenderProxy renders the proxy template
func (t *Template) RenderProxyWithValues(values map[string]interface{}, lang string) ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(proxyTmpl)
	if err != nil {
		panic(err)
	}

	t.ModbusParams("", values)

	for index, p := range t.Params {
		for k, v := range values {
			if p.Name != k {
				continue
			}

			switch p.ValueType {
			case ParamValueTypeStringList:
				for _, e := range v.([]string) {
					t.Params[index].Values = append(p.Values, yamlQuote(e))
				}
			default:
				switch v := v.(type) {
				case string:
					t.Params[index].Value = yamlQuote(v)
				case int:
					t.Params[index].Value = fmt.Sprintf("%d", v)
				}
			}
		}
	}

	// remove params with no values
	var newParams []Param
	for _, param := range t.Params {
		if !param.Required {
			switch param.ValueType {
			case ParamValueTypeStringList:
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
	data := map[string]interface{}{
		"Template": t.Template,
		"Params":   t.Params,
	}
	err = tmpl.Execute(out, data)

	return bytes.TrimSpace(out.Bytes()), err
}

// RenderResult renders the result template to instantiate the proxy
func (t *Template) RenderResult(renderMode string, other map[string]interface{}) ([]byte, map[string]interface{}, error) {
	values := t.Defaults(renderMode)
	if err := util.DecodeOther(other, &values); err != nil {
		return nil, values, err
	}

	t.ModbusValues(renderMode, values)

	// add the common templates
	for _, v := range t.ConfigDefaults.Presets {
		if !strings.Contains(t.Render, v.Render) {
			t.Render += "\n" + v.Render
		}
	}

	res := make(map[string]interface{})

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
	// All keys *must* be assigned or rendering will create "<no value>" artifacts.

	for key, val := range values {
		out := strings.ToLower(key)

		if i, p := t.ParamByName(key); i == -1 {
			if !slices.Contains(predefinedTemplateProperties, strings.ToLower(key)) {
				return nil, values, fmt.Errorf("invalid key: %s", key)
			}
		} else {
			out = p.Name
		}

		switch typed := val.(type) {
		case []interface{}:
			var list []string
			for _, v := range typed {
				list = append(list, yamlQuote(fmt.Sprintf("%v", v)))
			}
			if res[out] == nil || len(res[out].([]interface{})) == 0 {
				res[out] = list
			}

		case []string:
			var list []string
			for _, v := range typed {
				list = append(list, yamlQuote(v))
			}
			if res[out] == nil || len(res[out].([]string)) == 0 {
				res[out] = list
			}

		default:
			if res[out] == nil || res[out].(string) == "" {
				res[out] = yamlQuote(fmt.Sprintf("%v", val))
			}
		}
	}

	tmpl := template.New("yaml")
	funcMap := template.FuncMap{
		// include function
		// copied from: https://github.com/helm/helm/blob/8648ccf5d35d682dcd5f7a9c2082f0aaf071e817/pkg/engine/engine.go#L147-L154
		"include": func(name string, data interface{}) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
	}

	tmpl, err := tmpl.Funcs(template.FuncMap(sprig.FuncMap())).Funcs(funcMap).Parse(t.Render)
	if err != nil {
		return nil, res, err
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, res)

	return bytes.TrimSpace(out.Bytes()), res, err
}
