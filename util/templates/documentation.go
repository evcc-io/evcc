package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

//go:embed documentation.tpl
var documentationTmpl string

//go:embed documentation_modbus.tpl
var documentationModbusTmpl string

// RenderDocumentation renders the documentation template
func (t *Template) RenderDocumentation(product Product, lang string) ([]byte, error) {
	values := t.Defaults(TemplateRenderModeDocs)

	for index, p := range t.Params {
		for k, v := range values {
			if p.Name != k {
				continue
			}

			switch p.Type {
			case TypeStringList:
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

	var modbusRender string
	if modbusChoices := t.ModbusChoices(); len(modbusChoices) > 0 {
		if i, _ := t.ParamByName(ParamModbus); i > -1 {
			modbusTmpl, err := template.New("yaml").Funcs(sprig.TxtFuncMap()).Parse(documentationModbusTmpl)
			if err != nil {
				panic(err)
			}

			modbusData := make(map[string]interface{})
			t.ModbusValues(TemplateRenderModeDocs, modbusData)

			out := new(bytes.Buffer)
			if err := modbusTmpl.Execute(out, modbusData); err != nil {
				panic(err)
			}

			modbusRender = out.String()
		}
	}

	var hasAdvancedParams bool

	// remove usage and deprecated from params and check if there are advanced params
	var filteredParams []Param
	for _, param := range t.Params {
		if param.IsDeprecated() || param.Name == ParamUsage {
			continue
		}

		if param.IsAdvanced() {
			hasAdvancedParams = true
		}

		filteredParams = append(filteredParams, param)
	}

	data := map[string]interface{}{
		"Template":               t.Template,
		"ProductBrand":           product.Brand,
		"ProductDescription":     product.Description.String(lang),
		"ProductGroup":           t.GroupTitle(lang),
		"Capabilities":           t.Capabilities,
		"Requirements":           t.Requirements.EVCC,
		"RequirementDescription": t.Requirements.Description.String(lang),
		"Params":                 filteredParams,
		"AdvancedParams":         hasAdvancedParams,
		"Usages":                 t.Usages(),
		"Modbus":                 modbusRender,
	}

	out := new(bytes.Buffer)

	funcMap := template.FuncMap{
		"localize": localize(lang),
	}

	tmpl, err := FuncMap(template.New("yaml")).Funcs(funcMap).Parse(documentationTmpl)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(out, data)

	return []byte(trimLines(out.String())), err
}

func localize(lang string) func(TextLanguage) string {
	return func(s TextLanguage) string {
		return strings.TrimSpace(s.String(lang))
	}
}
