package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/evcc-io/evcc/util/templates"
)

const basePath = "../../../templates/docs"

//go:generate go run generate.go

func main() {
	for _, class := range []string{templates.Meter, templates.Charger, templates.Vehicle} {
		path := fmt.Sprintf("%s/%s", basePath, class)
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0755); err != nil {
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
}

func generateClass(class string) error {
	for _, tmpl := range templates.ByClass(class) {
		if err := tmpl.Validate(); err != nil {
			return err
		}

		for index, product := range tmpl.Products {
			fmt.Println(tmpl.Template + ": " + tmpl.ProductTitle(product))

			err := writeTemplate(class, index, product, tmpl)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func writeTemplate(class string, index int, product templates.Product, tmpl templates.Template) error {
	values := tmpl.Defaults(templates.TemplateRenderModeDocs)

	b, err := tmpl.RenderDocumentation(product, values, "de")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s/%s_%d.yaml", basePath, class, tmpl.Template, index)
	if err := os.WriteFile(filename, b, 0644); err != nil {
		return err
	}
	return nil
}

func clearDir(dir string) error {
	names, err := ioutil.ReadDir(dir)
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
