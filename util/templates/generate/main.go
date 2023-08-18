package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
)

const (
	language    = "de"
	docsPath    = "../../../templates/docs"
	websitePath = "../../../templates/evcc.io"
)

//go:generate go run main.go

func main() {
	for _, class := range []templates.Class{templates.Meter, templates.Charger, templates.Vehicle} {
		path := fmt.Sprintf("%s/%s", docsPath, class)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				panic(err)
			}
		}
		if err := clearDir(path); err != nil {
			fmt.Printf("Could not clear directory for %s: %s\n", class, err)
		}

		if err := generateClass(class); err != nil {
			panic(err)
		}
	}

	if err := generateBrandJSON(); err != nil {
		panic(err)
	}
}

func generateClass(class templates.Class) error {
	for _, tmpl := range templates.ByClass(class) {
		if err := tmpl.Validate(); err != nil {
			return err
		}

		for index, product := range tmpl.Products {
			fmt.Println(tmpl.Template + ": " + product.Title(language))

			b, err := tmpl.RenderDocumentation(product, "de")
			if err != nil {
				return err
			}

			filename := fmt.Sprintf("%s/%s/%s_%d.yaml", docsPath, strings.ToLower(class.String()), tmpl.Template, index)
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
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
	return slices.Compact(keys)
}

func generateBrandJSON() error {
	chargers := make([]string, 0)
	smartPlugs := make([]string, 0)
	for _, tmpl := range templates.ByClass(templates.Charger) {
		for _, product := range tmpl.Products {
			if product.Brand != "" {
				if tmpl.Group == "switchsockets" {
					smartPlugs = append(smartPlugs, product.Brand)
				} else {
					chargers = append(chargers, product.Brand)
				}
			}
		}
	}

	vehicles := make([]string, 0)
	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		for _, product := range tmpl.Products {
			if product.Brand != "" {
				vehicles = append(vehicles, product.Brand)
			}
		}
	}

	meters := make([]string, 0)
	pvBattery := make([]string, 0)
	for _, tmpl := range templates.ByClass(templates.Meter) {
		for i := range tmpl.Params {
			if tmpl.Params[i].Name == "usage" {
				for j := range tmpl.Params[i].Choice {
					usage := tmpl.Params[i].Choice[j]
					for _, product := range tmpl.Products {
						if product.Brand != "" {
							switch usage {
							case "grid", "charge":
								meters = append(meters, product.Brand)
							case "pv", "battery":
								pvBattery = append(pvBattery, product.Brand)
							}
						}
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
