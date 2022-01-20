<template>
	<div class="visualization" :class="{ 'visualization--ready': visualizationReady }">
		<div class="label-scale">
			<div class="d-flex justify-content-start">
				<LabelBar v-bind="labelBarProps('top', 'pvProduction')">
					<shopicon-regular-sun></shopicon-regular-sun>
				</LabelBar>
				<LabelBar v-bind="labelBarProps('top', 'batteryDischarge')">
					<BatteryIcon :soc="batterySoC" />
				</LabelBar>
				<LabelBar v-bind="labelBarProps('top', 'gridImport')">
					<shopicon-regular-powersupply></shopicon-regular-powersupply>
				</LabelBar>
			</div>
		</div>
		<div ref="site_progress" class="site-progress">
			<div
				class="site-progress-bar self-consumption"
				:style="{ width: widthTotal(selfConsumptionAdjusted) }"
			>
				<span v-if="powerLabelEnoughSpace(selfConsumption)" class="power">
					{{ kw(selfConsumption) }}
				</span>
				<span v-else-if="powerLabelSomeSpace(selfConsumption)" class="power">
					{{ kwNoUnit(selfConsumption) }}
				</span>
			</div>
			<div
				class="site-progress-bar grid-import"
				:style="{ width: widthTotal(gridImportAdjusted) }"
			>
				<span v-if="powerLabelEnoughSpace(gridImport)" class="power">
					{{ kw(gridImport) }}
				</span>
				<span v-else-if="powerLabelSomeSpace(gridImport)" class="power">
					{{ kwNoUnit(gridImport) }}
				</span>
			</div>
			<div
				class="site-progress-bar pv-export"
				:style="{ width: widthTotal(pvExportAdjusted) }"
			>
				<span v-if="powerLabelEnoughSpace(pvExport)" class="power">
					{{ kw(pvExport) }}
				</span>
				<span v-else-if="powerLabelSomeSpace(pvExport)" class="power">
					{{ kwNoUnit(pvExport) }}
				</span>
			</div>
			<div v-if="totalAdjusted <= 0" class="site-progress-bar bg-light border no-wrap w-100">
				<span>{{ $t("main.energyflow.noEnergy") }}</span>
			</div>
		</div>
		<div class="label-scale">
			<div class="d-flex justify-content-start">
				<LabelBar v-bind="labelBarProps('bottom', 'homePower')">
					<shopicon-regular-home></shopicon-regular-home>
				</LabelBar>
				<LabelBar v-bind="labelBarProps('bottom', 'loadpoints')">
					<shopicon-regular-car3></shopicon-regular-car3>
				</LabelBar>
				<LabelBar v-bind="labelBarProps('bottom', 'batteryCharge')">
					<BatteryIcon :soc="batterySoC" />
				</LabelBar>
				<LabelBar v-bind="labelBarProps('bottom', 'gridExport')">
					<shopicon-regular-powersupply></shopicon-regular-powersupply>
				</LabelBar>
			</div>
		</div>
	</div>
</template>

<script>
import "../../icons";
import formatter from "../../mixins/formatter";
import BatteryIcon from "./BatteryIcon.vue";
import LabelBar from "./LabelBar.vue";
import "@h2d2/shopicons/es/regular/car3";
import "@h2d2/shopicons/es/regular/sun";
import "@h2d2/shopicons/es/regular/home";

