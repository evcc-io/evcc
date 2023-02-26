package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

//go:embed documentation.tpl
var documentationTmpl string

//go:embed documentation_modbus.tpl
var documentationModbusTmpl string

// RenderProxy renders the proxy template
func (t *Template) RenderDocumentation(product Product, values map[string]interface{}, lang string) ([]byte, error) {
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

	usages := t.Usages()
	modbusChoices := t.ModbusChoices()
	modbusRender := ""
	if len(modbusChoices) > 0 {
		if i, _ := t.ParamByName(ParamModbus); i > -1 {
			modbusTmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(documentationModbusTmpl)
			if err != nil {
				panic(err)
			}

			modbusData := map[string]interface{}{}
			t.ModbusValues(TemplateRenderModeDocs, modbusData)

			modbusOut := new(bytes.Buffer)

			err = modbusTmpl.Execute(modbusOut, modbusData)
			if err != nil {
				panic(err)
			}

			modbusRender = modbusOut.String()
		}
	}

	// remove usage and deprecated from params and check if there are advanced params
	var hasAdvancedParam bool
	var newParams []Param
	for _, param := range t.Params {
		if param.IsDeprecated() || param.Name == ParamUsage {
			continue
		}

		if param.IsAdvanced() {
			hasAdvancedParam = true
		}

		newParams = append(newParams, param)
	}
	t.Params = newParams

	out := new(bytes.Buffer)
	data := map[string]interface{}{
		"Template":               t.Template,
		"ProductBrand":           product.Brand,
		"ProductDescription":     product.Description.String(lang),
		"ProductGroup":           t.GroupTitle(lang),
		"Capabilities":           t.Capabilities,
		"Requirements":           t.Requirements.EVCC,
		"RequirementDescription": t.Requirements.Description.String(lang),
		"Params":                 t.Params,
		"AdvancedParams":         hasAdvancedParam,
		"Usages":                 usages,
		"Modbus":                 modbusRender,
	}

	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(documentationTmpl)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(out, data)

	// trim empty lines with whitespace
	regex, _ := regexp.Compile("\n *\n")
	string := regex.ReplaceAllString(out.String(), "\n\n")
	result := new(bytes.Buffer)
	result.WriteString(string)

	return result.Bytes(), err
}
