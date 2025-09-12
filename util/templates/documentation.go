package templates

import (
	"bytes"
	_ "embed"
	"slices"
	"strconv"
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
	values := t.Defaults(RenderModeDocs)

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

	var modbusRender string
	modbusData := make(map[string]interface{})
	if modbusChoices := t.ModbusChoices(); len(modbusChoices) > 0 {
		if i, _ := t.ParamByName(ParamModbus); i > -1 {
			modbusTmpl, err := template.New("yaml").Funcs(sprig.FuncMap()).Parse(documentationModbusTmpl)
			if err != nil {
				panic(err)
			}

			t.ModbusValues(RenderModeDocs, modbusData)

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

	// all advanced params should be sorted to the end
	slices.SortStableFunc(filteredParams, func(i, j Param) int {
		if i.IsAdvanced() && !j.IsAdvanced() {
			return 1
		}
		if !i.IsAdvanced() && j.IsAdvanced() {
			return -1
		}
		return 0
	})

	data := map[string]interface{}{
		"Template":               t.Template,
		"ProductIdentifier":      product.Identifier(),
		"ProductBrand":           product.Brand,
		"ProductDescription":     product.Description.String(lang),
		"ProductGroup":           t.GroupTitle(lang),
		"Capabilities":           t.Capabilities,
		"Countries":              t.Countries,
		"Requirements":           t.Requirements.EVCC,
		"RequirementDescription": t.Requirements.Description.String(lang),
		"Params":                 filteredParams,
		"AdvancedParams":         hasAdvancedParams,
		"Usages":                 t.Usages(),
		"Modbus":                 modbusRender,
		"ModbusData":             modbusData,
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
