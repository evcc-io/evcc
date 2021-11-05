package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/util"
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
		modbusChoices := tmpl.ModbusChoices()

		fmt.Println(tmpl.Type)

		if len(usages) == 0 {
			values := make(map[string]interface{})
			for _, modbusChoice := range modbusChoices {
				switch modbusChoice {
				case "rs485":
					values["modbusrs485serial"] = true
					values["modbusrs485tcpip"] = true
				case "tcpip":
					values["modbustcpip"] = true
				}
			}
			examples := tmpl.Examples()
			if err := util.DecodeOther(examples, &values); err != nil {
				return err
			}
			b, err := tmpl.RenderResult(values)
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
			values := map[string]interface{}{
				"usage": usage,
			}
			for _, modbusChoice := range modbusChoices {
				switch modbusChoice {
				case "rs485":
					values["modbusrs485serial"] = true
					values["modbusrs485tcpip"] = true
				case "tcpip":
					values["modbustcpip"] = true
				}
			}
			examples := tmpl.Examples()
			if err := util.DecodeOther(examples, &values); err != nil {
				return err
			}
			b, err := tmpl.RenderResult(values)

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
