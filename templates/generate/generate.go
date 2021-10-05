package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/evcc-io/evcc/templates"
)

const basePath = "../docs"

func main() {
	for _, class := range []string{templates.Meter, templates.Charger, templates.Vehicle} {
		if err := clearDir(fmt.Sprintf("%s/%s", basePath, class)); err != nil {
			panic(err)
		}

		if err := generateClass(class); err != nil {
			panic(err)
		}
	}
}

func generateClass(class string) error {
	for _, tmpl := range templates.ByClass(class) {
		usages := tmpl.Usages()

		fmt.Println(tmpl.Type)

		if len(usages) == 0 {
			b, err := tmpl.RenderResult(nil)
			if err != nil {
				println(string(b))
				return err
			}

			filename := fmt.Sprintf("%s/%s/%s.yaml", basePath, class, tmpl.Type)
			if err := os.WriteFile(filename, b, 0644); err != nil {
				return err
			}
		}

		// render all usages
		for _, usage := range usages {
			b, err := tmpl.RenderResult(map[string]interface{}{
				"usage": usage,
			})

			if err != nil {
				println(string(b))
				return err
			}

			filename := fmt.Sprintf("%s/%s/%s-%s.yaml", basePath, class, tmpl.Type, usage)
			if err := os.WriteFile(filename, b, 0644); err != nil {
				return err
			}
		}
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
