package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/api"
	"golang.org/x/tools/imports"
)

//go:generate go tool implement

//go:embed implement.tpl
var srcTmpl string

//go:embed capabilities.tpl
var capsTmpl string

type paramStruct struct {
	VarName, Signature string
}

type funcStruct struct {
	Signature, Function, VarName, ReturnTypes string
	Params                                    []paramStruct
	// Results are the non-error return values, ReturnExpr the matching return statement
	Results    []paramStruct
	ReturnExpr string
	HasError   bool
}

type typeStruct struct {
	Type      string
	Functions []funcStruct
}

func getTypeImport(t reflect.Type) string {
	n := t.Name()
	if n == "" {
		// unnamed type, e.g. []string
		return t.String()
	}
	if p := t.PkgPath(); p != "" {
		if s := strings.Split(p, "github.com/evcc-io/evcc/"); len(s) == 2 {
			return fmt.Sprintf("%s.%s", s[1], n)
		} else {
			return fmt.Sprintf("%s.%s", p, n)
		}
	}
	return n
}

func generate(out io.Writer, src string) error {
	tmpl, err := template.New("gen").Funcs(sprig.FuncMap()).Parse(src)
	if err != nil {
		fmt.Printf("invalid template: %s", err)
		os.Exit(2)
	}

	var types []typeStruct

	for _, typ := range []reflect.Type{
		reflect.TypeFor[api.Battery](),
		reflect.TypeFor[api.BatteryCapacity](),
		reflect.TypeFor[api.BatteryController](),
		reflect.TypeFor[api.BatteryPowerLimiter](),
		reflect.TypeFor[api.BatterySocLimiter](),
		reflect.TypeFor[api.ChargeController](),
		reflect.TypeFor[api.ChargeRater](),
		reflect.TypeFor[api.ChargerEx](),
		reflect.TypeFor[api.ChargeState](),
		reflect.TypeFor[api.CurrentController](),
		reflect.TypeFor[api.CurrentGetter](),
		reflect.TypeFor[api.CurrentLimiter](),
		reflect.TypeFor[api.Curtailer](),
		reflect.TypeFor[api.Dimmer](),
		reflect.TypeFor[api.Identifier](),
		reflect.TypeFor[api.MaxACPowerGetter](),
		reflect.TypeFor[api.Meter](),
		reflect.TypeFor[api.MeterEnergy](),
		reflect.TypeFor[api.MeterReturnEnergy](),
		reflect.TypeFor[api.PhaseCurrents](),
		reflect.TypeFor[api.PhaseGetter](),
		reflect.TypeFor[api.PhasePowers](),
		reflect.TypeFor[api.PhaseSwitcher](),
		reflect.TypeFor[api.PhaseVoltages](),
		reflect.TypeFor[api.PowerLimiter](),
		reflect.TypeFor[api.Resurrector](),
		reflect.TypeFor[api.SocLimiter](),
		reflect.TypeFor[api.StatusReasoner](),
		reflect.TypeFor[api.VehicleClimater](),
		reflect.TypeFor[api.VehicleFinishTimer](),
		reflect.TypeFor[api.VehicleOdometer](),
		reflect.TypeFor[api.VehiclePosition](),
		reflect.TypeFor[api.VehicleRange](),
	} {
		lastPart := typ.Name()
		var functions []funcStruct

		for methodIndex := 0; methodIndex < typ.NumMethod(); methodIndex++ {
			m := typ.Method(methodIndex)

			var params []paramStruct
			for paramIndex := 0; paramIndex < m.Type.NumIn(); paramIndex++ {
				p := m.Type.In(paramIndex)

				params = append(params, paramStruct{
					VarName:   "p" + strconv.Itoa(paramIndex),
					Signature: getTypeImport(p),
				})
			}

			var parameters []string
			for input := range m.Type.Ins() {
				parameters = append(parameters, getTypeImport(input))
			}

			var returns []string
			for output := range m.Type.Outs() {
				returns = append(returns, getTypeImport(output))
			}

			// split trailing error from the actual results
			results := make([]paramStruct, 0, len(returns))
			hasError := len(returns) > 0 && returns[len(returns)-1] == "error"
			for i, r := range returns {
				if hasError && i == len(returns)-1 {
					continue
				}
				results = append(results, paramStruct{
					VarName:   "r" + strconv.Itoa(i),
					Signature: r,
				})
			}

			var returnVars []string
			for _, r := range results {
				returnVars = append(returnVars, r.VarName)
			}
			if hasError {
				returnVars = append(returnVars, "err")
			}

			functions = append(functions, funcStruct{
				VarName:     strings.ToLower(lastPart[:1]) + lastPart[1:] + strconv.Itoa(methodIndex),
				Signature:   fmt.Sprintf("func(%s) (%s)", strings.Join(parameters, ", "), strings.Join(returns, ", ")),
				Function:    m.Name,
				Params:      params,
				ReturnTypes: fmt.Sprintf("(%s)", strings.Join(returns, ",")),
				Results:     results,
				ReturnExpr:  strings.Join(returnVars, ", "),
				HasError:    hasError,
			})
		}

		types = append(types, typeStruct{
			Type:      typ.Name(),
			Functions: functions,
		})
	}

	vars := struct {
		Types []typeStruct
	}{
		Types: types,
	}

	return tmpl.Execute(out, vars)
}

func main() {
	write("../../api/implement/implementations.go", srcTmpl)
	write("../../devicehost/capabilities.go", capsTmpl)
}

func write(name, src string) {
	generated := new(bytes.Buffer)
	if err := generate(generated, src); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	formatted, err := format.Source(generated.Bytes())
	if err != nil {
		formatted = generated.Bytes()
	}

	formatted, err = imports.Process(name, formatted, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	file, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	if _, err := file.Write(formatted); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}
