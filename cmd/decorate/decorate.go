package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/api"
	combinations "github.com/mxschmitt/golang-combinations"
	"github.com/samber/lo"
	"github.com/spf13/pflag"
	"golang.org/x/tools/imports"
)

//go:generate go tool decorate
//evcc:function decorateTest
//evcc:basetype api.Charger
//evcc:type api.MeterEnergy,TotalEnergy,func() (float64, error)
//evcc:type api.PhaseSwitcher,Phases1p3p,func(int) error
//evcc:type api.PhaseGetter,GetPhases,func() (int, error)

//go:embed decorate.tpl
var srcTmpl string

//go:embed header.tpl
var header string

type function struct {
	function, signature string
}

type dynamicType struct {
	typ       string
	functions []function
}

type funcStruct struct {
	Signature, Function, VarName, ReturnTypes string
	Params                                    []string
}

type typeStruct struct {
	Type, ShortType string
	Functions       []funcStruct
}

var a struct {
	api.Meter
	api.MeterEnergy
	api.PhaseCurrents
	api.PhaseVoltages
	api.PhasePowers
	api.MaxACPowerGetter

	api.PhaseSwitcher
	api.PhaseGetter

	api.Battery
	api.BatteryCapacity
	api.SocLimiter // vehicles only
	api.BatteryController
	api.BatterySocLimiter
	api.BatteryPowerLimiter

	api.CurrentController
	api.CurrentGetter
}

func typ(i any) string {
	return reflect.TypeOf(i).Elem().String()
}

var dependents = map[string][]string{
	typ(&a.Meter):             {typ(&a.MeterEnergy), typ(&a.PhaseCurrents), typ(&a.PhaseVoltages), typ(&a.MaxACPowerGetter)},
	typ(&a.PhaseCurrents):     {typ(&a.PhasePowers)}, // phase powers are only used to determine currents sign
	typ(&a.PhaseSwitcher):     {typ(&a.PhaseGetter)},
	typ(&a.Battery):           {typ(&a.BatteryCapacity), typ(&a.SocLimiter), typ(&a.BatteryController), typ(&a.BatterySocLimiter), typ(&a.BatteryPowerLimiter)},
	typ(&a.CurrentController): {typ(&a.CurrentGetter)},
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

func generate(out io.Writer, functionName, baseType string, dynamicTypes ...dynamicType) error {
	types := make(map[string]typeStruct, len(dynamicTypes))
	combos := make([]string, 0)

	tmpl, err := template.New("gen").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		// contains checks if slice contains string
		"contains": slices.Contains[[]string, string],
		// ordered returns a slice of funcStruct ordered by dynamicType
		"ordered": func() []funcStruct {
			ordered := make([]funcStruct, 0)
			for _, dt := range dynamicTypes {
				for _, fs := range types[dt.typ].Functions {
					ordered = append(ordered, fs)
				}
			}

			return ordered
		},
		"requiredType": func(c []string, typ string) bool {
			for master, details := range dependents {
				// exclude combinations where ...
				// - master is part of the decorators
				// - master is not part of the currently evaluated combination
				// - details are part of the currently evaluated combination
				if slices.Contains(combos, master) && !slices.Contains(c, master) && slices.Contains(details, typ) {
					return false
				}
			}
			return true
		},
		"empty": func() []string {
			return nil
		},
	}).Parse(srcTmpl)
	if err != nil {
		return err
	}

	for _, dt := range dynamicTypes {
		parts := strings.SplitN(dt.typ, ".", 2)
		lastPart := parts[len(parts)-1]

		var funcs []funcStruct

		for i, fun := range dt.functions {
			function := fun.function
			signature := fun.signature

			openingBrace := strings.Index(signature, "(")
			closingBrace := strings.Index(signature, ")")
			paramsStr := signature[openingBrace+1 : closingBrace]
			returns := signature[closingBrace+1:]

			var params []string
			if paramsStr = strings.TrimSpace(paramsStr); len(paramsStr) > 0 {
				params = strings.Split(paramsStr, ",")
			}

			varName := strings.ToLower(lastPart[:1]) + lastPart[1:]
			if len(dt.functions) > 1 {
				varName += strconv.Itoa(i)
			}

			funcs = append(funcs, funcStruct{
				VarName:     varName,
				Signature:   signature,
				Function:    function,
				Params:      params,
				ReturnTypes: returns,
			})
		}

		types[dt.typ] = typeStruct{
			Type:      dt.typ,
			ShortType: lastPart,
			Functions: funcs,
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
			// prune combinations where ...
			// - master is part of the decorators
			// - master is not part of the currently evaluated combination
			// - details are part of the currently evaluated combination
			// ... and remove details from the combination
			if slices.Contains(combos, master) && !slices.Contains(c, master) && hasIntersection(c, details) {
				c = lo.Without(c, details...)

				if len(c) == 0 {
					continue COMBO
				}
			}
		}

		// prune duplicates
		for _, v := range validCombos {
			if slices.Equal(v, c) {
				continue COMBO
			}
		}

		validCombos = append(validCombos, c)
	}

	vars := struct {
		Function            string
		BaseType, ShortBase string
		ReturnType          string
		Types               map[string]typeStruct
		Combinations        [][]string
	}{
		Function:     functionName,
		BaseType:     baseType,
		ShortBase:    shortBase,
		ReturnType:   returnType,
		Types:        types,
		Combinations: validCombos,
	}

	return tmpl.Execute(out, vars)
}

