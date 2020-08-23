package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	combinations "github.com/mxschmitt/golang-combinations"
	"github.com/spf13/pflag"
)

var srcTmpl = `
package {{.Package}}

// This file has been generated - do not modify

import (
	"{{.API}}"
)

{{define "case"}}
{{- $combo := .Combo}}
{{- $idx := 0}}
	case{{range $typ, $def := .Types}}{{if gt $idx 0}} &&{{else}}{{$idx = 1}}{{end}} {{$def.VarName}} {{if contains $combo $typ}}!={{else}}=={{end}} nil{{end}}:
		return &struct{
			{{.BaseType}}
{{- range $typ, $def := .Types}}{{if contains $combo $typ}}
			{{$typ}}{{end}}{{end}}
		}{
			Meter: base,
{{- range $typ, $def := .Types}}{{if contains $combo $typ}}
			{{$def.ShortType}}: &{{$def.VarName}}Impl{
				{{$def.VarName}}: {{$def.VarName}},
			},{{end}}{{end}}
		}
{{end -}}

func decorate(base {{.BaseType}}{{range ordered}}, {{.VarName}} func() {{slice .Signature 7}}{{end}}) {{.BaseType}} {
	switch {
{{- $basetype := .BaseType}}
{{- $types := .Types}}
{{- $idx := 0}}
	case{{range $typ, $def := .Types}}{{if gt $idx 0}} &&{{else}}{{$idx = 1}}{{end}} {{$def.VarName}} == nil{{end}}:
		return base
{{range $combo := .Combinations}}{{template "case" dict "BaseType" $basetype "Types" $types "Combo" $combo}}{{end}}	}

	return nil
}

{{range .Types}}type {{.VarName}}Impl struct {
	{{.VarName}} {{.Signature}}
}

func (impl *{{.VarName}}Impl) {{.Function}}{{slice .Signature 4}} {
	return impl.{{.VarName}}()
}

{{end}}
`

type dynamicType struct {
	typ, function, signature string
}

type typeStruct struct {
	Type, ShortType, Signature, Function, VarName string
}

func generate(baseType string, dynamicTypes ...dynamicType) {
	types := make(map[string]typeStruct, len(dynamicTypes))
	combos := make([]string, 0)

	tmpl, err := template.New("gen").Funcs(template.FuncMap{
		// dict combines key value pairs for passing structs into templates
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		// contains checks if slice contains string
		"contains": func(combo []string, typ string) bool {
			for _, v := range combo {
				if v == typ {
					return true
				}
			}
			return false
		},
		// ordered checks if slice ordered string
		"ordered": func() []typeStruct {
			ordered := make([]typeStruct, 0)
			for _, k := range dynamicTypes {
				ordered = append(ordered, types[k.typ])
			}

			return ordered
		},
	}).Parse(srcTmpl)
	if err != nil {
		panic(err)
	}

	for _, dt := range dynamicTypes {
		parts := strings.SplitN(dt.typ, ".", 2)
		varName := strings.ToLower(parts[1][:1]) + parts[1][1:]

		types[dt.typ] = typeStruct{
			Type:      dt.typ,
			ShortType: parts[1],
			Signature: dt.signature,
			Function:  dt.function,
			VarName:   varName,
		}

		combos = append(combos, dt.typ)
	}

	vars := struct {
		Package, API string
		BaseType     string
		Types        map[string]typeStruct
		Combinations [][]string
	}{
		Package:      "meter",
		API:          "github.com/andig/evcc/api",
		BaseType:     baseType,
		Types:        types,
		Combinations: combinations.All(combos),
	}

	tmpl.Execute(os.Stdout, vars)
}

var (
	base  = pflag.StringP("base", "b", "", "base type")
	types = pflag.StringArrayP("type", "t", nil, "comma-separated list of type definitions")
)

// Usage prints flags usage
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of decorate:\n")
	fmt.Fprintf(os.Stderr, "\ndecorate [flags] -type interface,interface function,function signature\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	pflag.PrintDefaults()
}

func main() {
	// generate("api.Meter", []dynamicType{
	// 	dynamicType{"api.MeterEnergy", "TotalEnergy", "func() (float64, error)"},
	// 	dynamicType{"api.MeterCurrent", "Currents", "func() (float64, float64, float64, error)"},
	// }...)
	pflag.Usage = Usage
	pflag.Parse()

	if *base == "" || len(*types) == 0 {
		Usage()
		os.Exit(2)
	}

	var dynamicTypes []dynamicType
	for _, v := range *types {
		split := strings.SplitN(v, ",", 3)
		dt := dynamicType{split[0], split[1], split[2]}
		dynamicTypes = append(dynamicTypes, dt)
	}

	generate(*base, dynamicTypes...)
}
