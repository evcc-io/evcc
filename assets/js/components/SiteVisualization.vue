<template>
	<div class="site-visualization">
		<div class="d-flex justify-content-between">
			<span class="usage-label">Verbrauch {{ kw(usage) }}</span>
			<span class="surplus-label" v-if="pvPower > 0">Produktion {{ kw(pvPower) }}</span>
		</div>
		<div class="site-progress">
			<div class="site-progress-bar usage" :style="{ width: widthTotal(usage) }">
				<div class="site-progress-bar grid-usage" :style="{ width: widthUsage(gridUsage) }">
					<span class="power">{{ kw(gridUsage) }}</span>
				</div>
				<div class="site-progress-bar pv-usage" :style="{ width: widthUsage(pvUsage) }">
					<span class="power">{{ kw(pvUsage) }}</span>
				</div>
				<div
					class="site-progress-bar battery-usage"
					:style="{ width: widthUsage(batteryUsage) }"
				>
					<span class="power">{{ kw(batteryUsage) }}</span>
				</div>
			</div>
			<div
				class="site-progress-bar surplus"
				:style="{ width: widthTotal(surplus), marginLeft: surplus > 0 ? null : 0 }"
			>
				<div
					class="site-progress-bar battery-charge"
					:style="{ width: widthSurplus(batteryCharge) }"
				>
					<span class="power">{{ kw(batteryCharge) }}</span>
				</div>
				<div class="site-progress-bar pv-export" :style="{ width: widthSurplus(pvExport) }">
					<span class="power">{{ kw(pvExport) }}</span>
				</div>
			</div>
		</div>
		<div class="d-flex justify-content-between">
			<span class="grid-usage-label" v-if="gridUsage"> Netzbezug </span>
			<span class="pv-usage-label" v-if="pvUsage"> Eigenverbrauch </span>
			<span class="battery-usage-label" v-if="batteryUsage"> Akkuverbrauch </span>
			<span class="battery-charge-label" v-if="batteryCharge"> Akku laden </span>
			<span class="pv-export-label" v-if="pvExport"> Einspeisung </span>
		</div>
		<!--
		<div class="site-charger">
			<div
				class="charger1"
				:style="{
					width: widthUsage(Math.min(usage, loadpoints[0].chargePower)),
				}"
			>
				<span>{{ kw(Math.min(usage, loadpoints[0].chargePower)) }}</span>
			</div>
		</div>
		-->
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

export default {
	name: "SiteVisualization",
	props: {
		gridConfigured: Boolean,
		gridPower: Number,
		pvConfigured: Boolean,
		pvPower: Number,
		batteryConfigured: Boolean,
		batteryPower: Number,
		batterySoC: Number,
		loadpoints: Array,
	},
	mixins: [formatter],
	computed: {
		usage: function () {
			return Math.max(
				0,
				this.pvPower + this.gridPower + this.batteryUsage - this.batteryCharge
			);
		},
		gridUsage: function () {
			return Math.max(0, this.gridPower);
		},
		pvUsage: function () {
			return Math.min(this.pvPower, this.pvPower + this.gridPower - this.batteryCharge);
		},
		pvExport: function () {
			return this.pvPower - this.pvUsage - this.batteryCharge;
		},
		batteryUsage: function () {
			return Math.max(0, this.batteryPower);
		},
		batteryCharge: function () {
			return Math.min(0, this.batteryPower) * -1;
		},
		surplus: function () {
			return this.pvExport + this.batteryCharge;
		},
		total: function () {
			return this.usage + this.surplus;
		},
	},
	methods: {
		widthTotal: function (power) {
			return (100 / this.total) * power + "%";
		},
		widthSurplus: function (power) {
			return (100 / this.surplus) * power + "%";
		},
		widthUsage: function (power) {
			return (100 / this.usage) * power + "%";
		},
		kw: function (watt) {
			return (watt / 1000).toFixed(1) + " kW";
		},
	},
};
</script>
<style scoped>
.site-visualization {
}
.site-progress {
	margin: 0.5rem 0 0.3rem;
	height: 1.5rem;
	display: flex;
}
.site-progress-bar {
	text-align: center;
	transition-property: margin, width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	overflow: hidden;
	white-space: nowrap;
}
.usage {
	border-radius: 5px;
	display: flex;
}
.grid-usage {
	color: #eee;
	background-color: #18191a;
}
.pv-usage {
	background-color: #66d85a;
}
.surplus {
	border-radius: 5px;
	display: flex;
	margin-left: 10px;
}
.pv-export {
	background-color: #fbdf4b;
	color: #18191a;
}
.battery-usage,
.battery-charge {
	background-color: #ee706b;
	color: #eee;
}
.usage-label,
.pv-export-label,
.pv-usage-label,
.grid-usage-label,
.battery-usage-label,
.battery-charge-label {
	color: var(--bs-gray-dark);
	text-decoration-line: underline;
	text-decoration-skip-ink: auto;
	text-decoration-thickness: 2px;
	text-decoration-style: solid;
	overflow: hidden;
	text-overflow: ellipsis;
	display: block;
	transition-property: opacity, width;
	transition-duration: 500ms;
	transition-timing-function: linear;
}
.power {
	display: block;
	margin: 0 0.5rem;
	text-overflow: ellipsis;
	overflow: hidden;
}
.usage-label {
	text-decoration: none;
}
.pv-export-label {
	text-decoration-color: #fbdf4b;
}
.grid-usage-label {
	text-decoration-color: #18191a;
	font-size: 0.875em;
}
.battery-usage-label {
	text-decoration-color: #ee706b;
	font-size: 0.875em;
}
.battery-charge-label {
	text-decoration-color: #ee706b;
	font-size: 0.875em;
}
.pv-usage-label {
	text-decoration-color: #66d85a;
	font-size: 0.875em;
}

.site-charger {
	margin: 1rem 0;
	padding: 1rem 0;
	display: flex;
	justify-content: flex-end;
	visibility: hidden;
}
.charger1 {
	position: relative;
	background-color: #18191a;
	text-align: center;
	height: 1px;
}
.charger1::before,
.charger1::after {
	content: "";
	border-left: 1px solid #18191a;
	position: absolute;
	top: -5px;
	height: 11px;
}
.charger1::before {
	left: 0;
}
.charger1::after {
	right: 0;
}
.charger1 span {
	background-color: #fff;
	top: -0.7em;
	line-height: 1em;
	position: relative;
	padding: 0.5em;
}
</style>
