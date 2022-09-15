package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	fullURLFile = "https://raw.githubusercontent.com/joscha82/wattpilot/da08c3fb387b06497e007bef1ff88f0112a080ea/src/wattpilot/ressources/wattpilot.yaml"
	output      = "wattpilot_mapping_gen.go"
)

func downloadWattpilotYaml() ([]byte, error) {

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	// Put content on file
	resp, err := client.Get(fullURLFile)
	if err != nil {
		print(err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func main() {
	s, _ := downloadWattpilotYaml()
	a := make(map[string]interface{})
	if err := yaml.Unmarshal([]byte(s), &a); err != nil {
		print(err)
		return
	}
	propertyMap := make(map[string]string)
	for _, v := range a["properties"].([]interface{}) {
		key := ""
		alias := ""
		data := v.(map[interface{}]interface{})
		for x, y := range data {

			switch x.(string) {
			case "key":
				key = y.(string)
			case "alias":
				alias = y.(string)
			}
		}
		if key != "" && alias != "" {

			propertyMap[alias] = key
		}
	}
	f, _ := os.Create(output)
	defer f.Close()

	w := bufio.NewWriter(f)
	w.WriteString("package wattpilot\nvar propertyMap = map[string]string {\n")
	for i, s := range propertyMap {
		w.WriteString(fmt.Sprintf("\"%s\": \"%s\",\n", i, s))
	}

	w.WriteString("}\n")

	w.Flush()

}
