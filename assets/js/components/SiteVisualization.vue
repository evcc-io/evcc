<template>
	<div class="site-visualization py-4" :class="{ 'show-values': showValues }">
		<div class="label-scale">
			<div class="d-flex justify-content-start">
				<div
					class="label-bar label-bar--down"
					v-if="usage"
					:style="{ width: widthTotal(usage) }"
					v-tooltip="{
						trigger: 'manual',
						content: $t('main.siteVisualization.consumption') + ': ' + kw(usage),
						placement: 'top',
						show: tooltip === 'usage',
					}"
					@click="toggleTooltip('usage')"
				>
					<div class="label-bar-scale">
						<div class="label-bar-icon">
							<fa-icon icon="plug"></fa-icon>
						</div>
					</div>
				</div>
				<div
					class="label-bar label-bar--down"
					v-if="batteryCharge"
					:style="{ width: widthTotal(batteryCharge) }"
					v-tooltip="{
						trigger: 'manual',
						content:
							$t('main.siteVisualization.batteryCharge') + ': ' + kw(batteryCharge),
						placement: 'top',
						show: tooltip === 'batteryCharge',
					}"
					@click="toggleTooltip('batteryCharge')"
				>
					<div class="label-bar-scale">
						<div class="label-bar-icon"><fa-icon :icon="batteryIcon"></fa-icon> ↑</div>
					</div>
				</div>
			</div>
		</div>
		<div class="site-progress" @click="toggleValues()">
			<div class="site-progress-bar grid-usage" :style="{ width: widthTotal(gridUsage) }">
				<span class="power" :class="{ 'd-none': hidePowerLabel(gridUsage) }">
					{{ kw(gridUsage) }}
				</span>
			</div>
			<div class="site-progress-bar self-usage" :style="{ width: widthTotal(batteryUsage) }">
				<span class="power" :class="{ 'd-none': hidePowerLabel(batteryUsage) }">
					{{ kw(batteryUsage) }}
				</span>
			</div>
			<div class="site-progress-bar self-usage" :style="{ width: widthTotal(pvUsage) }">
				<span class="power" :class="{ 'd-none': hidePowerLabel(pvUsage) }">
					{{ kw(pvUsage) }}
				</span>
			</div>
			<div class="site-progress-bar self-usage" :style="{ width: widthTotal(batteryCharge) }">
				<span class="power" :class="{ 'd-none': hidePowerLabel(batteryCharge) }">
					{{ kw(batteryCharge) }}
				</span>
			</div>
			<div class="site-progress-bar surplus" :style="{ width: widthTotal(pvExport) }">
				<span class="power" :class="{ 'd-none': hidePowerLabel(pvExport) }">
					{{ kw(pvExport) }}
				</span>
			</div>
		</div>
		<div class="label-scale">
			<div class="d-flex justify-content-end">
				<div
					class="label-bar label-bar--up"
					v-if="batteryUsage"
					:style="{ width: widthTotal(batteryUsage) }"
					v-tooltip="{
						trigger: 'manual',
						content:
							$t('main.siteVisualization.batteryDischarge') + ': ' + kw(batteryUsage),
						placement: 'bottom',
						show: tooltip === 'batteryUsage',
					}"
					@click="toggleTooltip('batteryUsage')"
				>
					<div class="label-bar-scale label-bar-scale--up">
						<div class="label-bar-icon"><fa-icon :icon="batteryIcon"></fa-icon> ↓</div>
					</div>
				</div>
				<div
					class="label-bar label-bar--up"
					v-if="pvPower"
					:style="{ width: widthTotal(pvPower) }"
					v-tooltip="{
						trigger: 'manual',
						content: $t('main.siteVisualization.pv') + ': ' + kw(pvPower),
						placement: 'bottom',
						show: tooltip === 'pvPower',
					}"
					@click="toggleTooltip('pvPower')"
				>
					<div class="label-bar-scale label-bar-scale--up">
						<div class="label-bar-icon">
							<fa-icon icon="sun"></fa-icon>
						</div>
					</div>
				</div>
			</div>
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
	},
	data: function () {
		return { showValues: false, tooltip: null };
	},
	mixins: [formatter],
	computed: {
		rawTotal: function () {
			return Math.max(0, this.batteryPower) + Math.max(0, this.gridPower) + this.pvPower;
		},
		gridUsage: function () {
			return this.applyThreshold(Math.max(0, this.gridPower));
		},
		pvUsage: function () {
			return this.applyThreshold(
				Math.min(this.pvPower, this.pvPower + this.gridPower - this.batteryCharge)
			);
		},
		batteryUsage: function () {
			return this.applyThreshold(Math.max(0, this.batteryPower));
		},
		usage: function () {
			return this.gridUsage + this.pvUsage + this.batteryUsage;
		},
		pvExport: function () {
			return this.applyThreshold(Math.min(0, this.gridPower) * -1);
		},
		batteryCharge: function () {
			return this.applyThreshold(Math.min(0, this.batteryPower) * -1);
		},
		selfUsage: function () {
			return this.pvUsage + this.batteryUsage;
		},
		surplus: function () {
			return this.pvExport + this.batteryCharge;
		},
		total: function () {
			return this.usage + this.surplus;
		},
		batteryIcon: function () {
			if (this.batterySoC > 80) return "battery-full";
			if (this.batterySoC > 60) return "battery-three-quarters";
			if (this.batterySoC > 40) return "battery-half";
			if (this.batterySoC > 20) return "battery-quarter";
			return "battery-empty";
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
			return Math.max(0, (watt / 1000).toFixed(1)) + " kW";
		},
		hidePowerLabel(power) {
			return (100 / this.total) * power < 18;
		},
		applyThreshold(power) {
			// adjust all values under 5% of the total power mix to 0
			return (100 / this.rawTotal) * power < 5 ? 0 : power;
		},
		toggleTooltip(name) {
			if (this.tooltip === name) {
				this.tooltip = null;
			} else {
				this.tooltip = name;
				this.showValues = true;
			}
		},
		toggleValues() {
			this.showValues = !this.showValues;
			this.tooltip = null;
		},
	},
};
</script>
<style scoped>
.site-visualization {
	--evcc-grid: #000033;
	--evcc-self: #66d85a;
	--evcc-surplus: #ffe000;
}
.site-progress {
	margin: 0.75rem 0;
	border-radius: 5px;
	display: flex;
	overflow: hidden;
	cursor: pointer;
}
.site-progress-bar {
	display: flex;
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	justify-content: center;
	align-items: center;
	overflow: hidden;
	height: 1.5rem;
	position: relative;
}
.site-progress-bar::before,
.site-progress-bar::after {
	content: "";
	top: 0;
	bottom: 0;
	width: 1px;
	background: white;
	position: absolute;
}
.site-progress-bar::before {
	left: 0px;
}
.site-progress-bar::after {
	right: 0px;
}
.grid-usage {
	background-color: var(--evcc-grid);
	color: var(--bs-light);
}
.self-usage {
	background-color: var(--evcc-self);
	color: var(--bs-light);
}
.surplus {
	background-color: var(--evcc-surplus);
	color: var(--bs-gray);
}

