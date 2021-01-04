package cmd

import (
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const demoYaml = `
log: info

javascript:
- vm: shared
  script: |
    state = {
      residualpower: 200,
      pvpower: -3000,
      batterypower: 200,
      gridpower: -2000,
      chargepower: 0,
      maxcurrent: 0,
      battery: 63, // car
    };
    function get() {
      console.log("state:", JSON.stringify(state));
    }
    function set() {
      console.log(param+":", val);
      console.log("state:", JSON.stringify(state));
    }

meters:
- name: grid
  type: default
  power:
    type: js
    vm: shared
    script: state.gridpower = state.pvpower + state.chargepower + state.residualpower + state.batterypower;

- name: pv
  type: default
  power:
    type: js
    vm: shared
    script: state.pvpower += 100*Math.random();

- name: battery
  type: default
  power:
    type: js
    vm: shared
    script: state.batterypower;
  soc:
    type: js
    vm: shared
    script: "30"
- name: charge
  type: default
  power:
    type: js
    vm: shared
    script: state.chargepower;
  soc:
    type: js
    vm: shared
    script: "30"

chargers:
- name: demo
  type: default
  enable:
    type: js
    vm: shared
    script: |
      set();
      state.enabled = val;
      if (state.enabled) state.chargepower = state.maxcurrent * 230; else state.chargepower = 0;
  enabled:
    type: js
    vm: shared
    script: |
      state.enabled;
  status:
    type: js
    vm: shared
    script: |
      if (state.enabled) "C"; else "B";
  maxcurrent:
    type: js
    vm: shared
    script: |
      set();
      state.maxcurrent = val;
      if (state.enabled) state.chargepower = state.maxcurrent * 230;

vehicles:
- name: demo
  title: e-Golf
  type: default
  charge:
    type: js
    vm: shared
    script: |
      if (state.chargepower > 0) state.battery++; else state.battery--;
      if (state.battery < 15) state.battery = 15;
      if (state.battery > 100) state.battery = 100;
      state.battery;
  cache: 1s

site:
  title: Demo
  meters:
    grid: grid
    pv: pv
    battery: battery

loadpoints:
- title: Carport
  charger: demo
  meters:
    charge: charge
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
