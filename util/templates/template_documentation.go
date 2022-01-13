package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/thoas/go-funk"
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

	usages := t.Usages()
	modbusChoices := t.ModbusChoices()
	modbusRender := ""
	if len(modbusChoices) > 0 {
		if i, modbusParam := t.ParamByName(ParamModbus); i > -1 {
			modbusTmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(documentationModbusTmpl)
			if err != nil {
				panic(err)
			}

			modbusData := map[string]interface{}{
				"id": modbusParam.ID,
			}

			// modbus defaults
			for k, v := range map[string]interface{}{
				ModbusParamNameId:       ModbusParamValueId,
				ModbusParamNameHost:     ModbusParamValueHost,
				ModbusParamNamePort:     ModbusParamValuePort,
				ModbusParamNameDevice:   ModbusParamValueDevice,
				ModbusParamNameBaudrate: ModbusParamValueBaudrate,
				ModbusParamNameComset:   ModbusParamValueComset,
			} {
				modbusData[k] = v
			}
			if modbusParam.ID != 0 {
				modbusData[ModbusParamNameId] = modbusParam.ID
			}
			if modbusParam.Port != 0 {
				modbusData[ModbusParamNameDevice] = modbusParam.Port
			}
			if modbusParam.Baudrate != 0 {
				modbusData[ModbusParamNameBaudrate] = modbusParam.Baudrate
			}
			if modbusParam.Comset != "" {
				modbusData[ModbusParamNameComset] = modbusParam.Comset
			}

			if funk.ContainsString(modbusChoices, ModbusChoiceRS485) {
				modbusData[ModbusRS485Serial] = true
				modbusData[ModbusRS485TCPIP] = true
			}
			if funk.ContainsString(modbusChoices, ModbusChoiceTCPIP) {
				modbusData[ModbusTCPIP] = true
			}

			modbusOut := new(bytes.Buffer)

			err = modbusTmpl.Execute(modbusOut, modbusData)
			if err != nil {
				panic(err)
			}

			modbusRender = modbusOut.String()
		}
	}

	// remove usage from params and check if there are advanced params
	var hasAdvancedParam bool
	var newParams []Param
	for _, param := range t.Params {
		if param.Name == ParamUsage {
			continue
		}
		if param.Advanced {
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
		"ProductGroup":           t.GroupTitle(),
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

	return bytes.TrimSpace(out.Bytes()), err
}
