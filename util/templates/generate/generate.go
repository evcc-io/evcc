package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"golang.org/x/exp/maps"
)

const (
	docsPath    = "../../../templates/docs"
	websitePath = "../../../templates/evcc.io"
)

//go:generate go run generate.go

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
			fmt.Println(tmpl.Template + ": " + product.Title(tmpl.Lang))

			if err := writeTemplate(class, index, product, tmpl); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeTemplate(class templates.Class, index int, product templates.Product, tmpl templates.Template) error {
	values := tmpl.Defaults(templates.TemplateRenderModeDocs)

	b, err := tmpl.RenderDocumentation(product, values, "de")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s/%s_%d.yaml", docsPath, class, tmpl.Template, index)
	if err := os.WriteFile(filename, b, 0o644); err != nil {
		return err
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

func sortedKeys(data map[string]bool) []string {
	keys := maps.Keys(data)
	sort.Slice(keys, func(i, j int) bool { return strings.ToLower(keys[i]) < strings.ToLower(keys[j]) })
	return keys
}

func generateBrandJSON() error {
	chargers := make(map[string]bool)
	smartPlugs := make(map[string]bool)
	for _, tmpl := range templates.ByClass(templates.Charger) {
		for _, product := range tmpl.Products {
			if product.Brand != "" {
				if tmpl.Group == "switchsockets" {
					smartPlugs[product.Brand] = true
				} else {
					chargers[product.Brand] = true
				}
			}
		}
	}

	vehicles := make(map[string]bool)
	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		for _, product := range tmpl.Products {
			if product.Brand != "" {
				vehicles[product.Brand] = true
			}
		}
	}

	meters := make(map[string]bool)
	pvBattery := make(map[string]bool)
	for _, tmpl := range templates.ByClass(templates.Meter) {
		for i := range tmpl.Params {
			if tmpl.Params[i].Name == "usage" {
				for j := range tmpl.Params[i].Choice {
					usage := tmpl.Params[i].Choice[j]
					for _, product := range tmpl.Products {
						if product.Brand != "" {
							switch usage {
							case "grid", "charge":
								meters[product.Brand] = true
							case "pv", "battery":
								pvBattery[product.Brand] = true
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
		Chargers:   sortedKeys(chargers),
		SmartPlugs: sortedKeys(smartPlugs),
		Meters:     sortedKeys(meters),
		PVBattery:  sortedKeys(pvBattery),
		Vehicles:   sortedKeys(vehicles),
	}

	file, err := json.MarshalIndent(brands, "", " ")
	if err == nil {
		err = os.WriteFile(websitePath+"/brands.json", file, 0o644)
	}

	return err
}
