<template>
	<div class="root">
		<div class="container px-4">
			<TopHeader title="Configuration Editor 🧪" />
			<div class="wrapper">
				<h2 class="my-4">evcc.yaml</h2>
				<button class="btn btn-primary" @click="handleSave">Save</button>
				<vue-monaco-editor
					v-model:value="yaml"
					class="editor"
					:theme="theme"
					language="yaml"
					height="calc(100vh - 200px)"
					:options="editorOptions"
					@mount="handleMount"
				>
					loading editor ...
				</vue-monaco-editor>
			</div>
		</div>
	</div>
</template>

<script>
import TopHeader from "../components/TopHeader.vue";
import "@h2d2/shopicons/es/bold/arrowback";
import store from "../store";
import collector from "../mixins/collector";
import api from "../api";

export default {
	name: "ConfigEditor",
	components: { TopHeader },
	mixins: [collector],
	data() {
		return {
			theme: "vs",
			yaml: `network:
  port: 7070

log: debug

interval: 3s

javascript:
  - vm: shared
    script: |
      state = {
        residualpower: 500,
        pvpower: 5000,
        batterypower: -750,
        batterySoc: 55,
        gridpower: -1000,
        loadpoints: [
          { enabled: true, vehicleSoc: 62, maxcurrent: 6, phases: 1, chargepower: 0 },
          { enabled: false, vehicleSoc: 22, maxcurrent: 0, phases: 3, chargepower: 0 }
        ]
      };
      function logState() {
        console.log("state:", JSON.stringify(state));
      }

meters:
  - name: grid
    type: custom
    power:
      source: js
      vm: shared
      script: |
        state.gridpower = state.loadpoints[0].chargepower + state.loadpoints[1].chargepower + state.residualpower - batterypower - pvpower;
        state.gridpower;
      in:
        - name: pvpower
          type: float
          config:
            source: js
            vm: shared
            script: |
              state.pvpower = 8000+500*Math.random();
              state.pvpower
        - name: batterypower
          type: float
          config:
            source: js
            vm: shared
            script: |
              state.batterypower = state.gridpower > 0 ? 1000 * Math.random() : 0;
              state.batterypower

  - name: pv
    type: custom
    power:
      source: js
      vm: shared
      script: state.pvpower;

  - name: battery
    type: custom
    power:
      source: js
      vm: shared
      script: state.batterypower;
    soc:
      source: js
      vm: shared
      script: |
        if (state.batterypower < 0) state.batterySoc++; else state.batterySoc--;
        if (state.batterySoc < 10) state.batterySoc = 90;
        if (state.batterySoc > 90) state.batterySoc = 10;
        state.batterySoc;
    capacity: 13.4

  - name: meter_charger_1
    type: custom
    power:
      source: js
      vm: shared
      script: state.loadpoints[0].chargepower;

  - name: meter_charger_2
    type: custom
    power:
      source: js
      vm: shared
      script: state.loadpoints[1].chargepower;

chargers:
  - name: charger_1
    type: custom
    enable:
      source: js
      vm: shared
      script: |
        logState();
        var lp = state.loadpoints[0];
        lp.enabled = enable;
        enable;
      out:
        - name: enable
          type: bool
          config:
            source: js
            vm: shared
            script: |
              if (enable) lp.chargepower = lp.maxcurrent * 230 * lp.phases; else lp.chargepower = 0;
    enabled:
      source: js
      vm: shared
      script: |
        state.loadpoints[0].enabled;
    status:
      source: js
      vm: shared
      script: |
        if (state.loadpoints[0].enabled) "C"; else "B";
    maxcurrent:
      source: js
      vm: shared
      script: |
        logState();
        var lp = state.loadpoints[0];
        lp.maxcurrent = maxcurrent;
        if (lp.enabled) lp.chargepower = lp.maxcurrent * 230 * lp.phases; else lp.chargepower = 0;

  - name: charger_2
    type: custom
    enable:
      source: js
      vm: shared
      script: |
        logState();
        var lp = state.loadpoints[1];
        lp.enabled = enable;
        if (lp.enabled) lp.chargepower = lp.maxcurrent * 230 * lp.phases; else lp.chargepower = 0;
    enabled:
      source: js
      vm: shared
      script: |
        state.loadpoints[1].enabled;
    status:
      source: js
      vm: shared
      script: |
        if (state.loadpoints[1].enabled) "C"; else "B";
    maxcurrent:
      source: js
      vm: shared
      script: |
        logState();
        var lp = state.loadpoints[1];
        lp.maxcurrent = maxcurrent;
        if (lp.enabled) lp.chargepower = lp.maxcurrent * 230 * lp.phases; else lp.chargepower = 0;
    tos: true
    phases1p3p:
      source: js
      vm: shared
      script: |
        logState();
        if (phases === 1) lp.phases = 1; else lp.phases = 3;
        lp.phases;

vehicles:
  - name: vehicle_1
    title: blauer e-Golf
    type: custom
    soc:
      source: js
      vm: shared
      script: |
        var lp = state.loadpoints[0];
        if (lp.chargepower > 0) lp.vehicleSoc+=0.1; else lp.vehicleSoc-=0.1;
        if (lp.vehicleSoc < 15) lp.vehicleSoc = 80;
        if (lp.vehicleSoc > 80) lp.vehicleSoc = 15;
        lp.vehicleSoc;
    range:
      source: js
      vm: shared
      script: |
        var lp = state.loadpoints[0]
        var range = (44 * lp.vehicleSoc) / 15;
        range
    capacity: 44
  - name: vehicle_2
    title: weißes Model 3
    type: custom
    soc:
      source: js
      vm: shared
      script: |
        var lp = state.loadpoints[1];
        if (lp.chargepower > 0) lp.vehicleSoc++; else lp.vehicleSoc--;
        if (lp.vehicleSoc < 15) lp.vehicleSoc = 75;
        if (lp.vehicleSoc > 75) lp.vehicleSoc = 15;
        lp.vehicleSoc;
    range:
      source: js
      vm: shared
      script: |
        var lp = state.loadpoints[1]
        var range = (80 * lp.vehicleSoc) / 17;
        range
    status:
      source: js
      vm: shared
      script: |
        "B"
    capacity: 80
  - name: vehicle_3
    type: template
    template: offline
    title: grüner Honda e
    capacity: 8
  - name: vehicle_4
    type: template
    template: offline
    title: schwarzes VanMoof
    icon: bike
    capacity: 0.46
  - name: vehicle_5
    type: template
    template: offline
    title: Wärmepumpe
    icon: waterheater

site:
  title: Zuhause
  meters:
    grid: grid
    pv: pv
    battery: battery

loadpoints:
  - title: Carport
    charger: charger_1
    mode: pv
    phases: 1
    meter: meter_charger_1
    vehicle: vehicle_1
  - title: Garage
    charger: charger_2
    mode: "off"
    meter: meter_charger_2
    vehicle: vehicle_2

tariffs:
  currency: EUR # three letter ISO-4217 currency code (default EUR)
  grid:
    type: fixed
    price: 0.399 # EUR/kWh
  feedin:
    type: fixed
    price: 0.08 # EUR/kWh
  co2:
    type: grünstromindex
    zip: 12349
`,
		};
	},
	computed: {
		TopHeader: function () {
			const vehicleLogins = store.state.auth ? store.state.auth.vehicles : {};
			return { vehicleLogins, ...this.collectProps(TopHeader, store.state) };
		},
		editorOptions: function () {
			return {
				automaticLayout: true,
				formatOnType: true,
				formatOnPaste: true,
				minimap: { enabled: false },
				showFoldingControls: "always",
				scrollBeyondLastLine: false,
			};
		},
	},
	async mounted() {
		this.updateTheme();
		document.querySelector("html").addEventListener("themechange", this.updateTheme);
		const res = await api.get("/config/yaml");
		this.yaml = res.data?.result?.Content;
	},
	unmounted() {
		document.querySelector("html").removeEventListener("themechange", this.updateTheme);
	},
	methods: {
		updateTheme() {
			this.theme = document.querySelector("html").classList.contains("dark")
				? "vs-dark"
				: "vs";
		},
		handleMount() {
			console.log("not implemented yet");
		},
		async handleSave() {
			await api.put("/config/yaml", this.yaml);
		},
	},
	errorCaptured(err, vm, info) {
		console.log({ err, vm, info });
		return false;
	},
};
</script>
<style scoped>
.back {
	width: 22px;
	height: 22px;
	position: relative;
	top: -2px;
}
.container {
	max-width: 900px;
}
.editor :global(.monaco-editor) {
	--vscode-editor-background: var(--evcc-box) !important;
	--vscode-editorGutter-background: var(--evcc-box-border) !important
;
}
</style>