.power {
	display: block;
	margin: 0 0.5rem;
	white-space: nowrap;
	opacity: 0;
	transition: opacity 100ms ease;
}

.show-values .power {
	opacity: 1;
}

.usage-label,
.production-label {
	flex-basis: 33%;
}
.battery-label {
	position: relative;
}
.usage-label {
	justify-content: flex-start;
}
.production-label {
	justify-content: flex-end;
}
.arrow-up,
.arrow-down {
	position: absolute;
	left: 0.35rem;
	width: 0.5rem;
}
.arrow-up {
	top: -0.3rem;
}
.arrow-down {
	bottom: -0.3rem;
}
.label-bar {
	margin: 0;
	height: 1.25rem;
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
	cursor: pointer;
}
.label-bar--down {
	padding-top: 0.75rem;
}
.label-bar--up {
	padding-bottom: 0.75rem;
}
.label-bar-scale {
	border: 1px solid var(--bs-gray);
	height: 7px;
	background: none;
	display: flex;
	justify-content: center;
	align-items: center;
	white-space: nowrap;
}
.label-bar--down .label-bar-scale {
	border-bottom: 5px solid transparent;
}
.label-bar--up .label-bar-scale {
	border-top: 5px solid transparent;
}
.label-bar-icon {
	background-color: white;
	color: var(--bs-gray);
	padding: 0 0.75rem;
}
</style>
