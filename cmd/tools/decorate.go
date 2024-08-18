package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/go-sprout/sprout"
	combinations "github.com/mxschmitt/golang-combinations"
	"github.com/spf13/pflag"
	"golang.org/x/tools/imports"
)

//go:embed decorate.tpl
var srcTmpl string

type dynamicType struct {
	typ, function, signature string
}

type typeStruct struct {
	Type, ShortType, Signature, Function, VarName, ReturnTypes string
	Params                                                     []string
}

var dependents = map[string][]string{
	"api.Meter":         {"api.MeterEnergy", "api.PhaseCurrents", "api.PhaseVoltages", "api.PhasePowers"},
	"api.PhaseCurrents": {"api.PhasePowers"}, // phase powers are only used to determine currents sign
	"api.PhaseSwitcher": {"api.PhaseGetter"},
	"api.Battery":       {"api.BatteryCapacity", "api.BatteryController"},
}

// hasIntersection returns if the slices intersect
func hasIntersection[T comparable](a, b []T) bool {
	for _, el := range a {
		if slices.Contains(b, el) {
			return true
		}
	}
	return false
}

func generate(out io.Writer, packageName, functionName, baseType string, dynamicTypes ...dynamicType) error {
	types := make(map[string]typeStruct, len(dynamicTypes))
	combos := make([]string, 0)

	tmpl, err := template.New("gen").Funcs(sprout.FuncMap()).Funcs(template.FuncMap{
		// contains checks if slice contains string
		"contains": slices.Contains[[]string, string],
		// ordered returns a slice of typeStructs ordered by dynamicType
		"ordered": func() []typeStruct {
			ordered := make([]typeStruct, 0)
			for _, k := range dynamicTypes {
				ordered = append(ordered, types[k.typ])
			}

			return ordered
		},
	}).Parse(srcTmpl)
	if err != nil {
		return err
	}

	for _, dt := range dynamicTypes {
		parts := strings.SplitN(dt.typ, ".", 2)
		lastPart := parts[len(parts)-1]

		openingBrace := strings.Index(dt.signature, "(")
		closingBrace := strings.Index(dt.signature, ")")
		paramsStr := dt.signature[openingBrace+1 : closingBrace]

		var params []string
		if paramsStr = strings.TrimSpace(paramsStr); len(paramsStr) > 0 {
			params = strings.Split(paramsStr, ",")
		}

		types[dt.typ] = typeStruct{
			Type:        dt.typ,
			ShortType:   lastPart,
			VarName:     strings.ToLower(lastPart[:1]) + lastPart[1:],
			Signature:   dt.signature,
			Function:    dt.function,
			Params:      params,
			ReturnTypes: dt.signature[closingBrace+1:],
		}

		combos = append(combos, dt.typ)
	}

	returnType := *ret
	if returnType == "" {
		returnType = baseType
	}

	shortBase := strings.TrimLeft(baseType, "*")
	if baseTypeParts := strings.SplitN(baseType, ".", 2); len(baseTypeParts) > 1 {
		shortBase = baseTypeParts[1]
	}

	validCombos := make([][]string, 0)
COMBO:
	for _, c := range combinations.All(combos) {
		for master, details := range dependents {
			if returnType != master && !slices.Contains(c, master) && hasIntersection(c, details) {
				continue COMBO
			}
		}
		validCombos = append(validCombos, c)
	}

	vars := struct {
		API                 string
		Package, Function   string
		BaseType, ShortBase string
		ReturnType          string
		Types               map[string]typeStruct
		Combinations        [][]string
	}{
		API:          "github.com/evcc-io/evcc/api",
		Package:      packageName,
		Function:     functionName,
		BaseType:     baseType,
		ShortBase:    shortBase,
		ReturnType:   returnType,
		Types:        types,
		Combinations: validCombos,
	}

	return tmpl.Execute(out, vars)
}

var (
	target   = pflag.StringP("out", "o", "", "output file")
	pkg      = pflag.StringP("package", "p", "", "package name")
	function = pflag.StringP("function", "f", "decorate", "function name")
	base     = pflag.StringP("base", "b", "", "base type")
	ret      = pflag.StringP("return", "r", "", "return type")
	types    = pflag.StringArrayP("type", "t", nil, "comma-separated list of type definitions")
)

// Usage prints flags usage
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of decorate:\n")
	fmt.Fprintf(os.Stderr, "\ndecorate [flags] -type interface,interface function,function signature\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	pflag.PrintDefaults()
}

func main() {
	pflag.Usage = Usage
	pflag.Parse()

	// read target from go:generate
	if gopkg, ok := os.LookupEnv("GOPACKAGE"); *pkg == "" && ok {
		pkg = &gopkg
	}

	if *base == "" || *pkg == "" || len(*types) == 0 {
		Usage()
		os.Exit(2)
	}

	var dynamicTypes []dynamicType
	for _, v := range *types {
		split := strings.SplitN(v, ",", 3)
		dt := dynamicType{split[0], split[1], split[2]}
		dynamicTypes = append(dynamicTypes, dt)
	}

	var buf bytes.Buffer
	if err := generate(&buf, *pkg, *function, *base, dynamicTypes...); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	generated := strings.TrimSpace(buf.String()) + "\n"

	var out io.Writer = os.Stdout

	// read target from go:generate
	if gofile, ok := os.LookupEnv("GOFILE"); *target == "" && ok {
		gofile = strings.TrimSuffix(gofile, ".go") + "_decorators.go"
		target = &gofile
	}

	var name string
	if target != nil {
		name = *target
		if !strings.HasSuffix(name, ".go") {
			name += ".go"
		}

		dst, err := os.Create(name)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		defer dst.Close()
		out = dst
	}

	formatted, err := format.Source([]byte(generated))
	if err != nil {
		formatted = []byte(generated)
	}

	formatted, err = imports.Process(name, formatted, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(3)
	}

	if _, err := out.Write(formatted); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}
