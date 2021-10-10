package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/templates/builtin"
	"github.com/evcc-io/evcc/vehicle"
)

const basePath = "../../docs"

//go:generate go run generate.go

func main() {
	for typ, m := range vehicle.Registry {
		fmt.Println("---")
		fmt.Println(typ)

		if m.Config == nil {
			continue
		}

		meta := builtin.Annotate(m.Config)

		b, err := render(typ, meta)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))

		filename := fmt.Sprintf("%s/%s/%s.yaml", basePath, "vehicle", typ)
		if err := os.WriteFile(filename, b, 0644); err != nil {
			panic(err)
		}
	}
}

//go:embed vehicle.tpl
var tpl string

func render(typ string, params []builtin.FieldMetadata) ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(tpl)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"Type":   typ,
		"Params": params,
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, values)

	return bytes.TrimSpace(out.Bytes()), err
}
