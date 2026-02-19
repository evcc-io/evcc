package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"io"
	"maps"
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
//evcc:types api.MeterEnergy,api.PhaseSwitcher,api.PhaseGetter

//go:embed decorate.tpl
var srcTmpl string

//go:embed header.tpl
var header string

type funcStruct struct {
	Signature, Function, VarName, ReturnTypes string
	Params                                    []string
}

type typeStruct struct {
	Type, ShortType string
	Functions       []funcStruct
}

var interfaces = make(map[string]reflect.Type)
var dependents = make(map[string][]string)

func init() {
	reflectTypes := map[reflect.Type][]reflect.Type{
		reflect.TypeFor[api.Meter]():             {reflect.TypeFor[api.MeterEnergy](), reflect.TypeFor[api.PhaseCurrents](), reflect.TypeFor[api.PhaseVoltages](), reflect.TypeFor[api.MaxACPowerGetter]()},
		reflect.TypeFor[api.PhaseCurrents]():     {reflect.TypeFor[api.PhasePowers]()}, // phase powers are only used to determine currents sign
		reflect.TypeFor[api.PhaseSwitcher]():     {reflect.TypeFor[api.PhaseGetter]()},
		reflect.TypeFor[api.Battery]():           {reflect.TypeFor[api.BatteryCapacity](), reflect.TypeFor[api.SocLimiter](), reflect.TypeFor[api.BatteryController](), reflect.TypeFor[api.BatterySocLimiter](), reflect.TypeFor[api.BatteryPowerLimiter]()},
		reflect.TypeFor[api.ChargeState]():       {reflect.TypeFor[api.ChargeController](), reflect.TypeFor[api.CurrentController]()},
		reflect.TypeFor[api.CurrentController](): {reflect.TypeFor[api.CurrentGetter]()},
	}

	for typ, types := range reflectTypes {
		interfaces[typ.String()] = typ
		for _, t := range types {
			interfaces[t.String()] = t
		}

		dependents[typ.String()] = lo.Map(types, func(typ reflect.Type, _ int) string {
			return typ.String()
		})
	}

	for _, typ := range []reflect.Type{
		reflect.TypeFor[api.Curtailer](),
		reflect.TypeFor[api.Resurrector](),
		reflect.TypeFor[api.VehicleOdometer](),
		reflect.TypeFor[api.VehicleRange](),
		reflect.TypeFor[api.VehicleClimater](),
		reflect.TypeFor[api.VehicleFinishTimer](),
		reflect.TypeFor[api.Identifier](),
		reflect.TypeFor[api.ChargerEx](),
		reflect.TypeFor[api.ChargeRater](),
		reflect.TypeFor[api.StatusReasoner](),
	} {
		interfaces[typ.String()] = typ
	}
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

func getCombinations(combos []string) [][]string {
	validCombos := make([][]string, 0)
	sortedDependents := slices.Sorted(maps.Keys(dependents))

COMBO:
	for _, c := range combinations.All(combos) {
		// order the cases for generation
		for _, master := range sortedDependents {
			details := dependents[master]
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

	return validCombos
}

func getTemplate(dtypes []reflect.Type, types map[string]typeStruct, combos []string) *template.Template {
	tmpl, err := template.New("gen").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		// contains checks if slice contains string
		"contains": slices.Contains[[]string, string],
		// ordered returns a slice of funcStruct ordered by dynamicType
		"ordered": func() []funcStruct {
			ordered := make([]funcStruct, 0)
			for _, t := range dtypes {
				for _, f := range types[getTypeImport(t)].Functions {
					ordered = append(ordered, f)
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
		fmt.Printf("invalid template: %s", err)
		os.Exit(2)
	}

	return tmpl
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

func generate(out io.Writer, functionName, baseType string, dtypes []reflect.Type) error {
	var combos []string
	types := make(map[string]typeStruct)

	for _, t := range dtypes {
		lastPart := t.Name()

		var funcs []funcStruct

		for i := 0; i < t.NumMethod(); i++ {
			m := t.Method(i)

			varName := strings.ToLower(lastPart[:1]) + lastPart[1:]
			if t.NumMethod() > 1 {
				varName += strconv.Itoa(i)
			}

			var params []string
			for input := range m.Type.Ins() {
				params = append(params, getTypeImport(input))
			}

			var returns []string
			for output := range m.Type.Outs() {
				returns = append(returns, getTypeImport(output))
			}

			funcs = append(funcs, funcStruct{
				VarName:     varName,
				Signature:   fmt.Sprintf("func(%s) (%s)", strings.Join(params, ", "), strings.Join(returns, ", ")),
				Function:    m.Name,
				Params:      params,
				ReturnTypes: fmt.Sprintf("(%s)", strings.Join(returns, ",")),
			})
		}

		types[getTypeImport(t)] = typeStruct{
			Type:      t.Name(),
			ShortType: lastPart,
			Functions: funcs,
		}

		combos = append(combos, getTypeImport(t))
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
		Combinations: getCombinations(combos),
	}

	return getTemplate(dtypes, types, combos).Execute(out, vars)
}

type decorationSet struct {
	function, base, ret, types string
}

var (
	target   = pflag.StringP("out", "o", "", "output file")
	pkg      = pflag.StringP("package", "p", "", "package name")
	funcname = pflag.StringP("function", "f", "", "function name")
	base     = pflag.StringP("base", "b", "", "base type")
	ret      = pflag.StringP("return", "r", "", "return type")
	types    = pflag.StringP("type", "t", "", "comma-separated list of type definitions")
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
			case "types":
				current.types = segs[1]
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
		var types []reflect.Type

		for t := range strings.SplitSeq(set.types, ",") {
			typ, ok := interfaces[t]

			if !ok {
				fmt.Printf("don't know interface %s\n", t)
				os.Exit(2)
			}

			types = append(types, typ)
		}

		var buf bytes.Buffer
		if err := generate(&buf, set.function, set.base, types); err != nil {
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
