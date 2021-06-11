<template>
	<div class="site-visualization">
		<div class="d-flex justify-content-between">
			<span class="usage-label d-flex align-items-center">
				<fa-icon icon="plug" class="d-block me-1"></fa-icon>
				<span class="d-none d-sm-block me-1">Verbrauch</span>
				<span>{{ kw(usage) }}</span>
			</span>
			<span class="surplus-label d-flex align-items-center" v-if="batteryConfigured">
				<fa-icon icon="battery-three-quarters" class="d-block me-1"></fa-icon>
				<span class="d-none d-sm-block me-1">Batterie</span>
				<span class="d-block me-1">{{ batterySoC }}%</span>
				<fa-icon
					icon="chevron-right"
					class="arrow"
					:class="{
						'arrow-up': batteryCharge,
						'arrow-down': batteryUsage,
					}"
				></fa-icon>
			</span>
			<span class="surplus-label d-flex align-items-center" v-if="pvConfigured">
				<fa-icon icon="sun" class="d-block me-1"></fa-icon>
				<span class="d-none d-sm-block me-1">Produktion</span>
				{{ kw(pvPower) }}
			</span>
		</div>
		<div class="site-progress">
			<div class="site-progress-bar usage" :style="{ width: widthTotal(usage) }">
				<div class="site-progress-bar grid-usage" :style="{ width: widthUsage(gridUsage) }">
					<span class="power" :class="{ 'd-none': hidePowerLabel(gridUsage) }">
						{{ kw(gridUsage) }}
					</span>
				</div>
				<div class="site-progress-bar pv-usage" :style="{ width: widthUsage(pvUsage) }">
					<span class="power" :class="{ 'd-none': hidePowerLabel(pvUsage) }">
						{{ kw(pvUsage) }}
					</span>
				</div>
				<div
					class="site-progress-bar battery-usage"
					:style="{ width: widthUsage(batteryUsage) }"
				>
					<span class="power" :class="{ 'd-none': hidePowerLabel(batteryUsage) }">
						{{ kw(batteryUsage) }}
					</span>
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
					<span class="power" :class="{ 'd-none': hidePowerLabel(batteryCharge) }">
						{{ kw(batteryCharge) }}
					</span>
				</div>
				<div class="site-progress-bar pv-export" :style="{ width: widthSurplus(pvExport) }">
					<span class="power" :class="{ 'd-none': hidePowerLabel(pvExport) }">
						{{ kw(pvExport) }}
					</span>
				</div>
			</div>
		</div>
		<div class="d-flex justify-content-between">
			<span class="grid-usage-label" v-if="showLabel(gridUsage)">Netzbezug</span>
			<span class="pv-usage-label" v-if="showLabel(pvUsage)">Direktverbrauch</span>
			<span class="battery-label" v-if="showLabel(batteryUsage) || showLabel(batteryCharge)"
				>Batterie</span
			>
			<span class="pv-export-label" v-if="showLabel(pvExport)">Einspeisung</span>
		</div>
	</div>
</template>

<script>
import "../icons";
import formatter from "../mixins/formatter";

export default {
	name: "SiteVisualization",
	props: {
		gridConfigured: Boolean,
		gridPower: { type: Number, default: 0 },
		pvConfigured: Boolean,
		pvPower: { type: Number, default: 0 },
		batteryConfigured: Boolean,
		batteryPower: { type: Number, default: 0 },
		batterySoC: { type: Number, default: 0 },
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
			return Math.min(0, this.gridPower) * -1;
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
		hidePowerLabel(power) {
			return (100 / this.total) * power < 18;
		},
		showLabel(power) {
			const threshold = 50;
			return power > threshold;
		},
	},
};
</script>
<style scoped>
.site-visualization {
	--evcc-grid: #000033;
	--evcc-pv-usage: #66d85a;
	--evcc-battery: #006600;
	--evcc-pv-export: #ffff00;
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
}
.usage {
	border-radius: 5px;
	display: flex;
}
.grid-usage {
	background-color: var(--evcc-grid);
	color: var(--bs-light);
}
.pv-usage {
	background-color: var(--evcc-pv-usage);
}
.surplus {
	border-radius: 5px;
	display: flex;
	margin-left: 10px;
}
.battery-usage,
.battery-charge {
	background-color: var(--evcc-battery);
	color: var(--bs-light);
}
.pv-export {
	background-color: var(--evcc-pv-export);
	color: var(--bs-dark);
}
.pv-export-label,
.pv-usage-label,
.grid-usage-label,
.battery-label {
	color: var(--bs-gray-dark);
	text-decoration-line: underline;
	text-decoration-skip-ink: auto;
	text-decoration-thickness: 2px;
	text-decoration-style: solid;
	overflow: hidden;
	white-space: nowrap;
	text-overflow: ellipsis;
	display: block;
	transition-property: opacity, width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	font-size: 0.875em;
}
.power {
	display: block;
	margin: 0 0.5rem;
	white-space: nowrap;
}
.usage-label {
	text-decoration: none;
}
.pv-export-label {
	text-decoration-color: var(--evcc-pv-export);
}
.grid-usage-label {
	text-decoration-color: var(--evcc-grid);
}
.battery-label {
	text-decoration-color: var(--evcc-battery);
}
.pv-usage-label {
	text-decoration-color: var(--evcc-pv-usage);
}
.arrow {
	transition-property: opacity, transform;
	transition-duration: 500ms;
	transition-timing-function: ease-in;
	opacity: 0;
	transform: rotate(0);
}
.arrow-up {
	opacity: 1;
	transform: rotate(-90deg);
}
.arrow-down {
	opacity: 1;
	transform: rotate(90deg);
}
</style>
