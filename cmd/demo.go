package cmd

import (
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const demoYaml = `
log: info

meters:
- name: grid
  type: default
  power:
    type: js
    script: "1200"
- name: pv
  type: default
  power:
    type: js
    script: "6400"
- name: battery
  type: default
  power:
    type: js
    script: "200"
  soc:
    type: js
    script: "30"

chargers:
- name: demo
  type: default
  enable:
    type: js
    script: "true"
  enabled:
    type: js
    script: "true"
  status:
    type: js
    script: "'B'"
  maxcurrent:
    type: js
    script: "6"

vehicles:
- name: demo
  title: e-Golf
  type: default
  charge:
    type: js
    script: "63"

site:
  title: Demo
  meters:
    grid: grid
    pv: pv
    battery: battery

loadpoints:
- title: Carport
  charger: demo
  vehicle: demo
`

func demoConfig() (conf config) {
	demo := map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(demoYaml), &demo); err != nil {
		log.FATAL.Fatalf("failed decoding demo config: %+v", err)
	}

	for k, v := range demo {
		viper.Set(k, v)
	}

	if err := viper.UnmarshalExact(&conf); err != nil {
		log.FATAL.Fatalf("failed loading demo config: %v", err)
	}

	return conf
}
