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

type paramStruct struct {
	VarName, Signature string
}

type funcStruct struct {
	Signature, Function, VarName, ReturnTypes string
	Params                                    []paramStruct
}

type typeStruct struct {
	Type      string
	Functions []funcStruct
}

func getTypeImport(t reflect.Type) string {
	n := t.Name()
	if p := t.PkgPath(); p != "" {
		if s := strings.Split(p, "github.com/evcc-io/evcc/"); len(s) == 2 {
			return fmt.Sprintf("%s.%s", s[1], n)
		} else {
			return fmt.Sprintf("%s.%s", p, n)
		}
	}
	return n
}

func generate(out io.Writer) error {
	tmpl, err := template.New("gen").Funcs(sprig.FuncMap()).Parse(srcTmpl)
	if err != nil {
		fmt.Printf("invalid template: %s", err)
		os.Exit(2)
	}

	var types []typeStruct

	for _, typ := range []reflect.Type{
		reflect.TypeFor[api.Meter](),
		reflect.TypeFor[api.BatteryCapacity](),
		reflect.TypeFor[api.SocLimiter](),
		reflect.TypeFor[api.BatteryController](),
		reflect.TypeFor[api.BatterySocLimiter](),
		reflect.TypeFor[api.BatteryPowerLimiter](),
		reflect.TypeFor[api.PhasePowers](),
		reflect.TypeFor[api.PhaseGetter](),
		reflect.TypeFor[api.ChargeController](),
		reflect.TypeFor[api.CurrentController](),
		reflect.TypeFor[api.PhaseSwitcher](),
		reflect.TypeFor[api.Battery](),
		reflect.TypeFor[api.ChargeState](),
		reflect.TypeFor[api.MeterImport](),
		reflect.TypeFor[api.MeterExport](),
		reflect.TypeFor[api.PhaseCurrents](),
		reflect.TypeFor[api.PhaseVoltages](),
		reflect.TypeFor[api.MaxACPowerGetter](),
		reflect.TypeFor[api.CurrentGetter](),
		reflect.TypeFor[api.Curtailer](),
		reflect.TypeFor[api.Resurrector](),
		reflect.TypeFor[api.VehicleOdometer](),
		reflect.TypeFor[api.VehicleRange](),
		reflect.TypeFor[api.VehicleClimater](),
		reflect.TypeFor[api.VehicleFinishTimer](),
		reflect.TypeFor[api.VehiclePosition](),
		reflect.TypeFor[api.Identifier](),
		reflect.TypeFor[api.ChargerEx](),
		reflect.TypeFor[api.ChargeRater](),
		reflect.TypeFor[api.StatusReasoner](),
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

			functions = append(functions, funcStruct{
				VarName:     strings.ToLower(lastPart[:1]) + lastPart[1:] + strconv.Itoa(methodIndex),
				Signature:   fmt.Sprintf("func(%s) (%s)", strings.Join(parameters, ", "), strings.Join(returns, ", ")),
				Function:    m.Name,
				Params:      params,
				ReturnTypes: fmt.Sprintf("(%s)", strings.Join(returns, ",")),
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
	name := "../../api/implement/implementations.go"

	generated := new(bytes.Buffer)
	if err := generate(generated); err != nil {
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
