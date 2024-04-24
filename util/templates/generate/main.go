package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/samber/lo"
)

const (
	docsPath    = "../../../templates/docs"
	websitePath = "../../../templates/evcc.io"
)

//go:generate go run main.go

func main() {
	for _, lang := range []string{"de", "en"} {
		if err := generateDocs(lang); err != nil {
			panic(err)
		}
	}

	if err := generateBrandJSON(); err != nil {
		panic(err)
	}
}

func generateDocs(lang string) error {
	for _, class := range templates.ClassValues() {
		path := fmt.Sprintf("%s/%s/%s", docsPath, lang, strings.ToLower(class.String()))
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		}
		if err := clearDir(path); err != nil {
			fmt.Printf("Could not clear directory for %s: %s\n", class, err)
		}

		if err := generateClass(class, lang); err != nil {
			return err
		}
	}

	return nil
}

func generateClass(class templates.Class, lang string) error {
	tmpls := lo.Filter(templates.ByClass(class), func(t templates.Template, _ int) bool {
		return !t.Deprecated
	})

	for _, tmpl := range tmpls {
		if err := tmpl.Validate(); err != nil {
			return err
		}

		for index, product := range tmpl.Products {
			fmt.Println(tmpl.Template + ": " + product.Title(lang))

			b, err := tmpl.RenderDocumentation(product, lang)
			if err != nil {
				return err
			}

			filename := fmt.Sprintf("%s/%s/%s/%s_%d.yaml", docsPath, lang, strings.ToLower(class.String()), tmpl.Template, index)
			if err := os.WriteFile(filename, b, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func clearDir(dir string) error {
	names, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range names {
		if err := os.RemoveAll(path.Join([]string{dir, entry.Name()}...)); err != nil {
			return err
		}
	}

	return nil
}

func sorted(keys []string) []string {
	slices.SortFunc(keys, func(i, j string) int {
		return strings.Compare(strings.ToLower(i), strings.ToLower(j))
	})
	return slices.Compact(keys)
}

func generateBrandJSON() error {
	var chargers, smartPlugs []string
	for _, tmpl := range templates.ByClass(templates.Charger) {
		for _, product := range tmpl.Products {
			if product.Brand == "" {
				continue
			}

			if tmpl.Group == "switchsockets" {
				smartPlugs = append(smartPlugs, product.Brand)
			} else {
				chargers = append(chargers, product.Brand)
			}
		}
	}

	var vehicles []string
	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		for _, product := range tmpl.Products {
			if product.Brand != "" {
				vehicles = append(vehicles, product.Brand)
			}
		}
	}

	var meters, pvBattery []string
	for _, tmpl := range templates.ByClass(templates.Meter) {
		for i := range tmpl.Params {
			if tmpl.Params[i].Name != templates.ParamUsage {
				continue
			}

			for j := range tmpl.Params[i].Choice {
				usage, _ := templates.UsageString(tmpl.Params[i].Choice[j])
				for _, product := range tmpl.Products {
					if product.Brand == "" {
						continue
					}

					switch usage {
					case templates.UsageGrid, templates.UsageCharge, templates.UsageAux:
						meters = append(meters, product.Brand)
					case templates.UsagePV, templates.UsageBattery:
						pvBattery = append(pvBattery, product.Brand)
					}
				}
			}
		}
	}

	brands := struct {
		Chargers, SmartPlugs, Meters, PVBattery, Vehicles []string
	}{
		Chargers:   sorted(chargers),
		SmartPlugs: sorted(smartPlugs),
		Meters:     sorted(meters),
		PVBattery:  sorted(pvBattery),
		Vehicles:   sorted(vehicles),
	}

	file, err := json.MarshalIndent(brands, "", " ")
	if err == nil {
		err = os.WriteFile(websitePath+"/brands.json", file, 0o644)
	}

	return err
}