export default {
	name: "Visualization",
	components: { BatteryIcon, LabelBar },
	mixins: [formatter],
	props: {
		gridImport: { type: Number, default: 0 },
		selfConsumption: { type: Number, default: 0 },
		pvExport: { type: Number, default: 0 },
		loadpoints: { type: Number, default: 0 },
		batteryCharge: { type: Number, default: 0 },
		batteryDischarge: { type: Number, default: 0 },
		pvProduction: { type: Number, default: 0 },
		homePower: { type: Number, default: 0 },
		batterySoC: { type: Number, default: 0 },
		valuesInKw: { type: Boolean, default: false },
	},
	data: function () {
		return { width: 0, visualizationReady: false };
	},
	computed: {
		gridExport: function () {
			return this.applyThreshold(this.pvExport);
		},
		totalRaw: function () {
			return this.gridImport + this.selfConsumption + this.pvExport;
		},
		gridImportAdjusted: function () {
			return this.applyThreshold(this.gridImport);
		},
		selfConsumptionAdjusted: function () {
			return this.applyThreshold(this.selfConsumption);
		},
		pvExportAdjusted: function () {
			return this.applyThreshold(this.pvExport);
		},
		totalAdjusted: function () {
			return this.gridImportAdjusted + this.selfConsumptionAdjusted + this.pvExportAdjusted;
		},
	},
	watch: {
		totalAdjusted: function () {
			if (!this.visualizationReady && this.totalAdjusted > 0)
				setTimeout(() => {
					this.visualizationReady = true;
				}, 500);
		},
	},
	mounted: function () {
		this.$nextTick(function () {
			window.addEventListener("resize", this.updateElementWidth);
			this.updateElementWidth();
		});
	},
	beforeDestroy() {
		window.removeEventListener("resize", this.updateElementWidth);
	},
	methods: {
		widthTotal: function (power) {
			if (this.totalAdjusted === 0) return "0%";
			return (100 / this.totalAdjusted) * power + "%";
		},
		kw: function (watt) {
			return this.fmtKw(watt, this.valuesInKw, true);
		},
		kwNoUnit: function (watt) {
			return this.fmtKw(watt, this.valuesInKw, false);
		},
		powerLabelAvailableSpace(power) {
			if (this.totalAdjusted === 0) return 0;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent;
		},
		powerLabelEnoughSpace(power) {
			return this.powerLabelAvailableSpace(power) > 60;
		},
		powerLabelSomeSpace(power) {
			return this.powerLabelAvailableSpace(power) > 35;
		},
		hideLabelIcon(power, minWidth = 32) {
			if (this.totalAdjusted === 0) return true;
			const percent = (100 / this.totalAdjusted) * power;
			return (this.width / 100) * percent < minWidth;
		},
		applyThreshold(power) {
			const percent = (100 / this.totalRaw) * power;
			return percent < 2 ? 0 : power;
		},
		updateElementWidth() {
			this.width = this.$refs.site_progress.getBoundingClientRect().width;
		},
		labelBarProps(position, name) {
			const value = this[name];
			const minWidth = 40;
			return {
				value,
				hideIcon: this.hideLabelIcon(value, minWidth),
				style: { width: this.widthTotal(value) },
				[position]: true,
			};
		},
	},
};
</script>
<style scoped>
.site-progress {
	--height: 32px;
	height: var(--height);
	border-radius: 10px;
	display: flex;
	overflow: hidden;
}
.site-progress-bar {
	display: flex;
	justify-content: center;
	align-items: center;
	overflow: hidden;
	position: relative;
	font-size: 0.75rem;
	width: 0;
}
.visualization--ready .site-progress-bar {
	transition-property: width;
	transition-duration: 500ms;
	transition-timing-function: linear;
}
.grid-import {
	background-color: var(--evcc-grid);
	color: var(--bs-white);
}
.self-consumption {
	background-color: var(--evcc-self);
	color: var(--bs-white);
}
.pv-export {
	background-color: var(--evcc-export);
	color: var(--bs-dark);
}
.power {
	display: block;
	margin: 0 0.2rem;
	white-space: nowrap;
	overflow: hidden;
}
.visualization--ready >>> .label-bar {
	transition-property: width, opacity;
	transition-duration: 500ms, 250ms;
	transition-timing-function: linear, ease;
}
.visualization--ready >>> .label-bar-icon {
	transition: opacity 250ms ease-in;
}
</style>
