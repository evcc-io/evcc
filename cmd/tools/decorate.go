package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"slices"
	"strings"
	"text/template"

	combinations "github.com/mxschmitt/golang-combinations"
	"github.com/spf13/pflag"
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

func generate(out io.Writer, packageName, functionName, baseType string, dynamicTypes ...dynamicType) error {
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
			return slices.Contains(combo, typ)
		},
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

		openingBrace := strings.Index(dt.signature, "(")
		closingBrace := strings.Index(dt.signature, ")")
		paramsStr := dt.signature[openingBrace+1 : closingBrace]

		paramsStr = strings.TrimSpace(paramsStr)

		var params []string
		if len(paramsStr) > 0 {
			params = strings.Split(paramsStr, ",")
		}

		returnValuesStr := dt.signature[closingBrace+1:]

		types[dt.typ] = typeStruct{
			Type:        dt.typ,
			ShortType:   parts[1],
			VarName:     strings.ToLower(parts[1][:1]) + parts[1][1:],
			Signature:   dt.signature,
			Function:    dt.function,
			Params:      params,
			ReturnTypes: returnValuesStr,
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
		Combinations: combinations.All(combos),
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

	if target != nil {
		name := *target
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

	if _, err := out.Write(formatted); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}