type decorationSet struct {
	function, base, ret string
	types               []string
}

var (
	target   = pflag.StringP("out", "o", "", "output file")
	pkg      = pflag.StringP("package", "p", "", "package name")
	funcname = pflag.StringP("function", "f", "", "function name")
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

func parseFile(file string) ([]decorationSet, error) {
	var res []decorationSet

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var current decorationSet

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if s, ok := strings.CutPrefix(line, "//evcc:"); ok {
			segs := strings.SplitN(s, " ", 2)
			if len(segs) != 2 {
				panic("invalid segments: " + s)
			}

			switch segs[0] {
			case "function":
				// must be first
				if current.function != "" {
					res = append(res, current)
					current = decorationSet{}
				}
				current.function = segs[1]
			case "basetype":
				current.base = segs[1]
			case "returntype":
				current.ret = segs[1]
			case "type":
				current.types = append(current.types, segs[1])
			default:
				panic("invalid directive //evcc:" + segs[0])
			}
		}
	}

	if current.function != "" {
		res = append(res, current)
	}

	return res, scanner.Err()
}

func splitTopLevel(s string) []string {
	var res []string
	brackets := 0
	start := 0

	for i, r := range s {
		switch r {
		case '(':
			brackets++
		case ')':
			brackets--
		case ',':
			if brackets == 0 {
				res = append(res, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	res = append(res, strings.TrimSpace(s[start:]))
	return res
}

func parseFunctions(iface string) []function {
	parts := splitTopLevel(iface)

	var res []function
	for i := 0; i+1 < len(parts); i += 2 {
		res = append(res, function{
			function:  parts[i],
			signature: parts[i+1],
		})
	}
	return res
}

func main() {
	pflag.Usage = Usage
	pflag.Parse()

	// read target from go:generate
	gofile, ok := os.LookupEnv("GOFILE")
	if *target == "" && ok {
		gofile := strings.TrimSuffix(gofile, ".go") + "_decorators.go"
		target = &gofile
	}

	// read target from go:generate
	if gopkg, ok := os.LookupEnv("GOPACKAGE"); *pkg == "" && ok {
		pkg = &gopkg
	}

	sets := []decorationSet{{*funcname, *base, *ret, *types}}

	if *funcname == "" {
		all, err := parseFile(gofile)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		sets = all
	}

	if *pkg == "" || len(sets) == 0 || sets[0].base == "" || len(sets[0].types) == 0 {
		Usage()
		os.Exit(2)
	}

	var out io.Writer = os.Stdout

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

	generated := new(bytes.Buffer)
	fmt.Fprintln(generated, strings.ReplaceAll(header, "{{.Package}}", *pkg))

	for _, set := range sets {
		var dynamicTypes []dynamicType

		for _, v := range set.types {
			split := strings.SplitN(v, ",", 2) // iface,...
			dynamicTypes = append(dynamicTypes, dynamicType{split[0], parseFunctions(split[1])})
		}

		var buf bytes.Buffer
		if err := generate(&buf, set.function, set.base, dynamicTypes...); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}

		fmt.Fprintln(generated, buf.String())
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

	if _, err := out.Write(formatted); err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}
