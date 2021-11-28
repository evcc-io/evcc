package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/evcc-io/evcc/util/templates"
)

const basePath = "../../../templates/docs"

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
		usages := tmpl.Usages()

		fmt.Println(tmpl.Template)

		if len(usages) == 0 {
			err := writeTemplate(class, tmpl, "")
			if err != nil {
				return err
			}
		}

		// render all usages
		for _, usage := range usages {
			err := writeTemplate(class, tmpl, usage)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func writeTemplate(class string, tmpl templates.Template, usage string) error {
	values := tmpl.Defaults(true)

	if usage != "" {
		values["usage"] = usage
	}

	modbusChoices := tmpl.ModbusChoices()

	for _, modbusChoice := range modbusChoices {
		switch modbusChoice {
		case templates.ModbusChoiceRS485:
			values[templates.ModbusRS485Serial] = true
			values[templates.ModbusRS485TCPIP] = true
		case templates.ModbusChoiceTCPIP:
			values[templates.ModbusTCPIP] = true
		}
	}
	b, err := tmpl.RenderProxyWithValues(values, true)

	if err != nil {
		println(string(b))
		return err
	}

	filename := fmt.Sprintf("%s/%s/%s.yaml", basePath, class, tmpl.Template)
	if usage != "" {
		filename = fmt.Sprintf("%s/%s/%s-%s.yaml", basePath, class, tmpl.Template, usage)
	}
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
