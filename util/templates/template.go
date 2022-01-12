package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/templates/definition"
	"github.com/evcc-io/evcc/util"
	"github.com/thoas/go-funk"
	"gopkg.in/yaml.v3"
)

// Template describes is a proxy device for use with cli and automated testing
type Template struct {
	TemplateDefinition

	Lang string

	title  string
	titles []string
}

func (t *Template) Validate() error {
	for _, p := range t.Params {
		switch p.Name {
		case ParamUsage:
			for _, c := range p.Choice {
				if !funk.ContainsString(ValidUsageChoices, c) {
					return fmt.Errorf("invalid usage choice '%s' in template %s", c, t.Template)
				}
			}
		case ParamModbus:
			for _, c := range p.Choice {
				if !funk.ContainsString(ValidModbusChoices, c) {
					return fmt.Errorf("invalid modbus choice '%s' in template %s", c, t.Template)
				}
			}
		}
	}

	return nil
}

func (t *Template) SetCombinedTitle() {
	if len(t.titles) == 0 {
		t.resolveTitles()
	}

	title := ""
	for _, t := range t.titles {
		if title != "" {
			title += "/"
		}
		title += t
	}
	t.title = title
}

func (t *Template) SetTitle(title string) {
	t.title = title
}

func (t *Template) Title() string {
	return t.title
}

func (t *Template) ProductTitle(p Product) string {
	title := ""

	if p.Brand != "" {
		title += p.Brand
	}

	description := p.Description.String(t.Lang)
	if description != "" {
		if title != "" {
			title += " "
		}
		title += description
	}

	return title
}

func (t *Template) Titles(lang string) []string {
	t.Lang = lang

	if len(t.titles) == 0 {
		t.resolveTitles()
	}

	return t.titles
}

func (t *Template) resolveTitles() {
	var titles []string

	for _, p := range t.Products {
		titles = append(titles, t.ProductTitle(p))
	}

	t.titles = titles
}

// add the referenced base Params and overwrite existing ones
func (t *Template) ResolveParamBases() error {
	if paramBaseList == nil {
		err := yaml.Unmarshal([]byte(definition.ParamBaseListDefinition), &paramBaseList)
		if err != nil {
			return fmt.Errorf("Error: failed to parse paramBasesDefinition: %v\n", err)
		}
	}

	currentParams := make([]Param, len(t.Params))
	copy(currentParams, t.Params)
	t.Params = []Param{}
	for _, p := range currentParams {
		if p.Base != "" {
			base, ok := paramBaseList[p.Base]
			if !ok {
				return fmt.Errorf("Error: Could not find parambase definition: %s\n", p.Base)
			}

			t.Params = append(t.Params, base.Params...)
			continue
		}

		if i, item := t.ParamByName(p.Name); item != nil {
			// we only allow overwriting a few fields
			if p.Default != "" {
				t.Params[i].Default = p.Default
			}
			if p.Example != "" {
				t.Params[i].Example = p.Example
			}
		} else {
			t.Params = append(t.Params, p)
		}
	}

	return nil
}

func (t *Template) ResolveGroup() error {
	if groupList == nil {
		err := yaml.Unmarshal([]byte(definition.GroupListDefinition), &groupList)
		if err != nil {
			return fmt.Errorf("Error: failed to parse deviceGroupListDefinition: %v\n", err)
		}
	}

	if t.Group == "" {
		return nil
	}

	_, ok := groupList[t.Group]
	if !ok {
		return fmt.Errorf("Error: Could not find devicegroup definition: %s\n", t.Group)
	}

	return nil
}

func (t *Template) GroupTitle() string {
	tl := groupList[t.Group]
	return tl.String(t.Lang)
}

// Defaults returns a map of default values for the template
func (t *Template) Defaults(renderMode string) map[string]interface{} {
	values := make(map[string]interface{})
	for _, p := range t.Params {
		switch p.ValueType {
		case ParamValueTypeStringList:
			values[p.Name] = []string{}
		case ParamValueTypeChargeModes:
			values[p.Name] = ""
		default:
			if p.Test != "" {
				values[p.Name] = p.Test
			} else if p.Example != "" && funk.ContainsString([]string{TemplateRenderModeDocs, TemplateRenderModeUnitTest}, renderMode) {
				values[p.Name] = p.Example
			} else {
				values[p.Name] = p.Default // may be empty
			}
		}
	}

	return values
}

// return the param with the given name
func (t *Template) ParamByName(name string) (int, *Param) {
	for i, p := range t.Params {
		if p.Name == name {
			return i, &p
		}
	}
	return 0, nil
}

// Usages returns the list of supported usages
func (t *Template) Usages() []string {
	if _, p := t.ParamByName(ParamUsage); p != nil {
		return p.Choice
	}

	return nil
}

func (t *Template) ModbusChoices() []string {
	if _, p := t.ParamByName(ParamModbus); p != nil {
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

	t.ModbusParams(values)

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
	for _, v := range paramBaseList {
		if !strings.Contains(t.Render, v.Render) {
			t.Render = fmt.Sprintf("%s\n%s", t.Render, v.Render)
		}
	}

	for item, p := range values {
		_, param := t.ParamByName(item)
		if param == nil && !funk.ContainsString(predefinedTemplateProperties, item) {
			return nil, values, fmt.Errorf("invalid element 'name: %s'", item)
		}

		switch p := p.(type) {
		case []interface{}:
			var list []string
			for _, v := range p {
				list = append(list, yamlQuote(fmt.Sprintf("%v", v)))
			}
			values[item] = list
		case []string:
			var list []string
			for _, v := range p {
				list = append(list, yamlQuote(v))
			}
			values[item] = list
		default:
			values[item] = yamlQuote(fmt.Sprintf("%v", p))
		}
	}

	tmpl := template.New("yaml")
	var funcMap template.FuncMap = map[string]interface{}{}
	// copied from: https://github.com/helm/helm/blob/8648ccf5d35d682dcd5f7a9c2082f0aaf071e817/pkg/engine/engine.go#L147-L154
	funcMap["include"] = func(name string, data interface{}) (string, error) {
		buf := bytes.NewBuffer(nil)
		if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	tmpl, err := tmpl.Funcs(template.FuncMap(sprig.FuncMap())).Funcs(funcMap).Parse(t.Render)
	if err != nil {
		return nil, values, err
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, values)

	return bytes.TrimSpace(out.Bytes()), values, err
}
